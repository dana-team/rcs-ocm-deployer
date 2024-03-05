package utils

import (
	"context"

	. "github.com/onsi/gomega"
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetAddOnPlacementScore fetches an existing GetAddOnPlacementScore and returns its instance.
func GetAddOnPlacementScore(k8sClient client.Client, name string, namespace string) *clusterv1alpha1.AddOnPlacementScore {
	addonPlacementScore := &clusterv1alpha1.AddOnPlacementScore{}
	Expect(k8sClient.Get(context.Background(), client.ObjectKey{Name: name, Namespace: namespace}, addonPlacementScore)).To(Succeed())
	return addonPlacementScore
}
