package utils

import (
	"context"
	"fmt"

	rcsv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	v1 "open-cluster-management.io/api/work/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const FinalizerCleanupCapp = "dana.io/capp-cleanup"

// HandleResourceDeletion handles the deletion of a Capp custom resource. It checks if the resource has a deletion timestamp
// and contains the specified finalizer. If so, it finalizes the service by cleaning up associated resources.
// It removes the finalizer once cleanup is complete and updates the resource.
func HandleResourceDeletion(ctx context.Context, capp rcsv1alpha1.Capp, log logr.Logger, r client.Client) error {
	if controllerutil.ContainsFinalizer(&capp, FinalizerCleanupCapp) {
		mwName := NamespaceManifestWorkPrefix + capp.Namespace + "-" + capp.Name
		if err := finalizeService(ctx, mwName, capp.Status.ApplicationLinks.Site, log, r); err != nil {
			if errors.IsNotFound(err) {
				return removeFinalizer(ctx, capp, log, r)
			}
			return err
		}
	}
	return nil
}

func removeFinalizer(ctx context.Context, capp rcsv1alpha1.Capp, log logr.Logger, r client.Client) error {
	log.Info("Already deleted ManifestWork, commit the Workflow finalizer removal")
	controllerutil.RemoveFinalizer(&capp, FinalizerCleanupCapp)
	if err := r.Update(ctx, &capp); err != nil {
		return fmt.Errorf("Failed to remove finalizer from capp %s", err.Error())
	}
	return nil
}

// FinalizeService deletes the ManifestWork associated with the service on the specified managed cluster.
// The function gets the context, manifest work name, managed cluster name, and logger.
func finalizeService(ctx context.Context, mwName string, managedClusterName string, log logr.Logger, r client.Client) error {
	// delete the ManifestWork associated with this service
	var work v1.ManifestWork
	if err := r.Get(ctx, types.NamespacedName{Name: mwName, Namespace: managedClusterName}, &work); err != nil {
		return err
	}
	if err := r.Delete(ctx, &work); err != nil {
		return fmt.Errorf("Unable to delete ManifestWork %s", err.Error())
	}
	return nil
}

// EnsureFinalizer ensures the service has the finalizer specified (FinalizerCleanupCapp).
func EnsureFinalizer(ctx context.Context, service rcsv1alpha1.Capp, r client.Client) error {
	if !controllerutil.ContainsFinalizer(&service, FinalizerCleanupCapp) {
		controllerutil.AddFinalizer(&service, FinalizerCleanupCapp)
		if err := r.Update(ctx, &service); err != nil {
			return fmt.Errorf("Failed to add finalizer to capp %s", err.Error())
		}
	}
	return nil
}
