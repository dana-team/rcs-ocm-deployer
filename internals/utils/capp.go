package utils

import (
	"context"

	rcsv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	knativev1 "knative.dev/serving/pkg/apis/serving/v1"

	v1 "open-cluster-management.io/api/work/v1"
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

// GenerateNamespace generates namespace from given names
func GenerateNamespace(name string) corev1.Namespace {
	return corev1.Namespace{
		TypeMeta:   metav1.TypeMeta{Kind: "Namespace", APIVersion: corev1.SchemeGroupVersion.Version},
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       corev1.NamespaceSpec{},
		Status:     corev1.NamespaceStatus{},
	}
}

func GatherCappResources(capp rcsv1alpha1.Capp, ctx context.Context, l logr.Logger, r client.Client) ([]v1.Manifest, error) {
	manifests := []v1.Manifest{}
	svc := PrepareServiceForWorkPayload(capp)
	ns := GenerateNamespace(capp.Namespace)
	manifests = append(manifests, v1.Manifest{RawExtension: runtime.RawExtension{Object: &svc}}, v1.Manifest{RawExtension: runtime.RawExtension{Object: &ns}})
	configMaps, secrets := GetResourceVolumesFromContainerSpec(capp, ctx, l, r)
	cmAndSecrets, err := prepareVolumesManifests(secrets, configMaps, capp, ctx, l, r)
	if err != nil {
		return manifests, err
	}
	manifests = append(manifests, cmAndSecrets...)
	return manifests, nil
}

func prepareVolumesManifests(secrets []string, configMaps []string, capp rcsv1alpha1.Capp, ctx context.Context, l logr.Logger, r client.Client) ([]v1.Manifest, error) {
	resources := []v1.Manifest{}
	for _, resource := range configMaps {
		cm := &corev1.ConfigMap{}
		if err := r.Get(ctx, types.NamespacedName{Name: resource, Namespace: capp.Namespace}, cm); err != nil {
			l.Error(err, "unable to fetch configmap")
			return resources, err
		} else {
			resources = append(resources, v1.Manifest{RawExtension: runtime.RawExtension{Object: cm}})
		}
	}
	for _, resource := range secrets {
		secret := &corev1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{Name: resource, Namespace: capp.Namespace}, secret); err != nil {
			l.Error(err, "unable to fetch secret")
			return resources, err
		} else {
			resources = append(resources, v1.Manifest{RawExtension: runtime.RawExtension{Object: secret}})
		}
	}
	return resources, nil
}

func UpdateCappDestination(capp rcsv1alpha1.Capp, managedClusterName string, ctx context.Context, r client.Client) error {
	capp.Status.ApplicationLinks.Site = managedClusterName
	if err := r.Status().Update(ctx, &capp); err != nil {
		return err
	}
	if err := AddCappHasPlacementAnnotation(capp, managedClusterName, ctx, r); err != nil {
		return err
	}
	return nil
}

func GetResourceVolumesFromContainerSpec(capp rcsv1alpha1.Capp, ctx context.Context, l logr.Logger, r client.Client) ([]string, []string) {
	var configMaps []string
	var secrets []string
	for _, containerSpec := range capp.Spec.ConfigurationSpec.Template.Spec.Containers {
		for _, resourceEnv := range containerSpec.EnvFrom {
			if resourceEnv.ConfigMapRef != nil {
				configMaps = append(configMaps, resourceEnv.ConfigMapRef.Name)
			}
			if resourceEnv.SecretRef != nil {
				secrets = append(secrets, resourceEnv.SecretRef.Name)
			}
		}
	}
	for _, volume := range capp.Spec.ConfigurationSpec.Template.Spec.Volumes {

		if volume.ConfigMap != nil {
			configMaps = append(configMaps, volume.ConfigMap.Name)
		}
		if volume.Secret != nil {
			secrets = append(secrets, volume.Secret.SecretName)
		}
	}

	return configMaps, secrets
}

func AddCappHasPlacementAnnotation(capp rcsv1alpha1.Capp, managedClusterName string, ctx context.Context, r client.Client) error {
	cappAnno := capp.GetAnnotations()
	if cappAnno == nil {
		cappAnno = make(map[string]string)
	}
	cappAnno[AnnotationKeyHasPlacement] = managedClusterName
	capp.SetAnnotations(cappAnno)
	return r.Update(ctx, &capp)
}

func RemoveCreatedAnnotation(ctx context.Context, service rcsv1alpha1.Capp, r client.Client) error {
	cappAnno := service.GetAnnotations()
	delete(cappAnno, "AnnotationNamespaceCreated")
	service.SetAnnotations(cappAnno)
	if err := r.Update(ctx, &service); err != nil {
		return err
	}
	return nil
}
