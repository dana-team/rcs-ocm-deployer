package utils

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	workv1 "open-cluster-management.io/api/work/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	RbacObjectSuffix = "-logs-reader"
)

// convertManifestToUnstructred gets a manifest from a ManifestWork, which is a slice of bytes and returns it as an unstructred object
func convertManifestToUnstructred(manifest []byte) (unstructured.Unstructured, error) {
	unstructuredObj := &unstructured.Unstructured{}
	err := unstructuredObj.UnmarshalJSON(manifest)
	return *unstructuredObj, err
}

// IsObjInManifestWork checks if a given object is in the ManifestWork's manifests list
func IsObjInManifestWork(k8sClient client.Client, manifestWork workv1.ManifestWork, objName string, objNamespace string, object client.Object, kind string) (bool, error) {
	err := k8sClient.Get(context.Background(), client.ObjectKey{Name: objName, Namespace: objNamespace}, object)
	if err != nil {
		return false, err
	}
	for _, manifest := range manifestWork.Spec.Workload.Manifests {
		obj, err := convertManifestToUnstructred(manifest.Raw)
		if err != nil {
			return false, err
		} else {
			if obj.GetKind() == kind && obj.GetName() == object.GetName() && obj.GetNamespace() == object.GetNamespace() {
				return true, nil
			}
		}
	}
	return false, nil
}

// IsRbacObjInManifestWork checks if a given  role/rolebinding object is in the ManifestWork's manifests list
func IsRbacObjInManifestWork(k8sClient client.Client, manifestWork workv1.ManifestWork, cappName string, nsName string, kind string) bool {
	for _, manifest := range manifestWork.Spec.Workload.Manifests {
		obj, err := convertManifestToUnstructred(manifest.Raw)
		if err != nil {
			return false
		} else {
			if obj.GetKind() == kind && obj.GetName() == cappName+RbacObjectSuffix && obj.GetNamespace() == nsName {
				return true
			}
		}
	}
	return false
}

// GetCappFromManifestWork returns a Capp from its corresponding ManifestWork
func GetCappFromManifestWork(k8sClient client.Client, manifestWork workv1.ManifestWork) unstructured.Unstructured {
	for _, manifest := range manifestWork.Spec.Workload.Manifests {
		obj, err := convertManifestToUnstructred(manifest.Raw)
		if err != nil {
			return unstructured.Unstructured{}
		} else if obj.GetKind() == "Capp" {
			return obj
		}
	}
	return unstructured.Unstructured{}
}
