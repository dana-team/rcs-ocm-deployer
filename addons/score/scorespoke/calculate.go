package scorespoke

import (
	"fmt"

	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	corev1informers "k8s.io/client-go/informers/core/v1"
	corev1lister "k8s.io/client-go/listers/core/v1"
)

const (
	maxScore              = float64(100)
	minScore              = float64(-100)
	defaultMaxCPUCount    = "100"
	defaultMinCPUCount    = "0"
	defaultMaxMemoryBytes = "1099511627776"
	defaultMinMemoryBytes = "0"
	maxCPUCount           = "MAX_CPU_COUNT"
	minCPUCount           = "MIN_CPU_COUNT"
	maxMemoryBytes        = "MAX_MEMORY_BYTES"
	minMemoryBytes        = "MIN_MEMORY_BYTES"
)

// ResourceScore defines a struct used to compute the score of a resource.
type ResourceScore struct {
	nodeLister        corev1lister.NodeLister
	podListener       corev1lister.PodLister
	enablePodOverhead bool
	logger            logr.Logger
}

// NewResourceScore returns a new instance of ResourceScore.
func NewResourceScore(nodeInformer corev1informers.NodeInformer, podInformer corev1informers.PodInformer, logger logr.Logger) *ResourceScore {
	return &ResourceScore{
		nodeLister:        nodeInformer.Lister(),
		podListener:       podInformer.Lister(),
		enablePodOverhead: true,
		logger:            logger,
	}
}

// calculateScore returns the score based on CPU and Memory request in the cluster.
func (s *ResourceScore) calculateScore() (int64, int64, error) {
	cpuAllocation, cpuUsage, err := s.resourceAllocationUsage(corev1.ResourceCPU)
	if err != nil {
		return 0, 0, fmt.Errorf("failed calculating CPU allocation and usage: %v", err.Error())
	}

	memoryAllocation, memoryUsage, err := s.resourceAllocationUsage(corev1.ResourceMemory)
	if err != nil {
		return 0, 0, fmt.Errorf("failed calculating Memory allocation and usage: %v", err.Error())
	}

	return s.normalizeScore(cpuAllocation, cpuUsage, memoryAllocation, memoryUsage)
}

// resourceAllocationUsage returns the allocatable amount and used amount of a certain resource.
func (s *ResourceScore) resourceAllocationUsage(resource corev1.ResourceName) (float64, float64, error) {
	allocation, err := s.calculateClusterAllocatable(resource)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to calculate allocation for resource %q: %v", resource.String(), err.Error())
	}

	usage, err := s.calculatePodResourceRequest(resource)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to calculate usgae for resource %q: %v", resource.String(), err.Error())
	}

	return allocation, usage, nil
}

// normalizeScore returns the normalized score of the CPU and Memory based on the passed values.
// A valid score must be in the range minScore to maxScore, and there is a need to normalize the
// scores before updating it into AddOnPlacementScore.
func (s *ResourceScore) normalizeScore(cpuAllocation, cpuUsage, memoryAllocation, memoryUsage float64) (int64, int64, error) {
	s.logger.Info("Computing normalized score from resource allocation and usage",
		"cpuAllocation", cpuAllocation, "cpuUsage", cpuUsage,
		"memoryAllocation", memoryAllocation, "memoryUsage", memoryUsage)

	minMax, err := s.getMaxMinValues(maxCPUCount, minCPUCount, maxMemoryBytes, minMemoryBytes)
	if err != nil {
		return 0, 0, fmt.Errorf("failed getting max and min values: %v", err.Error())
	}

	availableCPU := cpuAllocation - cpuUsage
	availableMem := memoryAllocation - memoryUsage

	cpuScore := int64(s.calculateNormalizedScore(availableCPU, minMax[maxCPUCount], minMax[minCPUCount]))
	memoryScore := int64(s.calculateNormalizedScore(availableMem, minMax[maxMemoryBytes], minMax[minMemoryBytes]))

	s.logger.Info("Successfully calculated normalized scores", "cpuScore", cpuScore, "memScore", memoryScore)

	return cpuScore, memoryScore, nil
}

