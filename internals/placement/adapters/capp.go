package adapters

import (
	"context"
	"fmt"

	rcsv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// AnnotationKeyHasPlacement is the key used to store the managed cluster name in an annotation on a Capp resource
	AnnotationKeyHasPlacement = "rcs.dana.io/has-placement"
)

// UpdateCappDestination updates the Site field in the Status.ApplicationLinks object of a Capp custom resource.
// The Site field specifies the managed cluster name where the application is running.
// This function also calls the AddCappHasPlacementAnnotation function to add an annotation to the Capp resource that indicates the placement of the application.
func UpdateCappDestination(capp rcsv1alpha1.Capp, managedClusterName string, ctx context.Context, r client.Client) error {
	capp.Status.ApplicationLinks.Site = managedClusterName
	if err := r.Status().Update(ctx, &capp); err != nil {
		return fmt.Errorf("failed to update Capp status with selected site: %s", err.Error())
	}
	if err := AddCappHasPlacementAnnotation(capp, managedClusterName, ctx, r); err != nil {
		return err
	}
	return nil
}

// AddCappHasPlacementAnnotation adds an annotation to the Capp custom resource that indicates the managed cluster where the application is placed.
func AddCappHasPlacementAnnotation(capp rcsv1alpha1.Capp, managedClusterName string, ctx context.Context, r client.Client) error {
	cappAnno := capp.GetAnnotations()
	if cappAnno == nil {
		cappAnno = make(map[string]string)
	}
	cappAnno[AnnotationKeyHasPlacement] = managedClusterName
	capp.SetAnnotations(cappAnno)
	return r.Update(ctx, &capp)
}
