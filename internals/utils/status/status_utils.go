package status_utils

// utility package for retrieving and updating status information related to a Capp (Container App).

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	utils "github.com/dana-team/rcs-ocm-deployer/internals/utils"
	"knative.dev/pkg/apis"
	v1 "knative.dev/pkg/apis/duck/v1"
	knativev1 "knative.dev/serving/pkg/apis/serving/v1"

	rcsv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	workv1 "open-cluster-management.io/api/work/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// This function takes a Capp object, a ManifestWork object, a resource name, and a resource kind as arguments.
// It looks for a manifest with the specified resource name and kind in the ManifestWork object's Status.ResourceStatus.Manifests slice and returns the StatusFeedbacks.Values slice of that manifest.
func GetStatusFromMwByResource(capp rcsv1alpha1.Capp, mw workv1.ManifestWork, resourceName string, resourceKind string) []workv1.FeedbackValue {
	for _, manifest := range mw.Status.ResourceStatus.Manifests {
		if manifest.ResourceMeta.Kind == resourceKind && manifest.ResourceMeta.Name == resourceName {
			return manifest.StatusFeedbacks.Values
		}
	}
	return nil
}

// This function takes a slice of FeedbackValues and a field name as arguments.
// It looks for a FeedbackValue with the specified field name in the slice, and returns the value of the corresponding field as a string.
// The field can be of type String, Integer, or Boolean.
func GetFieldFromFeedbackRules(feedbacks []workv1.FeedbackValue, fieldName string) string {
	for _, feedback := range feedbacks {
		if fieldName != feedback.Name {
			continue
		}
		switch valueType := feedback.Value.Type; valueType {
		case "String":
			return fmt.Sprintf("%v", *feedback.Value.String)
		case "Integer":
			return fmt.Sprintf("%v", *feedback.Value.Integer)
		case "Boolean":
			return fmt.Sprintf("%v", *feedback.Value.Boolean)
		}
	}
	return ""
}

// This function takes a context, a Kubernetes client, a logger, and a Capp object as arguments.
// It first retrieves the ManifestWork object related to the Capp
// It updates the Capp's status fields based on the values retrieved from the FeedbackValues.
func SyncStatusFromMW(ctx context.Context, r client.Client, l logr.Logger, capp rcsv1alpha1.Capp) error {
	mw, err := utils.GetRelatedManifestwork(ctx, r, l, capp)
	if err != nil {
		return err
	}
	feedbackRules := GetStatusFromMwByResource(capp, mw, capp.Name, "Capp")
	if err := r.Get(ctx, types.NamespacedName{Name: capp.Name, Namespace: capp.Namespace}, &capp); err != nil {
		return err
	}
	cappStatus := &capp.Status
	cappStatus.ApplicationLinks.ClusterSegment = GetFieldFromFeedbackRules(feedbackRules, "clusterSegment")
	cappStatus.ApplicationLinks.ConsoleLink = GetFieldFromFeedbackRules(feedbackRules, "consoleLink")
	cappStatus.ApplicationLinks.Site = GetFieldFromFeedbackRules(feedbackRules, "site")
	cappStatus.KnativeObjectStatus = knativev1.ServiceStatus{}
	cappStatus.KnativeObjectStatus.Address = &v1.Addressable{}
	capp.Status.KnativeObjectStatus.Address.URL = apis.HTTP(strings.Split(GetFieldFromFeedbackRules(feedbackRules, "url"), "://")[1])
	cappStatus.KnativeObjectStatus.LatestCreatedRevisionName = GetFieldFromFeedbackRules(feedbackRules, "latestCreatedRevisionName")
	cappStatus.KnativeObjectStatus.LatestReadyRevisionName = GetFieldFromFeedbackRules(feedbackRules, "latestReadyRevisionName")
	if reflect.DeepEqual(cappStatus, capp.Status) {
		return nil
	}
	if err := r.Status().Update(ctx, &capp); err != nil {
		fmt.Print(err)
		return err
	}
	return nil
}
