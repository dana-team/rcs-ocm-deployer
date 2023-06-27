package controllers

import (
	"context"
	"time"

	rcsv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
	utils "github.com/dana-team/rcs-ocm-deployer/internals/utils"
	status_utils "github.com/dana-team/rcs-ocm-deployer/internals/utils/status"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	workv1 "open-cluster-management.io/api/work/v1"

	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// ServiceNamespaceReconciler reconciles a ServiceNamespace object
type ServiceNamespaceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	// NamespaceManifestWorkPrefix prefix of the manifest work creating a namespace on the managed cluster
	FinalizerCleanupManifestWork = "dana.io/cleanup-ocm-manifestwork"
	NamespaceManifestWorkPrefix  = "mw-create-"
)

func (r *ServiceNamespaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	capp := rcsv1alpha1.Capp{}
	if err := r.Client.Get(ctx, req.NamespacedName, &capp); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	l.Info("Start reconciling Service in ServiceNamespace controller")
	if err, deleted := utils.HandleResourceDeletion(ctx, capp, l, r.Client); err != nil {
		if deleted {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	if err := utils.EnsureFinalizer(ctx, capp, r.Client); err != nil {
		return ctrl.Result{}, err
	}
	return r.SyncManifestWork(capp, ctx, l)
}

var CappPredicateFuncs = predicate.Funcs{
	UpdateFunc: func(e event.UpdateEvent) bool {
		newCapp := e.ObjectNew.(*rcsv1alpha1.Capp)
		return utils.ContainesPlacementAnnotation(*newCapp)

	},
	CreateFunc: func(e event.CreateEvent) bool {
		capp := e.Object.(*rcsv1alpha1.Capp)
		return utils.ContainesPlacementAnnotation(*capp)
	},

	DeleteFunc: func(e event.DeleteEvent) bool {
		capp := e.Object.(*rcsv1alpha1.Capp)
		return utils.ContainesPlacementAnnotation(*capp)
	},
}

// EnsureManifestWork checks whether the manifest work deploying the service exists in the managed cluster namespace
// If it does, it updates the service in the manifest work spec, if it doesn't, it creates it
func (r *ServiceNamespaceReconciler) SyncManifestWork(capp rcsv1alpha1.Capp, ctx context.Context, l logr.Logger) (ctrl.Result, error) {
	mwName := utils.NamespaceManifestWorkPrefix + capp.Namespace + "-" + capp.Name
	managedClusterName := capp.Annotations[utils.AnnotationKeyHasPlacement]
	var mw workv1.ManifestWork
	manifests, err := utils.GatherCappResources(capp, ctx, l, r.Client)
	if err != nil {
		status_utils.SetVolumesCondition(capp, ctx, r.Client, l, false, err.Error())
		return ctrl.Result{}, err
	}

	if err := r.Get(ctx, types.NamespacedName{Name: mwName, Namespace: managedClusterName}, &mw); err != nil {
		if errors.IsNotFound(err) {
			mw := utils.GenerateManifestWorkGeneric(mwName, managedClusterName, manifests, workv1.ManifestConfigOption{})
			utils.SetManifestWorkCappAnnotations(*mw, capp)
			if err := r.Create(ctx, mw); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	mw.Spec.Workload.Manifests = manifests

	if err = r.Update(ctx, &mw); err != nil {
		l.Error(err, "unable to update ManifestWork")
		if errors.IsConflict(err) {
			return ctrl.Result{RequeueAfter: time.Second * 2}, nil
		}
		return ctrl.Result{}, err
	}
	l.Info("done reconciling Workflow")
	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceNamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&rcsv1alpha1.Capp{}).
		WithEventFilter(CappPredicateFuncs).
		Complete(r)
}
