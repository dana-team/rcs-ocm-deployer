package controllers

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	knativev1 "knative.dev/serving/pkg/apis/serving/v1"
	workv1 "open-cluster-management.io/api/work/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ContainsValidOCMAnnotation(service knativev1.Service) bool {
	annos := service.GetAnnotations()
	if len(annos) == 0 {
		return false
	}

	managedClusterName, ok := annos[AnnotationKeyOCMManagedCluster]
	return ok && len(managedClusterName) > 0
}

func ContainsValidOCMNamespaceAnnotation(service knativev1.Service) bool {
	annos := service.GetAnnotations()
	if len(annos) == 0 {
		return false
	}

	namespace, ok := annos[AnnotationKeyOCMManagedClusterNamespace]
	return ok && len(namespace) > 0
}

func ContainsValidOCMPlacementAnnotation(service knativev1.Service) bool {
	annos := service.GetAnnotations()
	if len(annos) == 0 {
		return false
	}

	placementName, ok := annos[AnnotationKeyOCMPlacement]
	return ok && len(placementName) > 0
}

func ContainsNamespaceCreated(service knativev1.Service) bool {
	annos := service.GetAnnotations()
	if len(annos) == 0 {
		return false
	}

	nsCreated, ok := annos[AnnotationNamespaceCreated]
	return ok && nsCreated == "true"
}

// generateServiceNamespace returns the intended namespace for the Service in the following priority
// 1) Annotation specified custom namespace
// 2) Fallsback to 'knative'
func generateServiceNamespace(service knativev1.Service) string {
	annos := service.GetAnnotations()
	appNamespace := annos[AnnotationKeyOCMManagedClusterNamespace]
	if len(appNamespace) > 0 {
		return appNamespace
	}
	return "knative"
}

// generateManifestWorkName returns the ManifestWork name for a given workflow.
// It uses the Workflow name with the suffix of the first 5 characters of the UID
func GenerateManifestWorkName(service knativev1.Service) string {
	return service.Name + "-" + string(service.UID)[0:5]
}

// PrepareServiceForWorkPayload modifies the Service:
// - reste the type and object meta
// - set the namespace value
// - empty the status
func PrepareServiceForWorkPayload(service knativev1.Service) knativev1.Service {
	service.TypeMeta = metav1.TypeMeta{
		APIVersion: knativev1.SchemeGroupVersion.String(),
		Kind:       service.Kind,
	}
	service.Annotations[AnnotationKeyHubServiceUID] = string(service.UID)[0:5]
	service.ObjectMeta = metav1.ObjectMeta{
		Name:        service.Name,
		Namespace:   generateServiceNamespace(service),
		Labels:      service.Labels,
		Annotations: service.Annotations,
	}

	// empty the status
	service.Status = knativev1.ServiceStatus{}

	return service
}

// GenerateManifestWorkGeneric creates the ManifestWork that wraps object as payload
// With the status sync feedback of Workflow's phase
func GenerateManifestWorkGeneric(name, namespace string, obj client.Object, machineConfigOptions ...workv1.ManifestConfigOption) *workv1.ManifestWork {
	return &workv1.ManifestWork{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Annotations: map[string]string{AnnotationKeyHubServiceNamespace: obj.GetNamespace(),
				AnnotationKeyHubServiceName: obj.GetName()},
		},
		Spec: workv1.ManifestWorkSpec{
			Workload: workv1.ManifestsTemplate{
				Manifests: []workv1.Manifest{{RawExtension: runtime.RawExtension{Object: obj}}},
			},
			ManifestConfigs: machineConfigOptions,
		},
	}
}

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

func GenerateFeedbackRule(workStatusName, path string) workv1.FeedbackRule {
	return workv1.FeedbackRule{
		Type: workv1.JSONPathsType, JsonPaths: []workv1.JsonPath{{Name: workStatusName, Path: path}},
	}
}

func GenerateNamespace(name string) corev1.Namespace {
	return corev1.Namespace{
		TypeMeta:   metav1.TypeMeta{Kind: "Namespace", APIVersion: corev1.SchemeGroupVersion.Version},
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       corev1.NamespaceSpec{},
		Status:     corev1.NamespaceStatus{},
	}
}
