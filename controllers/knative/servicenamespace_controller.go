package controllers

import (
	"context"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	knativev1 "knative.dev/serving/pkg/apis/serving/v1"
	v1 "open-cluster-management.io/api/work/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ServiceNamespaceReconciler reconciles a ServiceNamespace object
type ServiceNamespaceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	// NamespaceManifestWorkPrefix prefix of the manifest work creating a namespace on the managed cluster
	NamespaceManifestWorkPrefix = "mw-create-"
)

// ServiceNamespacePredicateFunctions defines which service this controller should wrap inside ManifestWork's payload
var ServiceNamespacePredicateFunctions = predicate.Funcs{
	UpdateFunc: func(e event.UpdateEvent) bool {
		newService := e.ObjectNew.(*knativev1.Service)
		return ContainsValidOCMAnnotation(*newService) && ContainsValidOCMNamespaceAnnotation(*newService)

	},
	CreateFunc: func(e event.CreateEvent) bool {
		service := e.Object.(*knativev1.Service)
		return ContainsValidOCMAnnotation(*service) && ContainsValidOCMNamespaceAnnotation(*service)
	},

	DeleteFunc: func(e event.DeleteEvent) bool {
		service := e.Object.(*knativev1.Service)
		return ContainsValidOCMAnnotation(*service) && ContainsValidOCMNamespaceAnnotation(*service)
	},
}

func (r *ServiceNamespaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	service := knativev1.Service{}
	if err := r.Client.Get(ctx, req.NamespacedName, &service); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	l.Info("Start reconciling Service in ServiceNamespace controller")

	if err := r.ensureNamespaceExistence(ctx, service); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceNamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&knativev1.Service{}).
		WithEventFilter(ServiceNamespacePredicateFunctions).
		Complete(r)
}

// ensureNamespaceExistence gets context and service
// The function ensures the manifest work containing the namespace exists, if it doesn't it creates it
// Returns an error if occured
func (r *ServiceNamespaceReconciler) ensureNamespaceExistence(ctx context.Context, service knativev1.Service) error {
	namespaceName := service.GetAnnotations()[AnnotationKeyOCMManagedClusterNamespace]
	mwName := NamespaceManifestWorkPrefix + namespaceName
	mwNamespace := service.GetAnnotations()[AnnotationKeyOCMManagedCluster]
	ok, err := r.checkManifestWorkExistence(ctx, mwName, mwNamespace)
	if err != nil {
		return err
	}
	if !ok {
		ns := GenerateNamespace(namespaceName)
		w := GenerateManifestWorkGeneric(mwName, mwNamespace, &ns)
		if err := r.Client.Create(ctx, w); err != nil {
			return err
		}
	}
	if err := r.addNamespaceCreatedAnnotation(ctx, &service); err != nil {
		return err
	}
	return nil
}

// checkManifestWorkExistence gets context, manifest work name and manifest work namespace
// The function returns true whether the manifest work already exist and false otherwise
func (r *ServiceNamespaceReconciler) checkManifestWorkExistence(ctx context.Context, mwName string, mwNamespace string) (bool, error) {
	mw := v1.ManifestWork{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: mwName, Namespace: mwNamespace}, &mw); err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		} else {
			return false, err
		}
	}
	return true, nil
}

// addNamespaceCreatedAnnotation get context and service
// The function add annotation to the service that namespace created
// Return error if occured
func (r *ServiceNamespaceReconciler) addNamespaceCreatedAnnotation(ctx context.Context, service *knativev1.Service) error {
	svcAnno := service.GetAnnotations()
	svcAnno[AnnotationNamespaceCreated] = "true"
	service.SetAnnotations(svcAnno)
	if err := r.Client.Update(ctx, service); err != nil {
		return err
	}
	return nil
}
