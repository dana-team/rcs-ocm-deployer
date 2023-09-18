package webhooks

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/network"

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
	var clusterNames []string
	clusters := clusterv1.ManagedClusterList{}
	if err := r.List(ctx, &clusters); err != nil {
		return clusterNames, err
	}
	for _, cluster := range clusters.Items {
		clusterNames = append(clusterNames, cluster.Name)
	}
	return clusterNames, nil
}

// validateDomainName checks if the hostname is valid domain name and not part of the cluster's domain.
// it returns aggregated error if the any of the validations falied.
func validateDomainName(domainname string) (errs *apis.FieldError) {
	if domainname == "" {
		return nil
	}
	err := validation.IsFullyQualifiedDomainName(field.NewPath("name"), domainname)
	if err != nil {
		errs = errs.Also(apis.ErrGeneric(fmt.Sprintf(
			"invalid name %q: %s", domainname, err.ToAggregate()), "name"))
	}

	clusterLocalDomain := network.GetClusterDomainName()
	if strings.HasSuffix(domainname, "."+clusterLocalDomain) {
		errs = errs.Also(apis.ErrGeneric(
			fmt.Sprintf("invalid name %q: must not be a subdomain of cluster local domain %q", domainname, clusterLocalDomain), "name"))
	}
	return errs
}

// validateTlsFields checks if the fields of the tls feature in the capp spec is written correctly.
// It takes a rcsv1alpha1.Capp object and returns aggregated error if the any of the validations falied.
func validateTlsFields(capp rcsv1alpha1.Capp) (errs *apis.FieldError) {
	if capp.Spec.RouteSpec.TlsEnabled && capp.Spec.RouteSpec.TlsSecret == "" {
		errs = errs.Also(apis.ErrGeneric(fmt.Sprintf(
			"it's forbidden to set '.spec.routeSpec.tlsEnabled' to 'true' without specifying a secret name in the '.spec.routeSpec.tlsSecret' field")))
	} else if !capp.Spec.RouteSpec.TlsEnabled && capp.Spec.RouteSpec.TlsSecret != "" {
		errs = errs.Also(apis.ErrGeneric(fmt.Sprintf(
			"it's forbidden to set '.spec.routeSpec.tlsEnabled' to 'false' and specifying a secret name in the '.spec.routeSpec.tlsSecret' field")))
	}

	return errs
}

// validateLogSpec checks if the LogSpec is valid based on the Type field.
func validateLogSpec(logSpec rcsv1alpha1.LogSpec) *apis.FieldError {
	requiredFields := map[string][]string{
		"elastic": {"Host", "Index", "UserName", "PasswordSecretName"},
		"splunk":  {"Host", "Index", "HecTokenSecretName"},
	}
	required, exists := requiredFields[logSpec.Type]
	if !exists {
		validTypes := make([]string, 0, len(requiredFields))
		for validType := range requiredFields {
			validTypes = append(validTypes, validType)
		}
		return apis.ErrGeneric(
			fmt.Sprintf("Invalid LogSpec Type: %s. Valid types are: %s", logSpec.Type, strings.Join(validTypes, ", ")),
			"logSpec.Type")
	}
	missingFields := findMissingFields(logSpec, required)
	if len(missingFields) > 0 {
		return apis.ErrGeneric(
			fmt.Sprintf("%s log configuration is missing: %s", logSpec.Type, strings.Join(missingFields, ",")),
			"logSpec")
	}
	return nil
}

// findMissingFields checks for missing fields in LogSpec.
func findMissingFields(logSpec rcsv1alpha1.LogSpec, required []string) []string {
	missingFields := []string{}
	fieldValues := map[string]string{
		"Host":               logSpec.Host,
		"Index":              logSpec.Index,
		"UserName":           logSpec.UserName,
		"PasswordSecretName": logSpec.PasswordSecretName,
		"HecTokenSecretName": logSpec.HecTokenSecretName,
	}
	for _, reqField := range required {
		if value, ok := fieldValues[reqField]; !ok || value == "" {
			missingFields = append(missingFields, reqField)
		}
	}
	return missingFields
}
