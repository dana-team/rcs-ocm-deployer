// Package utils provides utility functions for working with Kubernetes resources and custom resources defined in the
// container-app-operator API.
package utils

import (
	rcsv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
)

const (
	// AnnotationKeyHasPlacement is the key used to store the managed cluster name in an annotation on a Capp resource
	AnnotationKeyHasPlacement = "rcs.dana.io/has-placement"
	// RcsConfigName is the name of the RCS Deployer Config CRD instance
	RcsConfigName = "rcs-config"
	// RcsConfigNamespace is the namespace that contains the RCS Deployer Config CRD instance
	RcsConfigNamespace = "rcs-deployer-system"
)

// ContainsPlacementAnnotation checks if a Capp resource has an annotation indicating it has been placed on a managed cluster.
func ContainsPlacementAnnotation(capp rcsv1alpha1.Capp) bool {
	annotations := capp.GetAnnotations()
	if len(annotations) == 0 {
		return false
	}

	namespace, ok := annotations[AnnotationKeyHasPlacement]
	return ok && len(namespace) > 0
}
