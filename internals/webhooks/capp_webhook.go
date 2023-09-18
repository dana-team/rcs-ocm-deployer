package webhooks

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	rcsv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type CappValidator struct {
	Client     client.Client
	Decoder    *admission.Decoder
	Placements []string
}

// +kubebuilder:webhook:path=/validate-capp,mutating=false,sideEffects=NoneOnDryRun,failurePolicy=fail,groups="rcs.dana.io",resources=capps,verbs=create;update,versions=v1alpha1,name=capp.validate.rcs.dana.io,admissionReviewVersions=v1;v1beta1

const ServingPath = "/validate-capp"

func (c *CappValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	logger := log.FromContext(ctx).WithValues("webhook", "capp Webhook", "Name", req.Name)
	logger.Info("Webhook request received")
	capp := rcsv1alpha1.Capp{}
	if err := c.Decoder.DecodeRaw(req.Object, &capp); err != nil {
		logger.Error(err, "could not decode capp object")
		return admission.Errored(http.StatusBadRequest, err)
	}
	return c.handle(ctx, req, capp)

}

func (c *CappValidator) handle(ctx context.Context, req admission.Request, capp rcsv1alpha1.Capp) admission.Response {
	if !isScaleMetricSupported(capp) {
		return admission.Denied(fmt.Sprintf("this scale metric %s is unsupported. the avilable options are %s", capp.Spec.ScaleMetric, strings.Join(SupportedScaleMetrics, ",")))
	}
	if !isSiteVaild(capp, c.Placements, c.Client, ctx) {
		return admission.Denied(fmt.Sprintf("this site %s is unsupported. Site field accepts either cluster name or placement name", capp.Spec.Site))
	}
	if errs := validateDomainName(capp.Spec.RouteSpec.Hostname); errs != nil {
		return admission.Denied(errs.Error())
	}
	if errs := validateTlsFields(capp); errs != nil {
		return admission.Denied(errs.Error())
	}
	if capp.Spec.LogSpec != (rcsv1alpha1.LogSpec{}) {
		if errs := validateLogSpec(capp.Spec.LogSpec); errs != nil {
			return admission.Denied(errs.Error())
		}
	}
	return admission.Allowed("")
}
