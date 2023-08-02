package utils

import (
	"context"
	"fmt"

	rcsv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	v1 "open-cluster-management.io/api/work/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// The prepareVolumesManifests function takes in a list of ConfigMaps and Secrets, a Capp object, a context.Context, a logr.Logger, and a Kubernetes client object.
// It returns a list of v1.Manifest objects and an error. This function fetches the ConfigMaps and Secrets specified in the input parameters using the Kubernetes client object and creates a corev1.ConfigMap or corev1.Secret object for each of them.
// It then creates a v1.Manifest object for each of these resources by using the runtime.RawExtension to store the corev1.ConfigMap or corev1.Secret object.
func prepareVolumesManifests(secrets []string, configMaps []string, capp rcsv1alpha1.Capp, ctx context.Context, l logr.Logger, r client.Client) ([]v1.Manifest, error) {
	resources := []v1.Manifest{}
	for _, resource := range configMaps {
		cm := corev1.ConfigMap{}
		if err := r.Get(ctx, types.NamespacedName{Name: resource, Namespace: capp.Namespace}, &cm); err != nil {
			return resources, fmt.Errorf("Unable to fetch ConfigMap from capp spec %s", err.Error())
		} else {
			cmManifest := &corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      cm.Name,
					Namespace: cm.Namespace,
				},
				Data: cm.Data,
			}
			resources = append(resources, v1.Manifest{RawExtension: runtime.RawExtension{Object: cmManifest}})
		}
	}
	for _, resource := range secrets {
		secret := &corev1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{Name: resource, Namespace: capp.Namespace}, secret); err != nil {
			return resources, fmt.Errorf("Unable to fetch Secret from capp spec %s", err.Error())
		} else {
			secretManifest := &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      secret.Name,
					Namespace: secret.Namespace,
				},
				Data: secret.Data,
			}
			resources = append(resources, v1.Manifest{RawExtension: runtime.RawExtension{Object: secretManifest}})
		}
	}
	return resources, nil
}

// GetResourceVolumesFromContainerSpec takes in a Capp object, a context.Context,
// a logr.Logger, and a Kubernetes client object. It returns two lists of
// strings, one for ConfigMaps and one for Secrets. This function iterates over
// the ContainerSpec objects specified in the Capp object and extracts the names
// of any ConfigMaps or Secrets specified in their EnvFrom fields. Additionally,
// it extracts the names of secrets located in the imagePullSecrets field. It
// also iterates over the Volumes specified in the Capp object and extracts the
// names of any ConfigMaps or Secrets specified in them.
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
	for _, secret := range capp.Spec.ConfigurationSpec.Template.Spec.ImagePullSecrets {
		secrets = append(secrets, secret.Name)
	}

	return configMaps, secrets
}
