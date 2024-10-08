package webhooks

import (
	"context"
	"fmt"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"

	cappv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type CappValidator struct {
	Client  client.Client
	Decoder admission.Decoder
	Log     logr.Logger
}

// +kubebuilder:webhook:path=/validate-capp,mutating=false,sideEffects=NoneOnDryRun,failurePolicy=fail,groups="rcs.dana.io",resources=capps,verbs=create;update,versions=v1alpha1,name=capp.validate.rcs.dana.io,admissionReviewVersions=v1;v1beta1

const ValidatorServingPath = "/validate-capp"

func (c *CappValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	logger := log.FromContext(ctx).WithValues("webhook", "capp Webhook", "Name", req.Name)
	logger.Info("Webhook request received")

	capp := cappv1alpha1.Capp{}
	if err := c.Decoder.DecodeRaw(req.Object, &capp); err != nil {
		logger.Error(err, "could not decode capp object")
		return admission.Errored(http.StatusBadRequest, err)
	}

	oldCapp := &cappv1alpha1.Capp{}
	if req.Operation == admissionv1.Update {
		err := c.Decoder.DecodeRaw(req.OldObject, oldCapp)
		if err != nil {
			logger.Error(err, "could not decode old capp object")
			return admission.Errored(http.StatusBadRequest, err)
		}
	}

	return c.handle(ctx, capp, oldCapp)
}

func (c *CappValidator) handle(ctx context.Context, capp cappv1alpha1.Capp, oldCapp *cappv1alpha1.Capp) admission.Response {
	config, err := getRCSConfig(ctx, c.Client)
	if err != nil {
		return admission.Denied("Failed to fetch RCSConfig")
	}

	placements := config.Spec.Placements
	if !isSiteValid(capp, placements, c.Client, ctx) {
		return admission.Denied(fmt.Sprintf("this site %s is unsupported. Site field accepts either cluster name or placement name", capp.Spec.Site))
	}

	var invalidHostnamePatterns []string
	if config.Spec.InvalidHostnamePatterns != nil {
		invalidHostnamePatterns = config.Spec.InvalidHostnamePatterns
	}

	if oldCapp == nil || capp.Spec.RouteSpec.Hostname != oldCapp.Spec.RouteSpec.Hostname {
		if errs := validateDomainName(capp.Spec.RouteSpec.Hostname, invalidHostnamePatterns); errs != nil {
			return admission.Denied(errs.Error())
		}
	}

	if capp.Spec.LogSpec != (cappv1alpha1.LogSpec{}) {
		if errs := validateLogSpec(capp.Spec.LogSpec); errs != nil {
			return admission.Denied(errs.Error())
		}
	}

	return admission.Allowed("")
}
