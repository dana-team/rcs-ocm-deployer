package webhooks

import (
	"context"
	"encoding/json"
	"net/http"

	cappv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type CappMutator struct {
	Decoder admission.Decoder
}

// +kubebuilder:webhook:path=/mutate-capp,mutating=true,sideEffects=NoneOnDryRun,failurePolicy=fail,groups=rcs.dana.io,resources=capps,verbs=create;update,versions=v1alpha1,name=capp.dana.io,admissionReviewVersions=v1;v1beta1

const (
	LastUpdatedByAnnotationKey = "rcs.dana.io/last-updated-by"
	MutatorServingPath         = "/mutate-capp"
)

// Handle implements the mutation webhook.
func (c *CappMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	logger := log.FromContext(ctx).WithValues("mutation webhook", "capp mutation Webhook", "Name", req.Name)
	capp := cappv1alpha1.Capp{}
	if err := c.Decoder.DecodeRaw(req.Object, &capp); err != nil {
		logger.Error(err, "could not decode capp object")
		return admission.Errored(http.StatusBadRequest, err)
	}

	c.handleInner(&capp, req)

	marshaledCapp, err := json.Marshal(capp)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledCapp)
}

// handleInner sets an annotation on the Capp object to track the last user who updated it.
func (c *CappMutator) handleInner(capp *cappv1alpha1.Capp, req admission.Request) {
	if capp.ObjectMeta.Annotations == nil {
		capp.ObjectMeta.Annotations = make(map[string]string)
	}

	capp.ObjectMeta.Annotations[LastUpdatedByAnnotationKey] = req.UserInfo.Username
}
