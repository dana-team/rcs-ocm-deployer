package testconsts

import "time"

const (
	Timeout                  = 60 * time.Second
	Interval                 = 2 * time.Second
	DefaultEventuallySeconds = 2
)

const (
	Placement = "test-placement"
	Cluster   = "cluster1"
)
