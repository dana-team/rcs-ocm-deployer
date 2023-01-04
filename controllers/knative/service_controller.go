/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	workv1 "open-cluster-management.io/api/work/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"k8s.io/apimachinery/pkg/runtime"
	knativev1 "knative.dev/serving/pkg/apis/serving/v1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// Service annotation that dictates which managed cluster this Workflow should be propgated to.
	AnnotationKeyOCMManagedCluster = "dana.io/ocm-managed-cluster"
	// Service annotation that dictates which managed cluster namespace this Workflow should be propgated to.
	AnnotationKeyOCMManagedClusterNamespace = "dana.io/ocm-managed-cluster-namespace"
	// ManifestWork annotation that shows the namespace of the hub Workflow.
	AnnotationKeyHubServiceNamespace = "dana.io/ocm-hub-service-namespace"
	// ManifestWork annotation that shows the name of the hub Workflow.
	AnnotationKeyHubServiceName = "dana.io/ocm-hub-service-name"
	// Service annotation that shows the first 5 characters of the dormant hub cluster service
	AnnotationKeyHubServiceUID = "dana.io/ocm-hub-service-uid"
	// FinalizerCleanupManifestWork is added to the Workflow so the associated ManifestWork gets cleaned up after a Workflow deletion.
	FinalizerCleanupManifestWork = "dana.io/cleanup-ocm-manifestwork"
	// Knative Service annotation that indicated that the Service's namespace was created in the managed cluster
	AnnotationNamespaceCreated = "dana.io/namespace-created"
)

// ServiceReconciler reconciles a Service object
type ServiceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=serving.knative.dev,resources=services,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=cluster.open-cluster-management.io,resources=managedclusters,verbs=get;list;watch
//+kubebuilder:rbac:groups=work.open-cluster-management.io,resources=manifestworks,verbs=get;list;watch;create;update;patch;delete

// ServicePlacementPredicateFunctions defines which Workflow this controller should wrap inside ManifestWork's payload
var ServicePredicateFunctions = predicate.Funcs{
	UpdateFunc: func(e event.UpdateEvent) bool {
		newService := e.ObjectNew.(*knativev1.Service)
		return ContainsValidOCMAnnotation(*newService) && ContainsNamespaceCreated(*newService)

	},
	CreateFunc: func(e event.CreateEvent) bool {
		service := e.Object.(*knativev1.Service)
		return ContainsValidOCMAnnotation(*service) && ContainsNamespaceCreated(*service)
	},

	DeleteFunc: func(e event.DeleteEvent) bool {
		service := e.Object.(*knativev1.Service)
		return ContainsValidOCMAnnotation(*service) && ContainsNamespaceCreated(*service)
	},
}

func (r *ServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	service := knativev1.Service{}
	if err := r.Client.Get(ctx, req.NamespacedName, &service); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	managedClusterName := service.GetAnnotations()[AnnotationKeyOCMManagedCluster]
	mwName := GenerateManifestWorkName(service)

	// the Workflow is being deleted, find the ManifestWork and delete that as well
	if service.ObjectMeta.DeletionTimestamp != nil {
		// remove finalizer from Workflow but do not 'commit' yet
		if controllerutil.ContainsFinalizer(&service, FinalizerCleanupManifestWork) {
			if err := r.finalizeService(ctx, mwName, managedClusterName, l); err != nil {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(&service, FinalizerCleanupManifestWork)
			if err := r.Client.Update(ctx, &service); err != nil {
				return ctrl.Result{}, nil
			}
		}
	}

	if err := r.ensureFinalizer(ctx, service); err != nil {
		return ctrl.Result{}, err
	}

	// verify the ManagedCluster actually exists
	if err := r.verifyManagedClusterExistence(ctx, l, managedClusterName); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.EnsureManifestWork(mwName, managedClusterName, service, ctx, l); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&knativev1.Service{}).
		WithEventFilter(ServicePredicateFunctions).
		Complete(r)
}

// finalizeService gets context, manifest work name, managed cluster name and logger
// The function checks whether the manifest work deploying the service exists
// If it does it deletes it
func (r *ServiceReconciler) finalizeService(ctx context.Context, mwName string, managedClusterName string, log logr.Logger) error {
	// delete the ManifestWork associated with this service
	var work workv1.ManifestWork
	if err := r.Get(ctx, types.NamespacedName{Name: mwName, Namespace: managedClusterName}, &work); err != nil {
		if errors.IsNotFound(err) {
			log.Info("already deleted ManifestWork, commit the Workflow finalizer removal")
			return nil
		} else {
			log.Error(err, "unable to fetch ManifestWork")
			return err
		}
	}
	if err := r.Delete(ctx, &work); err != nil {
		log.Error(err, "unable to delete ManifestWork")
		return err
	}
	return nil
}

// ensureFinalizer ensures the service has the finalizer
func (r *ServiceReconciler) ensureFinalizer(ctx context.Context, service knativev1.Service) error {
	if !controllerutil.ContainsFinalizer(&service, FinalizerCleanupManifestWork) {
		controllerutil.AddFinalizer(&service, FinalizerCleanupManifestWork)
		if err := r.Client.Update(ctx, &service); err != nil {
			return err
		}
	}
	return nil
}

// verifyManagedClusterExistence get managed cluster name and checks whether it exists or not
func (r *ServiceReconciler) verifyManagedClusterExistence(ctx context.Context, l logr.Logger, managedClusterName string) error {
	managedCluster := clusterv1.ManagedCluster{}
	if err := r.Get(ctx, types.NamespacedName{Name: managedClusterName}, &managedCluster); err != nil {
		l.Error(err, "unable to fetch ManagedCluster")
		return err
	}
	return nil
}

// EnsureManifestWork checks whether the manifest work deploying the service exists in the managed cluster namespace
// If it does, it updates the service in the manifest work spec, if it doesn't, it creates it
func (r *ServiceReconciler) EnsureManifestWork(mwName string, managedClusterName string, service knativev1.Service, ctx context.Context, l logr.Logger) error {
	l.Info("generating ManifestWork for Service")
	svc := PrepareServiceForWorkPayload(service)
	fr := GenerateFeedbackRule("knativeStatus", ".status")
	mco := GenerateManifestConfigOption(&svc, "services", knativev1.SchemeGroupVersion.Group, fr)
	w := GenerateManifestWorkGeneric(mwName, managedClusterName, &svc, mco)
	// create or update the ManifestWork depends if it already exists or not
	var mw workv1.ManifestWork
	err := r.Get(ctx, types.NamespacedName{Name: mwName, Namespace: managedClusterName}, &mw)
	if errors.IsNotFound(err) {
		if err := r.Client.Create(ctx, w); err != nil {
			l.Error(err, "unable to create ManifestWork")
			return err
		}
		l.Info("done reconciling Workflow")
		return nil
	}
	if err == nil {
		mw.Spec.Workload.Manifests = []workv1.Manifest{{RawExtension: runtime.RawExtension{Object: &svc}}}
		if err = r.Client.Update(ctx, &mw); err != nil {
			l.Error(err, "unable to update ManifestWork")
			return err
		}
		l.Info("done reconciling Workflow")
		return nil
	}
	l.Error(err, "unable to fetch ManifestWork")
	return err
}
