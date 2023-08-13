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
// and contains the specified finalizer. If so, it finalizes the Capp by cleaning up associated resources.
// It removes the finalizer once cleanup is complete and updates the resource.
func HandleResourceDeletion(ctx context.Context, capp rcsv1alpha1.Capp, log logr.Logger, r client.Client) error {
	if controllerutil.ContainsFinalizer(&capp, FinalizerCleanupCapp) {
		mwName := NamespaceManifestWorkPrefix + capp.Namespace + "-" + capp.Name
		if err := finalizeCapp(ctx, mwName, capp.Status.ApplicationLinks.Site, log, r); err != nil {
			if errors.IsNotFound(err) {
				return removeFinalizer(ctx, capp, log, r)
			}
			return err
		}
	}
	return nil
}

func removeFinalizer(ctx context.Context, capp rcsv1alpha1.Capp, log logr.Logger, r client.Client) error {
	log.Info("already deleted ManifestWork, commit the Workflow finalizer removal")
	controllerutil.RemoveFinalizer(&capp, FinalizerCleanupCapp)
	if err := r.Update(ctx, &capp); err != nil {
		return fmt.Errorf("failed to remove finalizer from Capp: %s", err.Error())
	}
	return nil
}

// finalizeCapp deletes the ManifestWork associated with the Capp on the specified managed cluster.
// The function gets the context, manifest work name, managed cluster name, and logger.
func finalizeCapp(ctx context.Context, mwName string, managedClusterName string, log logr.Logger, r client.Client) error {
	var work v1.ManifestWork
	if err := r.Get(ctx, types.NamespacedName{Name: mwName, Namespace: managedClusterName}, &work); err != nil {
		return err
	}
	if err := r.Delete(ctx, &work); err != nil {
		return fmt.Errorf("unable to delete ManifestWork: %s", err.Error())
	}
	return nil
}

// EnsureFinalizer ensures the Capp has the finalizer specified (FinalizerCleanupCapp).
func EnsureFinalizer(ctx context.Context, capp rcsv1alpha1.Capp, r client.Client) error {
	if !controllerutil.ContainsFinalizer(&capp, FinalizerCleanupCapp) {
		controllerutil.AddFinalizer(&capp, FinalizerCleanupCapp)
		if err := r.Update(ctx, &capp); err != nil {
			return fmt.Errorf("failed to add finalizer to Capp: %s", err.Error())
		}
	}
	return nil
}
