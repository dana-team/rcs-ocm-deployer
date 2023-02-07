package utils

import (
	rcsv1alpha1 "github.com/dana-team/rcs-ocm-deployer/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	knativev1 "knative.dev/serving/pkg/apis/serving/v1"

	workv1 "open-cluster-management.io/api/work/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const AnnotationKeyHasPlacement = "dana.io/has-placement"

// GenerateManifestWorkName returns the ManifestWork name for a given workflow.
// It uses the Service name with the suffix of the first 5 characters of the UID
func GenerateManifestWorkName(service knativev1.Service) string {
	return service.Name + "-" + string(service.UID)[0:5]
}

func ContainesPlacementAnnotation(capp rcsv1alpha1.Capp) bool {
	annos := capp.GetAnnotations()
	if len(annos) == 0 {
		return false
	}

	namespace, ok := annos[AnnotationKeyHasPlacement]
	return ok && len(namespace) > 0
}

func ContainsValidOCMNamespaceAnnotation(service rcsv1alpha1.Capp) bool {
	annos := service.GetAnnotations()
	if len(annos) == 0 {
		return false
	}

	namespace, ok := annos["AnnotationNamespaceCreated"]
	return ok && len(namespace) > 0
}

// PrepareServiceForWorkPayload modifies the Service:
// - set the namespace value
func PrepareServiceForWorkPayload(capp rcsv1alpha1.Capp) rcsv1alpha1.Capp {
	capp.TypeMeta = metav1.TypeMeta{
		APIVersion: rcsv1alpha1.GroupVersion.String(),
		Kind:       capp.Kind,
	}
	capp.ObjectMeta = metav1.ObjectMeta{
		Name:        capp.Name,
		Namespace:   capp.Namespace,
		Labels:      capp.Labels,
		Annotations: capp.Annotations,
	}

	return capp
}

// GenerateManifestWorkGeneric creates the ManifestWork that wraps object as payload
// With the status sync feedback of Service's phase
func GenerateManifestWorkGeneric(name, namespace string, obj client.Object, machineConfigOptions ...workv1.ManifestConfigOption) *workv1.ManifestWork {
	return &workv1.ManifestWork{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: workv1.ManifestWorkSpec{
			Workload: workv1.ManifestsTemplate{
				Manifests: []workv1.Manifest{{RawExtension: runtime.RawExtension{Object: obj}}},
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

// GenerateNamespace generates namespace from given names
func GenerateNamespace(name string) corev1.Namespace {
	return corev1.Namespace{
		TypeMeta:   metav1.TypeMeta{Kind: "Namespace", APIVersion: corev1.SchemeGroupVersion.Version},
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       corev1.NamespaceSpec{},
		Status:     corev1.NamespaceStatus{},
	}
}
