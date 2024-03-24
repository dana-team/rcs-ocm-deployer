package testconsts

import "time"

const (
	Timeout           = 300 * time.Second
	Interval          = 2 * time.Second
	DefaultEventually = 2 * time.Second
)

const (
	Placement                 = "test-placement"
	Cluster1                  = "cluster1"
	Cluster2                  = "cluster2"
	AddOnPlacementScoresName  = "resource-usage-score"
	AnnotationKeyHasPlacement = "rcs.dana.io/has-placement"
)
