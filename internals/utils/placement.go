// FIXME - rgolangh - seems like only the controller really use those functions. So I think both
// could be unexported and moved into the controllers package. Also, it is possible to
// make them the reconciler methods, so they'll get the client and the logger instance from it.
package utils

import (
	"context"

	rcsv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	PlacementsNamespace = "open-cluster-management-global-set"
)

// GetPlacementDecisionList gets service ,logger and placement name
// The function returns a placementDecisionList containing the placementDecision of the placement
// FIXME - capp argument seems not be in use. remove?
func GetPlacementDecisionList(capp rcsv1alpha1.Capp, log logr.Logger, ctx context.Context, placementRef string, r client.Client) (*clusterv1beta1.PlacementDecisionList, error) {
	// query all placementdecisions of the placement
	requirement, err := labels.NewRequirement(clusterv1beta1.PlacementLabel, selection.Equals, []string{placementRef})
	if err != nil {
		log.Error(err, "unable to create new PlacementDecision label requirement")
		return nil, err
	}
	// Note - moved it here cause there's no point intializing it few lines before it is even in use.
	listopts := &client.ListOptions{
		LabelSelector: labels.NewSelector().Add(*requirement),
		Namespace:     PlacementsNamespace,
	}
	placementDecisions := &clusterv1beta1.PlacementDecisionList{}
	if err = r.List(ctx, placementDecisions, listopts); err != nil {
		log.Error(err, "unable to list PlacementDecisions")
		return nil, err
	}
	return placementDecisions, nil
}

// GetDecisionClusterName gets placementDecisionList and a logger
// The function extracts from the placementDecision the managedCluster name to deploy to and returns it.
func GetDecisionClusterName(placementDecisions *clusterv1beta1.PlacementDecisionList, log logr.Logger) string {
	// TODO only handle one PlacementDecision target for now
	pd := placementDecisions.Items[0]
	if len(pd.Status.Decisions) == 0 {
		log.Info("unable to find any Decisions from PlacementDecision, try again after 10 seconds")
		return ""
	}

	// TODO rgolangh - how do you know you have 2 decisions in the status here? this would panic and your controller
	// would crash-loop until this is valid.

	// TODO only using the first decision
	managedClusterName := pd.Status.Decisions[0].ClusterName
	if managedClusterName == "local-cluster" {
		managedClusterName = pd.Status.Decisions[1].ClusterName
	}
	if len(managedClusterName) == 0 {
		log.Info("unable to find a valid ManagedCluster from PlacementDecision, try again after 10 seconds")
		return ""
	}
	return managedClusterName
}
