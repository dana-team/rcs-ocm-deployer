package directors

import (
	"context"

	cappv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
	builder "github.com/dana-team/rcs-ocm-deployer/internals/sync/builders"
	"github.com/dana-team/rcs-ocm-deployer/internals/utils/events"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	workv1 "open-cluster-management.io/api/work/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CappDirector struct {
	Ctx           context.Context
	K8sclient     client.Client
	Log           logr.Logger
	EventRecorder record.EventRecorder
}

// AssembleManifests compiles a comprehensive list of mw1.Manifest objects necessary for deploying a given capp.
// It begins by building basic capp and namespace manifests, then delegates to the VolumesDirector and AuthDirector
// to gather additional manifests related to volumes and authentication respectively. In case of errors in assembling
// volume or authentication manifests, it records an event and returns the encountered error.
// This method provides a central point for collating all necessary Kubernetes manifests for a capp deployment.
func (d CappDirector) AssembleManifests(capp cappv1alpha1.Capp) ([]workv1.Manifest, error) {
	manifests := []workv1.Manifest{builder.BuildCapp(capp), builder.BuildNamespace(capp.Namespace)}
	volumesDirector := VolumesDirector(d)
	volumesManifests, err := volumesDirector.AssembleManifests(capp)
	if err != nil {
		d.EventRecorder.Event(&capp, corev1.EventTypeWarning, events.EventCappVolumeNotFound, err.Error())
		return []workv1.Manifest{}, err
	}
	manifests = append(manifests, volumesManifests...)

	authDirector := AuthDirector(d)
	authManifests, err := authDirector.AssembleManifests(capp)
	if err != nil {
		d.EventRecorder.Event(&capp, corev1.EventTypeWarning, events.EventCappAuthFailed, err.Error())
		return []workv1.Manifest{}, err
	}
	manifests = append(manifests, authManifests...)
	return manifests, nil
}
