// Package utils provides utility functions for working with Kubernetes resources and custom resources defined in the
// container-app-operator API.
package utils

import (
	"context"
	"fmt"

	rcsv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	v1 "open-cluster-management.io/api/work/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// AnnotationKeyHasPlacement is the key used to store the managed cluster name in an annotation on a Capp resource.
const AnnotationKeyHasPlacement = "dana.io/has-placement"

// ContainesPlacementAnnotation checks if a Capp resource has an annotation indicating it has been placed on a managed cluster.
func ContainesPlacementAnnotation(capp rcsv1alpha1.Capp) bool {
	annos := capp.GetAnnotations()
	if len(annos) == 0 {
		return false
	}

	namespace, ok := annos[AnnotationKeyHasPlacement]
	return ok && len(namespace) > 0
}

// PrepareServiceForWorkPayload prepares a Capp resource for inclusion in a manifest work by setting its TypeMeta and ObjectMeta.
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

// GenerateNamespace generates a corev1.Namespace object with the specified name.
func GenerateNamespace(name string) corev1.Namespace {
	return corev1.Namespace{
		TypeMeta:   metav1.TypeMeta{Kind: "Namespace", APIVersion: corev1.SchemeGroupVersion.Version},
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       corev1.NamespaceSpec{},
		Status:     corev1.NamespaceStatus{},
	}
}

// GatherCappResources gathers all the Kubernetes resources required to deploy a Capp resource and returns them as an array of manifests.
func GatherCappResources(capp rcsv1alpha1.Capp, ctx context.Context, l logr.Logger, r client.Client) ([]v1.Manifest, error) {
	manifests := []v1.Manifest{}
	svc := PrepareServiceForWorkPayload(capp)
	ns := GenerateNamespace(capp.Namespace)
	manifests = append(manifests, v1.Manifest{RawExtension: runtime.RawExtension{Object: &svc}}, v1.Manifest{RawExtension: runtime.RawExtension{Object: &ns}})
	configMaps, secrets := GetResourceVolumesFromContainerSpec(capp, ctx, l, r)
	cmAndSecrets, err := prepareVolumesManifests(secrets, configMaps, capp, ctx, l, r)
	if err != nil {
		return manifests, fmt.Errorf("failed to prepare volumes for Capp: %s", err.Error())
	}
	manifests = append(manifests, cmAndSecrets...)
	role, rb, err := PrepareAdminsRolesForCapp(ctx, r, capp)
	if err != nil {
		return manifests, fmt.Errorf("failed to prepare Roles and roleBindings for Capp: %s", err.Error())
	}
	manifests = append(manifests, v1.Manifest{RawExtension: runtime.RawExtension{Object: &role}}, v1.Manifest{RawExtension: runtime.RawExtension{Object: &rb}})
	return manifests, nil
}

// UpdateCappDestination updates the Site field in the Status.ApplicationLinks object of a Capp custom resource.
// The Site field specifies the managed cluster name where the application is running.
// This function also calls the AddCappHasPlacementAnnotation function to add an annotation to the Capp resource that indicates the placement of the application.
func UpdateCappDestination(capp rcsv1alpha1.Capp, managedClusterName string, ctx context.Context, r client.Client) error {
	capp.Status.ApplicationLinks.Site = managedClusterName
	if err := r.Status().Update(ctx, &capp); err != nil {
		return fmt.Errorf("failed to update Capp status with selected site: %s", err.Error())
	}
	if err := AddCappHasPlacementAnnotation(capp, managedClusterName, ctx, r); err != nil {
		return err
	}
	return nil
}

// AddCappHasPlacementAnnotation adds an annotation to the Capp custom resource that indicates the managed cluster where the application is placed.
func AddCappHasPlacementAnnotation(capp rcsv1alpha1.Capp, managedClusterName string, ctx context.Context, r client.Client) error {
	cappAnno := capp.GetAnnotations()
	if cappAnno == nil {
		cappAnno = make(map[string]string)
	}
	cappAnno[AnnotationKeyHasPlacement] = managedClusterName
	capp.SetAnnotations(cappAnno)
	return r.Update(ctx, &capp)
}

// RemoveCreatedAnnotation removes an annotation from the Capp custom resource that indicates the namespace where the application is created.
func RemoveCreatedAnnotation(ctx context.Context, service rcsv1alpha1.Capp, r client.Client) error {
	cappAnno := service.GetAnnotations()
	delete(cappAnno, "AnnotationNamespaceCreated")
	service.SetAnnotations(cappAnno)
	if err := r.Update(ctx, &service); err != nil {
		return fmt.Errorf("failed to update Capp status with annotations: %s", err.Error())
	}
	return nil
}
