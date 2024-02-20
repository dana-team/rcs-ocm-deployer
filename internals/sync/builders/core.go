package builders

import (
	rcsv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	workv1 "open-cluster-management.io/api/work/v1"
)

// BuildCapp prepares a Capp resource for inclusion in a manifest work by setting its TypeMeta and ObjectMeta.
func BuildCapp(capp rcsv1alpha1.Capp) workv1.Manifest {
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

	return workv1.Manifest{RawExtension: runtime.RawExtension{Object: &capp}}
}

// BuildNamespace generates a corev1.Namespace object with the specified name.
func BuildNamespace(name string) workv1.Manifest {
	namespace := corev1.Namespace{
		TypeMeta:   metav1.TypeMeta{Kind: "Namespace", APIVersion: corev1.SchemeGroupVersion.Version},
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
	return workv1.Manifest{RawExtension: runtime.RawExtension{Object: &namespace}}
}
