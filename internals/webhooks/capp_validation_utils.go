package webhooks

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/types"

	cappv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/strings/slices"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/network"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// isSiteValid checks if the specified site cluster name is valid or not.
// It takes a cappv1alpha1.Capp object, a list of placements, a Kubernetes client.Client, and a context.Context.
// The function returns a boolean value based on the validity of the specified site cluster name.
func isSiteValid(capp cappv1alpha1.Capp, placements []string, r client.Client, ctx context.Context) bool {
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
// It takes a cappv1alpha1.Capp object and returns aggregated error if the any of the validations falied.
func validateTlsFields(capp cappv1alpha1.Capp, client client.Client, ctx context.Context) (errs *apis.FieldError) {
	if capp.Spec.RouteSpec.TlsEnabled && capp.Spec.RouteSpec.TlsSecret == "" {
		errs = errs.Also(apis.ErrGeneric(
			"it's forbidden to set '.spec.routeSpec.tlsEnabled' to 'true' without specifying a secret name in the '.spec.routeSpec.tlsSecret' field"))
	} else if !capp.Spec.RouteSpec.TlsEnabled && capp.Spec.RouteSpec.TlsSecret != "" {
		errs = errs.Also(apis.ErrGeneric(
			"it's forbidden to set '.spec.routeSpec.tlsEnabled' to 'false' and specifying a secret name in the '.spec.routeSpec.tlsSecret' field"))
	} else if capp.Spec.RouteSpec.TlsEnabled && capp.Spec.RouteSpec.TlsSecret != "" && capp.DeletionTimestamp.IsZero() {
		// The condition that checks if the deletion timestamp is zero is crucial to handle an edge case.
		// This case occurs when deleting a namespace, which results in the secret being deleted before attempting to remove the capp's finalizer through an update request.
		// If the secret doesn't exist anymore, this webhook will prevent the finalizer from being removed from the capp.
		// Therefore, this check ensures the process can proceed without interference due to the secret's deletion.
		tlsSecret, err := getSecret(capp.Spec.RouteSpec.TlsSecret, capp.Namespace, client, ctx)
		if err != nil {
			errs = errs.Also(apis.ErrGeneric(fmt.Sprintf(
				"it's forbidden to set '.spec.routeSpec.tlsSecret' without creating a matching tlsSecret. %q does not exist", capp.Spec.RouteSpec.TlsSecret)))
		} else if tlsSecret.Type != v1.SecretTypeTLS {
			errs = errs.Also(apis.ErrGeneric(
				fmt.Sprintf("it's forbidden to set '.spec.routeSpec.tlsSecret' without creating a matching tlsSecret. The secret type %q must be %q", tlsSecret.Name, v1.SecretTypeTLS)))
		}
	}
	return errs
}

// getSecret fetches the secret in the route spec.
func getSecret(secretName string, namespace string, client client.Client, ctx context.Context) (*v1.Secret, error) {
	secret := &v1.Secret{}
	err := client.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, secret)
	return secret, err
}

// validateLogSpec checks if the LogSpec is valid based on the Type field.
func validateLogSpec(logSpec cappv1alpha1.LogSpec) *apis.FieldError {
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
			fmt.Sprintf("Invalid LogSpec Type: %q. Valid types are: %q", logSpec.Type, strings.Join(validTypes, ", ")),
			"logSpec.Type")
	}
	missingFields := findMissingFields(logSpec, required)
	if len(missingFields) > 0 {
		return apis.ErrGeneric(
			fmt.Sprintf("%s log configuration is missing required fields: %q", logSpec.Type, strings.Join(missingFields, ", ")),
			"logSpec")
	}
	return nil
}

// findMissingFields checks for missing fields in LogSpec.
func findMissingFields(logSpec cappv1alpha1.LogSpec, required []string) []string {
	var missingFields []string
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
