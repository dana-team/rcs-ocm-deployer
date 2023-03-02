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

// getPlacementDecisionList gets service ,logger and placement name
// The function returns a placementDecisionList containing the placementDecision of the placement
func GetPlacementDecisionList(capp rcsv1alpha1.Capp, log logr.Logger, ctx context.Context, placementRef string, r client.Client) (*clusterv1beta1.PlacementDecisionList, error) {

	listopts := &client.ListOptions{}
	// query all placementdecisions of the placement
	requirement, err := labels.NewRequirement(clusterv1beta1.PlacementLabel, selection.Equals, []string{placementRef})
	if err != nil {
		log.Error(err, "unable to create new PlacementDecision label requirement")
		return nil, err
	}
	labelSelector := labels.NewSelector().Add(*requirement)
	listopts.LabelSelector = labelSelector
	listopts.Namespace = PlacementsNamespace
	placementDecisions := &clusterv1beta1.PlacementDecisionList{}
	if err = r.List(ctx, placementDecisions, listopts); err != nil {
		log.Error(err, "unable to list PlacementDecisions")
		return nil, err
	}
	return placementDecisions, nil
}

// getDecisionClusterName gets placementDecisionList and a logger
// The function extracts from the placementDecision the managedCluster name to deploy to and returns it.
func GetDecisionClusterName(placementDecisions *clusterv1beta1.PlacementDecisionList, log logr.Logger) string {
	// TODO only handle one PlacementDecision target for now
	pd := placementDecisions.Items[0]
	if len(pd.Status.Decisions) == 0 {
		log.Info("unable to find any Decisions from PlacementDecision, try again after 10 seconds")
		return ""
	}

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
