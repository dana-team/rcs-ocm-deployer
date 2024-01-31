package controllers

import (
	"context"
	"fmt"
	"time"

	rcsv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
	"github.com/dana-team/rcs-ocm-deployer/internals/utils"
	statusutils "github.com/dana-team/rcs-ocm-deployer/internals/utils/status"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	workv1 "open-cluster-management.io/api/work/v1"

	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// CappNamespaceReconciler reconciles a CappNamespace object
type CappNamespaceReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	EventRecorder record.EventRecorder
}

const (
	// NamespaceManifestWorkPrefix prefix of the manifest work creating a namespace on the managed cluster
	FinalizerCleanupManifestWork = "dana.io/cleanup-ocm-manifestwork"
	NamespaceManifestWorkPrefix  = "mw-create-"
)

//+kubebuilder:rbac:groups=rcs.dana.io,resources=capps/status,verbs=update
//+kubebuilder:rbac:groups=work.open-cluster-management.io,resources=manifestworks,verbs=get;list;watch;create;patch;update;delete
//+kubebuilder:rbac:groups="rcsd.dana.io",resources=rcsconfigs,verbs=get;list;watch
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=rolebindings,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *CappNamespaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("CappName", req.Name, "CappNamespace", req.Namespace)
	capp := rcsv1alpha1.Capp{}
	if err := r.Client.Get(ctx, req.NamespacedName, &capp); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	if capp.ObjectMeta.DeletionTimestamp != nil {
		if err := utils.HandleResourceDeletion(ctx, capp, logger, r.Client); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
	}
	if err := utils.EnsureFinalizer(ctx, capp, r.Client); err != nil {
		return ctrl.Result{}, err
	}
	return r.SyncManifestWork(capp, ctx, logger)
}

var CappPredicateFuncs = predicate.Funcs{
	UpdateFunc: func(e event.UpdateEvent) bool {
		newCapp := e.ObjectNew.(*rcsv1alpha1.Capp)
		return utils.ContainsPlacementAnnotation(*newCapp)

	},
	CreateFunc: func(e event.CreateEvent) bool {
		capp := e.Object.(*rcsv1alpha1.Capp)
		return utils.ContainsPlacementAnnotation(*capp)
	},

	DeleteFunc: func(e event.DeleteEvent) bool {
		capp := e.Object.(*rcsv1alpha1.Capp)
		return utils.ContainsPlacementAnnotation(*capp)
	},
}

// SyncManifestWork checks whether the manifest work deploying the Capp exists in the managed cluster namespace
// If it does, it updates the Capp in the manifest work spec. If it doesn't then it creates it
func (r *CappNamespaceReconciler) SyncManifestWork(capp rcsv1alpha1.Capp, ctx context.Context, logger logr.Logger) (ctrl.Result, error) {
	mwName := utils.NamespaceManifestWorkPrefix + capp.Namespace + "-" + capp.Name
	managedClusterName := capp.Annotations[utils.AnnotationKeyHasPlacement]
	var mw workv1.ManifestWork
	manifests, err := utils.GatherCappResources(capp, ctx, logger, r.Client)
	if err != nil {
		r.EventRecorder.Event(&capp, eventTypeError, eventCappVolumeNotFound, err.Error())
		if err := statusutils.SetVolumesCondition(capp, ctx, r.Client, logger, false, err.Error()); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to set volume condition: %s", err.Error())
		}
		return ctrl.Result{}, fmt.Errorf("failed to get one of the volumes from Capp spec: %s", err.Error())
	}

	if err := r.Get(ctx, types.NamespacedName{Name: mwName, Namespace: managedClusterName}, &mw); err != nil {
		if errors.IsNotFound(err) {
			mw := utils.GenerateManifestWorkGeneric(mwName, managedClusterName, manifests, workv1.ManifestConfigOption{})
			utils.SetManifestWorkCappAnnotations(*mw, capp)
			if err := r.Create(ctx, mw); err != nil {
				r.EventRecorder.Event(&capp, eventTypeError, eventCappManifestWorkCreationFailed, err.Error())
				return ctrl.Result{}, fmt.Errorf("failed to create ManifestWork: %s", err.Error())
			}
			logger.Info(fmt.Sprintf("Created ManifestWork %s for Capp %s", mwName, capp.Name))
			r.EventRecorder.Event(&capp, eventTypeNormal, eventCappManifestWorkCreated, fmt.Sprintf("Created ManifestWork %s for capp %s", mwName, capp.Name))
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	mw.Spec.Workload.Manifests = manifests

	if err = r.Update(ctx, &mw); err != nil {
		if errors.IsConflict(err) {
			logger.Info("Conflict while updating ManifestWork trying again in a few seconds")
			return ctrl.Result{RequeueAfter: time.Second * 2}, nil
		}
		return ctrl.Result{}, fmt.Errorf("failed to sync ManifestWork: %s", err.Error())
	}
	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *CappNamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&rcsv1alpha1.Capp{}).
		WithEventFilter(CappPredicateFuncs).
		Complete(r)
}
