package e2e_tests

import (
	"fmt"

	"github.com/dana-team/rcs-ocm-deployer/internal/webhooks"
	mock "github.com/dana-team/rcs-ocm-deployer/test/e2e_tests/mocks"
	utilst "github.com/dana-team/rcs-ocm-deployer/test/e2e_tests/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	adminAnnotationValue = "kubernetes-admin"
)

var _ = Describe("Validate the mutating webhook", func() {
	It("Should add annotation on create", func() {
		baseCapp := mock.CreateBaseCapp()
		capp := utilst.CreateCapp(k8sClient, baseCapp)

		annotation := capp.ObjectMeta.Annotations[webhooks.LastUpdatedByAnnotationKey]
		Expect(annotation).To(Equal(adminAnnotationValue))
	})

	It("Should add annotation on update", func() {
		baseCapp := mock.CreateBaseCapp()
		capp := utilst.CreateCapp(k8sClient, baseCapp)

		annotation := capp.ObjectMeta.Annotations[webhooks.LastUpdatedByAnnotationKey]
		Expect(annotation).To(Equal(adminAnnotationValue))

		utilst.SwitchUser(&k8sClient, cfg, mock.NSName, newScheme(), true)
		capp = utilst.GetCapp(k8sClient, capp.Name, capp.Namespace)
		capp.ObjectMeta.Annotations["test"] = "test"
		utilst.UpdateCapp(k8sClient, capp)

		updatedCapp := utilst.GetCapp(k8sClient, capp.Name, capp.Namespace)

		// Check if the annotation has changed
		updatedAnnotation := updatedCapp.ObjectMeta.Annotations[webhooks.LastUpdatedByAnnotationKey]
		Expect(updatedAnnotation).To(Equal(fmt.Sprintf(utilst.ServiceAccountNameFormat, mock.NSName, utilst.ServiceAccountName)))
	})

	AfterEach(func() {
		// Revert k8sClient back to use the original configuration
		utilst.SwitchUser(&k8sClient, cfg, mock.NSName, newScheme(), false)
	})
})
