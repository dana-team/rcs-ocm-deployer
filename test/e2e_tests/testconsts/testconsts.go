package testconsts

import (
	"time"

	"github.com/dana-team/container-app-operator/api/v1alpha1"
)

const (
	Timeout            = 300 * time.Second
	Interval           = 2 * time.Second
	DefaultEventually  = 2 * time.Second
	MangedByLabelValue = "rcs"
)

var (
	RCSAPIGroup                = v1alpha1.GroupVersion.Group
	LastUpdatedByAnnotationKey = RCSAPIGroup + "/last-updated-by"
	AnnotationKeyHasPlacement  = RCSAPIGroup + "/has-placement"
	MangedByLableKey           = RCSAPIGroup + "/managed-by"
)

const (
	Placement                = "test-placement"
	Cluster1                 = "cluster1"
	Cluster2                 = "cluster2"
	AddOnPlacementScoresName = "resource-usage-score"
)