// calculateNormalizedScore calculates the normalized score. Suppose the actual, non-normalized score, is X.
// Then, the formula to calculate the score is as follows:
// When the X is greater than this maximum value, the cluster can be considered healthy enough to deploy applications,
// and the score can be set as maxScore.
// When X is less than the minimum value, the score can be set as minScore.
// When X is in the interval [minScore, maxScore], then score ＝ ( (maxScore-minScore)*(X-minCount) ) / (maxCount - minCount) + minScore.
// The returned score should be an integer.
func (s *ResourceScore) calculateNormalizedScore(available, maxCount, minCount float64) float64 {
	var score float64

	switch {
	case available > maxCount:
		score = maxScore
	case available <= minCount:
		score = minScore
	default:
		score = ((maxScore-minScore)*(available-minCount))/(maxCount-minCount) + minScore
	}

	return score
}

// calculateClusterAllocatable returns the allocatable quantity of a given resource by listing
// the nodes in the cluster and summing their allocatable quantities.
func (s *ResourceScore) calculateClusterAllocatable(resourceName corev1.ResourceName) (float64, error) {
	nodes, err := s.nodeLister.List(labels.Everything())
	if err != nil {
		return 0, err
	}

	allocatableList := make(map[corev1.ResourceName]resource.Quantity)
	for _, node := range nodes {
		if !node.Spec.Unschedulable {
			for key, value := range node.Status.Allocatable {
				if allocatable, exist := allocatableList[key]; exist {
					allocatable.Add(value)
					allocatableList[key] = allocatable
				} else {
					allocatableList[key] = value
				}
			}
		}
	}

	quantity := allocatableList[resourceName]
	return quantity.AsApproximateFloat64(), nil
}

// calculatePodResourceRequest returns the total requested quantity of a certain resource by summing up
// the request of all containers in the cluster by looping over all pods.
func (s *ResourceScore) calculatePodResourceRequest(resourceName corev1.ResourceName) (float64, error) {
	var podRequest float64
	var podCount int

	pods, err := s.podListener.List(labels.Everything())
	if err != nil {
		return 0, err
	}

	for _, pod := range pods {
		for i := range pod.Spec.Containers {
			container := &pod.Spec.Containers[i]
			value := s.getRequestForResource(resourceName, container.Resources.Requests)
			podRequest += value
		}

		for i := range pod.Spec.InitContainers {
			initContainer := &pod.Spec.InitContainers[i]
			value := s.getRequestForResource(resourceName, initContainer.Resources.Requests)
			if podRequest < value {
				podRequest = value
			}
		}

		// If Overhead is being utilized, add to the total requests for the pod
		if pod.Spec.Overhead != nil && s.enablePodOverhead {
			if quantity, found := pod.Spec.Overhead[resourceName]; found {
				podRequest += quantity.AsApproximateFloat64()
			}
		}
		podCount++
	}
	return podRequest, nil
}

// getRequestForResource retrieves the resource request value for the specified resource from the given list of requests.
// It returns 0 if the requests pointer is nil or if the request for the specified resource is not found.
func (s *ResourceScore) getRequestForResource(resource corev1.ResourceName, requests corev1.ResourceList) float64 {
	if requests == nil {
		return 0
	}

	switch resource {
	case corev1.ResourceCPU:
		cpuQuantity, found := requests[corev1.ResourceCPU]
		if !found {
			return 0
		}
		return cpuQuantity.AsApproximateFloat64()
	case corev1.ResourceMemory:
		memQuantity, found := requests[corev1.ResourceMemory]
		if !found {
			return 0
		}
		return memQuantity.AsApproximateFloat64()
	default:
		quantity, found := requests[resource]
		if !found {
			return 0
		}
		return quantity.AsApproximateFloat64()
	}
}

// getMaxMinValues retrieves the maximum and minimum values for CPU and memory counts from environment variables.
// If the environment variables are not set, it uses default values.
func (s *ResourceScore) getMaxMinValues(maxCPUCount, minCPUCount, maxMemoryBytes, minMemoryBytes string) (map[string]float64, error) {
	keys := []string{maxCPUCount, minCPUCount, maxMemoryBytes, minMemoryBytes}
	envVars := getEnvAsMap(keys)
	defaults := map[string]string{
		maxCPUCount:    defaultMaxCPUCount,
		minCPUCount:    defaultMinCPUCount,
		maxMemoryBytes: defaultMaxMemoryBytes,
		minMemoryBytes: defaultMinMemoryBytes,
	}

	merged := mergeMaps(envVars, defaults)
	return convertMapStringToFloat64(merged)
}
