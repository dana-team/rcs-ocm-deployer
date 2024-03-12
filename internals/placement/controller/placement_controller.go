package controller

import (
	"context"
	"fmt"
	"time"

	rcsv1alpha1 "github.com/dana-team/rcs-ocm-deployer/api/v1alpha1"
	"github.com/dana-team/rcs-ocm-deployer/internals/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cappv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
	"github.com/dana-team/rcs-ocm-deployer/internals/placement/adapters"
	"github.com/dana-team/rcs-ocm-deployer/internals/utils/events"

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

const (
	// RCSConfigName is the name of the RCS Deployer Config CRD instance
	RCSConfigName = "rcs-config"
	// RCSConfigNamespace is the namespace that contains the RCS Deployer Config CRD instance
	RCSConfigNamespace = "rcs-deployer-system"
	// DefaultPlacementsNamespace is the default namespace contains the placements
	DefaultPlacementsNamespace = "default"

	RequeueTime = 20 * time.Second
)

// ErrNoManagedCluster is a custom error type for the requeue scenario
type ErrNoManagedCluster struct{}

func (e ErrNoManagedCluster) Error() string {
	return "No managed cluster was found to deploy on. Requeue"
}

// PlacementReconciler reconciles a CappPlacement object
type PlacementReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	EventRecorder record.EventRecorder
}

//+kubebuilder:rbac:groups=rcs.dana.io,resources=capps,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=cluster.open-cluster-management.io,resources=placementdecisions,verbs=get;list;watch
//+kubebuilder:rbac:groups=cluster.open-cluster-management.io,resources=placements,verbs=get;list;watch
//+kubebuilder:rbac:groups=cluster.open-cluster-management.io,resources=managedclusters,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *PlacementReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("CappName", req.Name, "CappNamespace", req.Namespace)
	config := rcsv1alpha1.RCSConfig{}
	if err := r.Get(ctx, types.NamespacedName{Name: RCSConfigName, Namespace: RCSConfigNamespace}, &config); err != nil {
		if statusError, isStatusError := err.(*errors.StatusError); isStatusError {
			if statusError.ErrStatus.Reason == metav1.StatusReasonNotFound {
				logger.Error(err, "rcs config has not been defined")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, err
	}
	placements := config.Spec.Placements
	placementsNamespace := config.Spec.PlacementsNamespace
	if placementsNamespace == "" {
		placementsNamespace = DefaultPlacementsNamespace
	}
	capp := cappv1alpha1.Capp{}
	if err := r.Client.Get(ctx, req.NamespacedName, &capp); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	placementRef := capp.Spec.Site
	if placementRef == "" || slices.Contains(placements, placementRef) {
		cluster, err := r.pickDecision(capp, placements, placementsNamespace, logger, ctx)
		if err != nil {
			if _, ok := err.(ErrNoManagedCluster); ok {
				logger.Info(fmt.Sprintf("Requeuing Capp %q, waiting for PlacementDecision to be satisfied", capp.Name))
				r.EventRecorder.Event(&capp, corev1.EventTypeWarning, "PlacementDecisionNotSatisfied", fmt.Sprintf("Failed to schedule Capp %q on managed cluster. PlacementDecision with optional clusters was not found for placement %q", capp.Name, placementRef))
				return ctrl.Result{RequeueAfter: RequeueTime}, nil
			}
			logger.Error(err, fmt.Sprintf("failed to pick managed cluster for placement %q", placementRef))
			return ctrl.Result{}, err
		}
		placementRef = cluster
	}
	if err := adapters.UpdateCappDestination(capp, placementRef, ctx, r.Client); err != nil {
		return ctrl.Result{}, fmt.Errorf("unable to update Capp with selected cluster: %v", err.Error())
	}
	r.EventRecorder.Event(&capp, corev1.EventTypeNormal, events.EventCappScheduled, fmt.Sprintf("Scheduled Capp %q on managed cluster %q", capp.Name, placementRef))
	return ctrl.Result{}, nil
}

var CappPredicateFunctions = predicate.Funcs{
	UpdateFunc: func(e event.UpdateEvent) bool {
		newCapp := e.ObjectNew.(*cappv1alpha1.Capp)
		return !utils.ContainsPlacementAnnotation(*newCapp)
	},
	CreateFunc: func(e event.CreateEvent) bool {
		capp := e.Object.(*cappv1alpha1.Capp)
		return !utils.ContainsPlacementAnnotation(*capp)
	},

	DeleteFunc: func(e event.DeleteEvent) bool {
		capp := e.Object.(*cappv1alpha1.Capp)
		return !utils.ContainsPlacementAnnotation(*capp)
	},
}

// SetupWithManager sets up the controller with the Manager.
func (r *PlacementReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cappv1alpha1.Capp{}).
		WithEventFilter(CappPredicateFunctions).
		Complete(r)
}

// pickDecision decides the name of the managed cluster to deploy the Capp on,
// and adds an annotation to the Capp with its name
func (r *PlacementReconciler) pickDecision(capp cappv1alpha1.Capp, placements []string, placementsNamespace string, log logr.Logger, ctx context.Context) (string, error) {
	placementRef := capp.Spec.Site
	if capp.Spec.Site == "" {
		placementRef = placements[0]
	}
	placement := clusterv1beta1.Placement{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: placementRef, Namespace: placementsNamespace}, &placement); err != nil {
		return "", fmt.Errorf("failed to get placement: %v", err.Error())
	}
	placementDecisions, err := adapters.GetPlacementDecisionList(ctx, placementRef, placementsNamespace, r.Client)
	if len(placementDecisions.Items) == 0 {
		return "", ErrNoManagedCluster{}
	}
	if err != nil {
		return "", fmt.Errorf("failed to list placementDecisions: %v", err.Error())
	}
	managedClusterName := adapters.GetDecisionClusterName(placementDecisions, log)
	if managedClusterName == "" {
		return "", ErrNoManagedCluster{}
	}
	return managedClusterName, nil
}
