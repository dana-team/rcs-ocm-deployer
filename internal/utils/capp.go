// Package utils provides utility functions for working with Kubernetes resources and custom resources defined in the
// container-app-operator API.
package utils

import (
	cappv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
	"github.com/dana-team/rcs-ocm-deployer/api/v1alpha1"
)

var (
	RCSAPIGroup = v1alpha1.GroupVersion.Group

	MangedByLableKey = RCSAPIGroup + "/managed-by"

	// AnnotationKeyHasPlacement is the key used to store the managed cluster name in an annotation on a Capp resource
	AnnotationKeyHasPlacement = RCSAPIGroup + "/has-placement"
)

const (
	// RCSConfigName is the name of the RCS Deployer Config CRD instance
	RCSConfigName = "rcs-config"

	// RCSConfigNamespace is the namespace that contains the RCS Deployer Config CRD instance
	RCSConfigNamespace = "rcs-deployer-system"

	MangedByLabelValue = "rcs"
)

// ContainsPlacementAnnotation checks if a Capp resource has an annotation indicating it has been placed on a managed cluster.
func ContainsPlacementAnnotation(capp cappv1alpha1.Capp) bool {
	annotations := capp.GetAnnotations()
	if len(annotations) == 0 {
		return false
	}

	namespace, ok := annotations[AnnotationKeyHasPlacement]
	return ok && len(namespace) > 0
}
