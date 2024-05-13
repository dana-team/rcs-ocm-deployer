package directors

import (
	"context"
	"fmt"

	cappv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
	builder "github.com/dana-team/rcs-ocm-deployer/internal/sync/builders"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	workv1 "open-cluster-management.io/api/work/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type VolumesDirector struct {
	Ctx           context.Context
	K8sclient     client.Client
	Log           logr.Logger
	EventRecorder record.EventRecorder
}

// AssembleManifests compiles a slice of manifests for secrets and config maps discovered from the capp spec.
func (d VolumesDirector) AssembleManifests(capp cappv1alpha1.Capp) ([]workv1.Manifest, error) {
	var manifests []workv1.Manifest
	configMapsNames, secretNames := getResourceVolumesFromContainerSpec(capp)
	configMapManifests, err := d.createConfigMapManifests(configMapsNames, capp.Namespace)
	if err != nil {
		return manifests, err
	}
	manifests = append(manifests, configMapManifests...)

	secretManifests, err := d.createSecretManifests(secretNames, capp.Namespace)
	if err != nil {
		return manifests, err
	}
	manifests = append(manifests, secretManifests...)
	return manifests, nil
}

// createConfigMapManifests generates a slice of workv1.Manifest objects for each ConfigMap in the provided list.
// It fetches each ConfigMap from the Kubernetes API server using the provided namespace and converts them into manifests.
// In case of any errors during fetching, it returns the already created manifests and the error.
func (d VolumesDirector) createConfigMapManifests(configMaps []string, namespace string) ([]workv1.Manifest, error) {
	var manifests []workv1.Manifest
	for _, resource := range configMaps {
		cm := v1.ConfigMap{}
		if err := d.K8sclient.Get(d.Ctx, types.NamespacedName{Name: resource, Namespace: namespace}, &cm); err != nil {
			return manifests, fmt.Errorf("unable to fetch ConfigMap from Capp spec: %v", err.Error())
		} else {
			cmManifest := builder.BuildConfigMap(cm)

			manifests = append(manifests, cmManifest)
		}
	}
	return manifests, nil
}

// createSecretManifests creates a slice of workv1.Manifest objects for each Secret specified in the list.
// It retrieves each Secret using the Kubernetes client based on the provided namespace and converts them into manifests.
// If unable to fetch a Secret, the function returns the manifests created so far along with the encountered error.
func (d VolumesDirector) createSecretManifests(secrets []string, namespace string) ([]workv1.Manifest, error) {
	var manifests []workv1.Manifest
	for _, secretName := range secrets {
		secret := v1.Secret{}
		if err := d.K8sclient.Get(d.Ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, &secret); err != nil {
			return manifests, fmt.Errorf("unable to fetch Secret from Capp spec: %v", err.Error())
		} else {
			cmManifest := builder.BuildSecret(secret)

			manifests = append(manifests, cmManifest)
		}
	}
	return manifests, nil
}

// gatherConfigEnvFrom extracts the names of ConfigMaps and Secrets referenced in the environment variables in the envFrom field of given container specs.
// These names are appended to the provided configMaps and secrets slices.
func gatherConfigEnvFrom(containers []v1.Container, configMaps []string, secrets []string) ([]string, []string) {
	for _, containerSpec := range containers {
		for _, resourceEnv := range containerSpec.EnvFrom {
			if resourceEnv.ConfigMapRef != nil {
				configMaps = append(configMaps, resourceEnv.ConfigMapRef.Name)
			}
			if resourceEnv.SecretRef != nil {
				secrets = append(secrets, resourceEnv.SecretRef.Name)
			}
		}
	}
	return configMaps, secrets
}

// gatherConfigValueFrom extracts the names of ConfigMaps and Secrets referenced in the environment variables in the valueFrom field of given container specs.
// These names are appended to the provided configMaps and secrets slices.
func gatherConfigValueFrom(containers []v1.Container, configMaps []string, secrets []string) ([]string, []string) {
	for _, containerSpec := range containers {
		for _, resourceEnv := range containerSpec.Env {
			if resourceEnv.ValueFrom != nil {
				if resourceEnv.ValueFrom.ConfigMapKeyRef != nil {
					configMaps = append(configMaps, resourceEnv.ValueFrom.ConfigMapKeyRef.Name)
				}
				if resourceEnv.ValueFrom.SecretKeyRef != nil {
					secrets = append(secrets, resourceEnv.ValueFrom.SecretKeyRef.Name)
				}
			}
		}
	}
	return configMaps, secrets
}

// gatherConfigVolumes scans a list of volumes and appends the names of any found ConfigMaps and Secrets to the respective provided slices.
// It is designed to aid in collecting configuration-related resources referenced in volume definitions.
func gatherConfigVolumes(volumes []v1.Volume, configMaps []string, secrets []string) ([]string, []string) {
	for _, volume := range volumes {
		if volume.ConfigMap != nil {
			configMaps = append(configMaps, volume.ConfigMap.Name)
		}
		if volume.Secret != nil {
			secrets = append(secrets, volume.Secret.SecretName)
		}
	}
	return configMaps, secrets
}

// getResourceVolumesFromContainerSpec extracts the names of ConfigMaps and Secrets referenced in a given capp's container specification.
// It consolidates ConfigMaps and Secrets from environment variables, volumes, image pull secrets, and TLS secrets.
func getResourceVolumesFromContainerSpec(capp cappv1alpha1.Capp) ([]string, []string) {
	var configMaps []string
	var secrets []string

	configMaps, secrets = gatherConfigEnvFrom(capp.Spec.ConfigurationSpec.Template.Spec.Containers, configMaps, secrets)
	configMaps, secrets = gatherConfigValueFrom(capp.Spec.ConfigurationSpec.Template.Spec.Containers, configMaps, secrets)
	configMaps, secrets = gatherConfigVolumes(capp.Spec.ConfigurationSpec.Template.Spec.Volumes, configMaps, secrets)

	for _, secret := range capp.Spec.ConfigurationSpec.Template.Spec.ImagePullSecrets {
		secrets = append(secrets, secret.Name)
	}

	if capp.Spec.RouteSpec.TlsSecret != "" {
		secrets = append(secrets, capp.Spec.RouteSpec.TlsSecret)
	}

	return configMaps, secrets
}
