package status_utils

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rcsv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// This module provides utility functions for setting conditions on the status of a Custom Resource Definition (CRD)
//object of type rcsv1alpha1.Capp. It allows for setting conditions related to volumes availability, placement, and deployment status.

var (
	condVolumesType = "VolumesAvilable"
)

var statusBoolean = map[bool]string{
	true:  "True",
	false: "False",
}

// This function is used internally to set the specified condition on the status
// of the given capp object. It uses the provided client.Client to update the
// status of the object in the Kubernetes API server. If an error occurs during
// the update, the function returns an error.
func setCappCondition(capp rcsv1alpha1.Capp, ctx context.Context, r client.Client, l logr.Logger, condition metav1.Condition) error {
	if meta.IsStatusConditionPresentAndEqual(capp.Status.Conditions, condition.Type, condition.Status) {
		meta.SetStatusCondition(&capp.Status.Conditions, condition)
	}
	if err := r.Status().Update(ctx, &capp); err != nil {
		l.Error(err, fmt.Sprintf("Unable to set Capp condition %v", condition.Type))
		return err
	}
	return nil
}

// This function generates a new metav1.Condition object with the specified
// condType and status, along with an optional reason. If no reason is specified,
// it defaults to "unknown".
func generateCondtion(condType string, status bool, reason ...string) metav1.Condition {
	if len(reason) == 0 {
		reason[0] = "unknown"
	}
	condition := metav1.Condition{
		Type:   condType,
		Status: metav1.ConditionStatus(statusBoolean[status]),
		Reason: reason[0],
	}
	return condition
}

// SetVolumesCondition sets the "VolumesAvailable" condition on the status of the
// given capp object to the specified status, along with an optional reason. It
// uses the setCappCondition function internally to update the status of the
// object.
func SetVolumesCondition(capp rcsv1alpha1.Capp, ctx context.Context, r client.Client, l logr.Logger, status bool, reason ...string) error {
	condition := generateCondtion(condVolumesType, status, reason...)
	if err := setCappCondition(capp, ctx, r, l, condition); err != nil {
		return fmt.Errorf("Failed to set volumeCondition to capp %s", err)
	}
	return nil
}

// SetHasPlacementCondition sets the "HasPlacement" condition on the status of
// the given capp object to the specified status, along with an optional reason.
// It uses the setCappCondition function internally to update the status of the
// object.
func SetHasPlacementCondition(capp rcsv1alpha1.Capp, ctx context.Context, r client.Client, l logr.Logger, status bool, reason ...string) error {
	condition := generateCondtion(condVolumesType, status, reason...)
	if err := setCappCondition(capp, ctx, r, l, condition); err != nil {
		return fmt.Errorf("Failed to set placementCondition to capp %s", err)
	}
	return nil
}

// SetDeployedCondition sets the "Deployed" condition on the status of the given
// capp object to the specified status, along with an optional reason. It uses
// the setCappCondition function internally to update the status of the object.
func SetDeployedCondition(capp rcsv1alpha1.Capp, ctx context.Context, r client.Client, l logr.Logger, status bool, reason ...string) error {
	condition := generateCondtion(condVolumesType, status, reason...)
	if err := setCappCondition(capp, ctx, r, l, condition); err != nil {
		return fmt.Errorf("Failed to set deployedCondition to capp %s", err)
	}
	return nil
}
