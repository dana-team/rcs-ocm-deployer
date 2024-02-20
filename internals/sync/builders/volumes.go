package builders

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	workv1 "open-cluster-management.io/api/work/v1"
)

// BuildConfigMap generates a Manifest with a configMap
func BuildConfigMap(cm corev1.ConfigMap) workv1.Manifest {
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
	return workv1.Manifest{RawExtension: runtime.RawExtension{Object: cmManifest}}
}

// BuildSecret generates manifest that contains a secret.
func BuildSecret(secret corev1.Secret) workv1.Manifest {
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
		Type: secret.Type,
	}
	return workv1.Manifest{RawExtension: runtime.RawExtension{Object: secretManifest}}
}
