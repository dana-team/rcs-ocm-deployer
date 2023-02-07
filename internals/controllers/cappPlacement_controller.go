package controllers

import (
	"context"
	"time"

	rcsv1alpha1 "github.com/dana-team/rcs-ocm-deployer/api/v1alpha1"
	utils "github.com/dana-team/rcs-ocm-deployer/internals/utils"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
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

	return r.setDestinationSite(capp, l, ctx)
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

func (r *ServicePlacementReconciler) setDestinationSite(capp rcsv1alpha1.Capp, log logr.Logger, ctx context.Context) (ctrl.Result, error) {
	if capp.Status.ApplicationLinks.Site != "" {
		r.addCappHasPlacementAnnotation(capp, capp.Status.ApplicationLinks.Site, ctx)
	}
	placementRef := capp.Spec.Site
	if placementRef == "" || slices.Contains(r.Placements, placementRef) {
		return r.pickDecision(capp, log, ctx)
	}
	if err := r.updateCappDestination(capp, placementRef, ctx); err != nil {
		log.Error(err, "unable to update capp")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// pickDecision gets a service logger and context
// The function decides the name of the managed cluster to deploy to
// And adds an annotation to the service with its name
// Returns controller result and error

func (r *ServicePlacementReconciler) pickDecision(capp rcsv1alpha1.Capp, log logr.Logger, ctx context.Context) (ctrl.Result, error) {
	placementRef := capp.Spec.Site
	if capp.Spec.Site == "" {
		// The default placement(regular clusters)
		placementRef = r.Placements[0]
	}
	placement := clusterv1beta1.Placement{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: placementRef, Namespace: capp.Namespace}, &placement); err != nil {
		return ctrl.Result{}, err
	}
	placementDecisions, err := r.getPlacementDecisionList(capp, log, ctx, placementRef)
	if len(placementDecisions.Items) == 0 {
		log.Info("unable to find any PlacementDecision, try again after 10 seconds")
		return ctrl.Result{RequeueAfter: time.Second * 10}, nil
	}
	if err != nil {
		return ctrl.Result{}, err
	}
	managedClusterName := getDecisionClusterName(placementDecisions, log)
	if managedClusterName == "" {
		return ctrl.Result{RequeueAfter: time.Second * 10}, nil
	}
	log.Info("updating Service with annotation ManagedCluster: " + managedClusterName)
	if err := r.updateCappDestination(capp, managedClusterName, ctx); err != nil {
		log.Error(err, "unable to update Knative Service")
		return ctrl.Result{}, err
	}
	log.Info("done reconciling Workflow for Placement evaluation")
	return ctrl.Result{}, nil
}

// getPlacementDecisionList gets service ,logger and placement name
// The function returns a placementDecisionList containing the placementDecision of the placement
func (r *ServicePlacementReconciler) getPlacementDecisionList(capp rcsv1alpha1.Capp, log logr.Logger, ctx context.Context, placementRef string) (*clusterv1beta1.PlacementDecisionList, error) {

	listopts := &client.ListOptions{}
	// query all placementdecisions of the placement
	requirement, err := labels.NewRequirement(clusterv1beta1.PlacementLabel, selection.Equals, []string{placementRef})
	if err != nil {
		log.Error(err, "unable to create new PlacementDecision label requirement")
		return nil, err
	}
	labelSelector := labels.NewSelector().Add(*requirement)
	listopts.LabelSelector = labelSelector
	listopts.Namespace = capp.Namespace
	placementDecisions := &clusterv1beta1.PlacementDecisionList{}
	if err = r.Client.List(ctx, placementDecisions, listopts); err != nil {
		log.Error(err, "unable to list PlacementDecisions")
		return nil, err
	}
	return placementDecisions, nil
}

// getDecisionClusterName gets placementDecisionList and a logger
// The function extracts from the placementDecision the managedCluster name to deploy to and returns it.
func getDecisionClusterName(placementDecisions *clusterv1beta1.PlacementDecisionList, log logr.Logger) string {
	// TODO only handle one PlacementDecision target for now
	pd := placementDecisions.Items[0]
	if len(pd.Status.Decisions) == 0 {
		log.Info("unable to find any Decisions from PlacementDecision, try again after 10 seconds")
		return ""
	}

	// TODO only using the first decision
	managedClusterName := pd.Status.Decisions[0].ClusterName
	if len(managedClusterName) == 0 {
		log.Info("unable to find a valid ManagedCluster from PlacementDecision, try again after 10 seconds")
		return ""
	}
	return managedClusterName
}

// updateServiceAnnotations gets service, managed cluster name and context
// The function adds an annotation to the service containing the name of the managed cluster
// Returns error if occured
func (r *ServicePlacementReconciler) updateCappDestination(capp rcsv1alpha1.Capp, managedClusterName string, ctx context.Context) error {
	capp.Status.ApplicationLinks.Site = managedClusterName
	return r.Client.Status().Update(ctx, &capp)
}

func (r *ServicePlacementReconciler) addCappHasPlacementAnnotation(capp rcsv1alpha1.Capp, managedClusterName string, ctx context.Context) error {
	cappAnno := capp.GetAnnotations()
	if cappAnno == nil {
		cappAnno = make(map[string]string)
	}
	cappAnno[utils.AnnotationKeyHasPlacement] = managedClusterName
	capp.SetAnnotations(cappAnno)
	return r.Client.Update(ctx, &capp)
}
