package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	workv1 "open-cluster-management.io/api/work/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const NamespaceManifestWorkPrefix = "mw-create-"

// GenerateManifestWorkGeneric creates the ManifestWork that wraps object as payload
// With the status sync feedback of Service's phase
func GenerateManifestWorkGeneric(name, namespace string, manifests []workv1.Manifest, machineConfigOptions ...workv1.ManifestConfigOption) *workv1.ManifestWork {
	return &workv1.ManifestWork{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: workv1.ManifestWorkSpec{
			Workload: workv1.ManifestsTemplate{
				Manifests: manifests,
			},
			ManifestConfigs: machineConfigOptions,
		},
	}
}

// GenerateManifestConfigOption generates manifestConfigObject from given parameters
func GenerateManifestConfigOption(obj client.Object, resource, group string, feedbackRules ...workv1.FeedbackRule) workv1.ManifestConfigOption {
	return workv1.ManifestConfigOption{
		ResourceIdentifier: workv1.ResourceIdentifier{
			Group:     group,
			Resource:  resource,
			Namespace: obj.GetNamespace(),
			Name:      obj.GetName(),
		},
		FeedbackRules: feedbackRules,
	}
}

// GenerateFeedbackRule generates feedbackRule from given parameters
func GenerateFeedbackRule(workStatusName, path string) workv1.FeedbackRule {
	return workv1.FeedbackRule{
		Type: workv1.JSONPathsType, JsonPaths: []workv1.JsonPath{{Name: workStatusName, Path: path}},
	}
}
