package unitTests

import (
	"context"
	"testing"

	cappv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
	"github.com/dana-team/rcs-ocm-deployer/internals/placement/adapters"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = cappv1alpha1.AddToScheme(s)
	_ = clusterv1beta1.AddToScheme(s)
	return s
}

func newFakeClient() client.Client {
	scheme := newScheme()
	return fake.NewClientBuilder().WithScheme(scheme).Build()
}

func TestGetPlacementDecisionList(t *testing.T) {
	// Create a fake client
	fakeClient := newFakeClient()

	// Call GetPlacementDecisionList with the test Capp, context, fake client, and placementRef
	placementDecisions, err := adapters.GetPlacementDecisionList(context.Background(), "placement-ref", "test-namespace", fakeClient)

	// Assert that there are no errors
	assert.NoError(t, err)

	// Assert that the returned placementDecisions is not nil
	assert.NotNil(t, placementDecisions)
}

func TestGetDecisionClusterName(t *testing.T) {
	// Create a test PlacementDecisionList
	placementDecisions := &clusterv1beta1.PlacementDecisionList{
		Items: []clusterv1beta1.PlacementDecision{
			{
				Status: clusterv1beta1.PlacementDecisionStatus{
					Decisions: []clusterv1beta1.ClusterDecision{
						{
							ClusterName: "cluster-1",
						},
						{
							ClusterName: "cluster-2",
						},
					},
				},
			},
		},
	}

	// Create a fake logger
	// Call GetDecisionClusterName with the test PlacementDecisionList and fake logger
	clusterName := adapters.GetDecisionClusterName(placementDecisions, logr.Discard())

	// Assert that the returned clusterName is "cluster-1"
	assert.Equal(t, "cluster-1", clusterName)
}
