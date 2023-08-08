package webhooks

import (
	"context"
	"encoding/json"

	rcsv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"

	"net/http"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type DefaultMutator struct {
	Client  client.Client
	Decoder *admission.Decoder
}

// +kubebuilder:webhook:path=/mutate-defaults-capp,mutating=true,sideEffects=NoneOnDryRun,failurePolicy=fail,groups="rcs.dana.io",resources=capps,verbs=create;update,versions=v1alpha1,name=capp.mutate.rcs.dana.io,admissionReviewVersions=v1;v1beta1

const (
	DefaultsServingPath = "/mutate-defaults-capp"
	DefaultScaleMetric  = "cpu"
)

func (c *DefaultMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	logger := log.FromContext(ctx).WithValues("webhook", "capp defaults Webhook", "Name", req.Name)
	logger.Info("Webhook request received")
	capp := rcsv1alpha1.Capp{}
	if err := c.Decoder.DecodeRaw(req.Object, &capp); err != nil {
		logger.Error(err, "could not decode Capp object")
		return admission.Errored(http.StatusBadRequest, err)
	}
	c.handle(&capp)
	marshaledCapp, err := json.Marshal(capp)
	if err != nil {
		admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledCapp)

}

func (c *DefaultMutator) handle(capp *rcsv1alpha1.Capp) {
	if capp.Spec.ScaleMetric == "" {
		capp.Spec.ScaleMetric = DefaultScaleMetric
	}
}
