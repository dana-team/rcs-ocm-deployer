package testconsts

import "time"

const (
	Timeout                  = 120 * time.Second
	Interval                 = 2 * time.Second
	DefaultEventuallySeconds = 2
)

const (
	Placement                 = "test-placement"
	Cluster                   = "cluster1"
	Hostname                  = "capp.dev"
	AnnotationKeyHasPlacement = "rcs.dana.io/has-placement"
)
