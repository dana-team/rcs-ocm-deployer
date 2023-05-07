package controllers

import (
	"context"

	rcsv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
	utils "github.com/dana-team/rcs-ocm-deployer/internals/utils"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
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
	Scheme     *runtime.Scheme
	Placements []string
}

//+kubebuilder:rbac:groups=cluster.open-cluster-management.io,resources=placementdecisions,verbs=get;list;watch
//+kubebuilder:rbac:groups=cluster.open-cluster-management.io,resources=placements,verbs=get;list;watch

func (r *ServicePlacementReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	capp := rcsv1alpha1.Capp{}
	if err := r.Client.Get(ctx, req.NamespacedName, &capp); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	placementRef := capp.Spec.Site
	if placementRef == "" || slices.Contains(r.Placements, placementRef) {
		cluster, err := r.pickDecision(capp, l, ctx)
		if err != nil {
			return ctrl.Result{}, err
		}
		if cluster == "requeue" {
			// FIXME - rgolangh - the 10 here is actually 10 nanosecond. If the intention is seconds for example
			// then use '10 * time.Seconds' (have a look at time.Duration constants)
			return ctrl.Result{RequeueAfter: 10}, nil
		}
		placementRef = cluster
	}
	if err := utils.UpdateCappDestination(capp, placementRef, ctx, r.Client); err != nil {
		l.Error(err, "unable to update capp")
		return ctrl.Result{}, err
	}
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
// and adds an annotation to the service with its name
// Returns controller result and error

func (r *ServicePlacementReconciler) pickDecision(capp rcsv1alpha1.Capp, log logr.Logger, ctx context.Context) (string, error) {
	placementRef := capp.Spec.Site
	if capp.Spec.Site == "" {
		placementRef = r.Placements[0]
	}
	placement := clusterv1beta1.Placement{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: placementRef, Namespace: utils.PlacementsNamespace}, &placement); err != nil {
		return "", err
	}

	// TODO rgolangh - an error with a proper type is easier to handle than a valid string. ErrNoManagedCluster for
	// example, and then the upper layer can handle it like any other error. what if someone decided to call their cluster
	// for whatever reason, 'requeue'?

	placementDecisions, err := utils.GetPlacementDecisionList(capp, log, ctx, placementRef, r.Client)
	// FIXME - rgolangh - check the error here, because the decisions might be nil so this might panic
	if err != nil {
		return "", err
	}
	if len(placementDecisions.Items) == 0 {
		log.Info("unable to find any PlacementDecision, try again after 10 seconds")
		return "requeue", nil
	}

	managedClusterName := utils.GetDecisionClusterName(placementDecisions, log)

	if managedClusterName == "" {
		return "requeue", nil
	}
	log.Info("done reconciling Workflow for Placement evaluation")
	return managedClusterName, nil
}
