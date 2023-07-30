package webhooks

import (
	"context"

	"regexp"

	rcsv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
	"k8s.io/utils/strings/slices"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var SupportedScaleMetrics = []string{"rps", "concurrency", "cpu", "memory"}

// isScaleMetricSupported checks if the specified scaling metric is supported by the system.
// It takes a rcsv1alpha1.Capp object and returns a boolean value indicating whether the metric is supported or not.
func isScaleMetricSupported(capp rcsv1alpha1.Capp) bool {
	return slices.Contains(SupportedScaleMetrics, capp.Spec.ScaleMetric)
}

// isSiteValid checks if the specified site cluster name is valid or not.
// It takes a rcsv1alpha1.Capp object, a list of placements, a Kubernetes client.Client, and a context.Context.
// The function returns a boolean value based on the validity of the specified site cluster name.
func isSiteVaild(capp rcsv1alpha1.Capp, placements []string, r client.Client, ctx context.Context) bool {
	if capp.Spec.Site == "" {
		return true
	}
	clusters, _ := getManagedClusters(r, ctx)
	return slices.Contains(clusters, capp.Spec.Site) || slices.Contains(placements, capp.Spec.Site)

}

// getManagedClusters retrieves the list of managed clusters from the Kubernetes API server
// and returns the list of cluster names as a slice of strings.
// If there is an error while retrieving the list of managed clusters, the function returns an error.
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

// validateDomainRegex checks if the specified domain name matches the valid regex pattern.
// It takes a domain name as a string and returns a boolean value indicating whether the domain name is valid or not.
func validateDomainRegex(domainname string) bool {
	match, _ := regexp.MatchString("^[a-zA-Z0-9][a-zA-Z0-9-]{0,61}[a-zA-Z0-9](?:\\.[a-zA-Z]{2,})+$", domainname)
	return match
}
