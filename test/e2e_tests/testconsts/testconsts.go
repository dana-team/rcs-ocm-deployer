package testconsts

import (
	"time"

	"github.com/dana-team/container-app-operator/api/v1alpha1"
)

const (
	Timeout           = 300 * time.Second
	Interval          = 2 * time.Second
	DefaultEventually = 2 * time.Second
)

var (
	RCSAPIGroup                = v1alpha1.GroupVersion.Group
	LastUpdatedByAnnotationKey = RCSAPIGroup + "/last-updated-by"
	AnnotationKeyHasPlacement  = RCSAPIGroup + "/has-placement"
)

const (
	Placement                = "test-placement"
	Cluster1                 = "cluster1"
	Cluster2                 = "cluster2"
	AddOnPlacementScoresName = "resource-usage-score"
)
