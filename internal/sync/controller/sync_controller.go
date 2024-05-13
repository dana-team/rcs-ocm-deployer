package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/dana-team/rcs-ocm-deployer/internal/sync/adapters"

	cappv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
	director "github.com/dana-team/rcs-ocm-deployer/internal/sync/directors"
	"github.com/dana-team/rcs-ocm-deployer/internal/utils"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	workv1 "open-cluster-management.io/api/work/v1"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const RequeueTime = 2 * time.Second

// SyncReconciler reconciles a CappNamespace object
type SyncReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	EventRecorder record.EventRecorder
}

//+kubebuilder:rbac:groups=rcs.dana.io,resources=capps/status,verbs=update
//+kubebuilder:rbac:groups=work.open-cluster-management.io,resources=manifestworks,verbs=get;list;watch;create;patch;update;delete
//+kubebuilder:rbac:groups="rcs.dana.io",resources=rcsconfigs,verbs=get;list;watch
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=rolebindings,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

func (r *SyncReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("CappName", req.Name, "CappNamespace", req.Namespace)
	capp := cappv1alpha1.Capp{}
	if err := r.Client.Get(ctx, req.NamespacedName, &capp); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	if capp.ObjectMeta.DeletionTimestamp != nil {
		if err := adapters.HandleCappDeletion(ctx, capp, logger, r.Client); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: RequeueTime}, nil
	}
	if err := adapters.EnsureFinalizer(ctx, capp, r.Client); err != nil {
		return ctrl.Result{}, err
	}
	return r.SyncManifestWork(capp, ctx, logger)
}

var CappPredicateFuncs = predicate.Funcs{
	UpdateFunc: func(e event.UpdateEvent) bool {
		newCapp := e.ObjectNew.(*cappv1alpha1.Capp)
		return utils.ContainsPlacementAnnotation(*newCapp)
	},
	CreateFunc: func(e event.CreateEvent) bool {
		capp := e.Object.(*cappv1alpha1.Capp)
		return utils.ContainsPlacementAnnotation(*capp)
	},

	DeleteFunc: func(e event.DeleteEvent) bool {
		capp := e.Object.(*cappv1alpha1.Capp)
		return utils.ContainsPlacementAnnotation(*capp)
	},
}

// SyncManifestWork checks whether the manifest work deploying the Capp exists in the managed cluster namespace
// If it does, it updates the Capp in the manifest work spec. If it doesn't then it creates it
func (r *SyncReconciler) SyncManifestWork(capp cappv1alpha1.Capp, ctx context.Context, logger logr.Logger) (ctrl.Result, error) {
	mwName := adapters.GenerateMWName(capp)
	managedClusterName := capp.Annotations[utils.AnnotationKeyHasPlacement]
	var mw workv1.ManifestWork
	cappDirector := director.CappDirector{Ctx: ctx, K8sclient: r.Client, Log: logger, EventRecorder: r.EventRecorder}
	manifests, err := cappDirector.AssembleManifests(capp)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to build ManifestWork: %v", err.Error())
	}

	if err := r.Get(ctx, types.NamespacedName{Name: mwName, Namespace: managedClusterName}, &mw); err != nil {
		if errors.IsNotFound(err) {
			err := adapters.CreateManifestWork(capp, managedClusterName, logger, r.Client, ctx, r.EventRecorder, manifests)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}
	mw.Spec.Workload.Manifests = manifests

	if err = r.Update(ctx, &mw); err != nil {
		if errors.IsConflict(err) {
			logger.Info("Conflict while updating ManifestWork trying again in a few seconds")
			return ctrl.Result{RequeueAfter: RequeueTime}, nil
		}
		return ctrl.Result{}, fmt.Errorf("failed to sync ManifestWork: %v", err.Error())
	}
	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *SyncReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cappv1alpha1.Capp{}).
		WithEventFilter(CappPredicateFuncs).
		Complete(r)
}
