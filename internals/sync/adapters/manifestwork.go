package adapters

import (
	"context"
	"fmt"

	rcsv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
	"github.com/dana-team/rcs-ocm-deployer/internals/utils/events"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	workv1 "open-cluster-management.io/api/work/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	NamespaceManifestWorkPrefix = "mw-create-"
	CappNameKey                 = "rcs.dana.io/capp-name"
	CappNamespaceKey            = "rcs.dana.io/capp-namespace"
)

// GenerateManifestWorkGeneric generates a new Kubernetes manifest work object
// with the specified name, namespace, and manifests. It takes an optional list
// of machine configuration options as well.
func GenerateManifestWorkGeneric(name string, namespace string, manifests []workv1.Manifest, machineConfigOptions ...workv1.ManifestConfigOption) *workv1.ManifestWork {
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

// GenerateMWName returns a manifestWork name combining NamespaceManifestWorkPrefix, capp namespace and name.
func GenerateMWName(capp rcsv1alpha1.Capp) string {
	return NamespaceManifestWorkPrefix + capp.Namespace + "-" + capp.Name
}

// CreateManifestWork uses the Kubernetes client to create a ManifestWork resource
// from the specified capp, cluster name, manifests, and logs the process.
func CreateManifestWork(capp rcsv1alpha1.Capp, managedClusterName string, logger logr.Logger, client client.Client, ctx context.Context, e record.EventRecorder, manifests []workv1.Manifest) error {
	mwName := GenerateMWName(capp)
	mw := GenerateManifestWorkGeneric(GenerateMWName(capp), managedClusterName, manifests, workv1.ManifestConfigOption{})
	SetManifestWorkCappAnnotations(*mw, capp)
	if err := client.Create(ctx, mw); err != nil {
		e.Event(&capp, corev1.EventTypeWarning, events.EventCappManifestWorkCreationFailed, err.Error())
		return fmt.Errorf("failed to create ManifestWork: %v", err.Error())
	}
	logger.Info(fmt.Sprintf("Created ManifestWork %q for Capp %q", mwName, capp.Name))
	e.Event(&capp, corev1.EventTypeNormal, events.EventCappManifestWorkCreated, fmt.Sprintf("Created ManifestWork %q for Capp %q", mwName, capp.Name))
	return nil
}

// SetManifestWorkCappAnnotations sets the annotations of the specified manifest
// work object with the name and namespace of the specified Capp object.
func SetManifestWorkCappAnnotations(mw workv1.ManifestWork, capp rcsv1alpha1.Capp) {
	mw.ObjectMeta.Annotations = make(map[string]string)
	mw.Annotations[CappNameKey] = capp.Name
	mw.Annotations[CappNamespaceKey] = capp.Namespace
}
