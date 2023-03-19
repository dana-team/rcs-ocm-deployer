package webhooks

import (
	"context"

	rcsv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
	"k8s.io/utils/strings/slices"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var SupportedScaleMetrics = []string{"rps", "cuncurrency", "cpu", "memory"}

// This function checks if the specified scaling metric is supported by the system and returns a boolean value accordingly.
func isScaleMetricSupported(capp rcsv1alpha1.Capp) bool {
	return slices.Contains(SupportedScaleMetrics, capp.Spec.ScaleMetric)
}

// This function checks if the specified site cluster name is valid or not. It returns a boolean value based on the validity of the specified site cluster name.
func isSiteClusterName(capp rcsv1alpha1.Capp, r client.Client, ctx context.Context) bool {
	if capp.Spec.Site == "" {
		return true
	}
	clusters, _ := getManagedClusters(r, ctx)
	return slices.Contains(clusters, capp.Spec.ScaleMetric)

}

// This function retrieves the list of managed clusters from the Kubernetes API server and returns the list of cluster names as a slice of strings. If there is an error while retrieving the list of managed clusters, the function returns an error.
func getManagedClusters(r client.Client, ctx context.Context) ([]string, error) {
	clusterNames := []string{}
	clusters := clusterv1.ManagedClusterList{}
	if err := r.List(ctx, &clusters); err != nil {
		return clusterNames, err
	}
	for _, cluster := range clusters.Items {
		clusterNames = append(clusterNames, cluster.Name)
	}
	return clusterNames, nil
}
