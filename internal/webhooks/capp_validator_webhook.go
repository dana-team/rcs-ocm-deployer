package webhooks

import (
	"context"
	"fmt"
	"net/http"

	rcsv1alpha1 "github.com/dana-team/rcs-ocm-deployer/api/v1alpha1"
	"github.com/dana-team/rcs-ocm-deployer/internal/utils"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	cappv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type CappValidator struct {
	Client  client.Client
	Decoder *admission.Decoder
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
	return c.handle(ctx, capp)
}

func (c *CappValidator) handle(ctx context.Context, capp cappv1alpha1.Capp) admission.Response {
	config, err := c.getRCSConfig(ctx)
	if err != nil {
		if statusError, isStatusError := err.(*errors.StatusError); isStatusError {
			if statusError.ErrStatus.Reason == metav1.StatusReasonNotFound {
				c.Log.Error(err, "rcs config has not been defined")
			}
		} else {
			c.Log.Error(err, "failed to fetch rcs config")
		}
		return admission.Denied("Failed to fetch RCSConfig")
	}
	placements := config.Spec.Placements
	if !isSiteValid(capp, placements, c.Client, ctx) {
		return admission.Denied(fmt.Sprintf("this site %s is unsupported. Site field accepts either cluster name or placement name", capp.Spec.Site))
	}
	if errs := validateDomainName(capp.Spec.RouteSpec.Hostname); errs != nil {
		return admission.Denied(errs.Error())
	}
	if capp.Spec.LogSpec != (cappv1alpha1.LogSpec{}) {
		if errs := validateLogSpec(capp.Spec.LogSpec); errs != nil {
			return admission.Denied(errs.Error())
		}
	}
	return admission.Allowed("")
}

func (c *CappValidator) getRCSConfig(ctx context.Context) (*rcsv1alpha1.RCSConfig, error) {
	config := rcsv1alpha1.RCSConfig{}
	key := types.NamespacedName{Name: utils.RcsConfigName, Namespace: utils.RcsConfigNamespace}
	if err := c.Client.Get(ctx, key, &config); err != nil {
		return nil, err
	}
	return &config, nil
}
