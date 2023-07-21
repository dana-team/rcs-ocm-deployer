package utils

import (
	"context"

	rcsv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	workv1 "open-cluster-management.io/api/work/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	NamespaceManifestWorkPrefix = "mw-create-"
	CappNameKey                 = "dana.io/capp-name"
	CappNamespaceKey            = "dana.io/capp-namespace"
)

// GenerateManifestWorkGeneric generates a new Kubernetes manifest work object
// with the specified name, namespace, and manifests. It takes an optional list
// of machine configuration options as well.
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

// SetManifestWorkCappAnnotations sets the annotations of the specified manifest
// work object with the name and namespace of the specified Capp object.
func SetManifestWorkCappAnnotations(mw workv1.ManifestWork, capp rcsv1alpha1.Capp) {
	mw.ObjectMeta.Annotations = make(map[string]string)
	mw.Annotations[CappNameKey] = capp.Name
	mw.Annotations[CappNamespaceKey] = capp.Namespace
}

// GenerateManifestConfigOption generates a new ManifestConfigOption object,
// which represents a configuration option that can be associated with a manifest
// work. The function takes a Kubernetes object, a resource name, a group name,
// and a list of feedback rules.
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

// GetRelatedManifestwork retrieves the manifest work object
// related to the specified Capp by name and namespace. It takes a context, a
// client, a logger, and the Capp object itself, and returns the related manifest
// work object or an error if it cannot be retrieved.
func GetRelatedManifestwork(ctx context.Context, r client.Client, l logr.Logger, capp rcsv1alpha1.Capp) (workv1.ManifestWork, error) {
	mw := workv1.ManifestWork{}
	mwName := NamespaceManifestWorkPrefix + capp.Namespace + "-" + capp.Name
	if err := r.Get(ctx, types.NamespacedName{Name: mwName, Namespace: capp.ObjectMeta.Annotations[AnnotationKeyHasPlacement]}, &mw); err != nil {
		return mw, err
	}
	return mw, nil
}
