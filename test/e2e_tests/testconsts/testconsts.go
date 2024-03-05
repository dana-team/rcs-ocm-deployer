package testconsts

import "time"

const (
	Timeout                  = 300 * time.Second
	Interval                 = 2 * time.Second
	DefaultEventuallySeconds = 2
)

const (
	Placement                 = "test-placement"
	Cluster1                  = "cluster1"
	Cluster2                  = "cluster2"
	AddOnPlacementScoresName  = "resource-usage-score"
	Hostname                  = "capp.dev"
	AnnotationKeyHasPlacement = "rcs.dana.io/has-placement"
)
