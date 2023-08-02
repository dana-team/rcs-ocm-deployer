package controllers

import (
	"context"
	"fmt"
	"time"

	rcsv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
	"github.com/dana-team/rcs-ocm-deployer/internals/utils"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/strings/slices"

	"k8s.io/apimachinery/pkg/runtime"
	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// ServicePlacementReconciler reconciles a ServicePlacement object
type ServicePlacementReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	EventRecorder       record.EventRecorder
	Placements          []string
	PlacementsNamespace string
}

//+kubebuilder:rbac:groups=rcs.dana.io,resources=capps,verbs=get;list;watch
//+kubebuilder:rbac:groups=cluster.open-cluster-management.io,resources=placementdecisions,verbs=get;list;watch
//+kubebuilder:rbac:groups=cluster.open-cluster-management.io,resources=placements,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *ServicePlacementReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("CappName", req.Name, "CappNamespace", req.Namespace)
	capp := rcsv1alpha1.Capp{}
	if err := r.Client.Get(ctx, req.NamespacedName, &capp); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	placementRef := capp.Spec.Site
	if placementRef == "" || slices.Contains(r.Placements, placementRef) {
		cluster, err := r.pickDecision(capp, logger, ctx)
		if err != nil {
			logger.Error(err, fmt.Sprintf("Failed to pick managed cluster for placement %s.", placementRef))
			return ctrl.Result{}, err
		}
		if cluster == "requeue" {
			logger.Info(fmt.Sprintf("requeuing capp %s, waiting for PlacementDecision to be satisfied.", capp.Name))
			r.EventRecorder.Event(&capp, "Warning", "PlacementDecisionNotSatisfied", fmt.Sprintf("Failed to schedule capp %s to managed cluster. PlacementDecision with optional clusters was not found for placement %s.", capp.Name, placementRef))
			return ctrl.Result{RequeueAfter: 10 * time.Second * 2}, nil
		}
		placementRef = cluster
	}
	if err := utils.UpdateCappDestination(capp, placementRef, ctx, r.Client); err != nil {
		return ctrl.Result{}, fmt.Errorf("Unable to update capp with selected cluster %s", err.Error())
	}
	r.EventRecorder.Event(&capp, "Normal", "CappScheduled", fmt.Sprintf("Scheduled Capp %s for managed cluster %s", capp.Name, placementRef))
	return ctrl.Result{}, nil
}

var ServicePredicateFunctions = predicate.Funcs{
	UpdateFunc: func(e event.UpdateEvent) bool {
		newCapp := e.ObjectNew.(*rcsv1alpha1.Capp)
		return !utils.ContainesPlacementAnnotation(*newCapp)

	},
	CreateFunc: func(e event.CreateEvent) bool {
		capp := e.Object.(*rcsv1alpha1.Capp)
		return !utils.ContainesPlacementAnnotation(*capp)
	},

	DeleteFunc: func(e event.DeleteEvent) bool {
		capp := e.Object.(*rcsv1alpha1.Capp)
		return !utils.ContainesPlacementAnnotation(*capp)
	},
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServicePlacementReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&rcsv1alpha1.Capp{}).
		WithEventFilter(ServicePredicateFunctions).
		Complete(r)
}

// pickDecision gets a service logger and context
// The function decides the name of the managed cluster to deploy to
// And adds an annotation to the capp with its name
// Returns controller result and error

func (r *ServicePlacementReconciler) pickDecision(capp rcsv1alpha1.Capp, log logr.Logger, ctx context.Context) (string, error) {
	placementRef := capp.Spec.Site
	if capp.Spec.Site == "" {
		placementRef = r.Placements[0]
	}
	placement := clusterv1beta1.Placement{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: placementRef, Namespace: r.PlacementsNamespace}, &placement); err != nil {
		return "", fmt.Errorf("Failed to get placement %s", err.Error())
	}
	placementDecisions, err := utils.GetPlacementDecisionList(capp, log, ctx, placementRef, r.PlacementsNamespace, r.Client)
	if len(placementDecisions.Items) == 0 {
		return "requeue", nil
	}
	if err != nil {
		return "", fmt.Errorf("Failed to list PlacementDecisions %s", err.Error())
	}
	managedClusterName := utils.GetDecisionClusterName(placementDecisions, log)
	if managedClusterName == "" {
		return "requeue", nil
	}
	return managedClusterName, nil
}
