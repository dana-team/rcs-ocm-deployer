package e2e_tests

import (
	mock "github.com/dana-team/rcs-ocm-deployer/test/e2e_tests/mocks"
	"github.com/dana-team/rcs-ocm-deployer/test/e2e_tests/testconsts"
	utilst "github.com/dana-team/rcs-ocm-deployer/test/e2e_tests/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Validate the placement controller", func() {
	It("Should update the site in status and an annotation", func() {
		baseCapp := mock.CreateBaseCapp()
		desiredCapp := utilst.CreateCapp(k8sClient, baseCapp)

		By("Checks unique creation of Capp")
		assertionCapp := utilst.GetCapp(k8sClient, desiredCapp.Name, desiredCapp.Namespace)
		Expect(assertionCapp.Name).ShouldNot(Equal(baseCapp.Name))

		By("Checks if Capp got the site in status and in annotation")
		Eventually(func() string {
			assertionCapp = utilst.GetCapp(k8sClient, assertionCapp.Name, assertionCapp.Namespace)
			return assertionCapp.Status.ApplicationLinks.Site
		}, testconsts.Timeout, testconsts.Interval).ShouldNot(Equal(""), "Should fetch capp.")
		Eventually(func() string {
			assertionCapp = utilst.GetCapp(k8sClient, assertionCapp.Name, assertionCapp.Namespace)
			return assertionCapp.Annotations[testconsts.AnnotationKeyHasPlacement]
		}, testconsts.Timeout, testconsts.Interval).ShouldNot(Equal(""), "Should fetch capp.")
	})

	It("Should update a site from placement in status and an annotation", func() {
		baseCapp := mock.CreateBaseCapp()
		baseCapp.Spec.Site = testconsts.Placement
		desiredCapp := utilst.CreateCapp(k8sClient, baseCapp)

		By("Checks unique creation of Capp")
		assertionCapp := utilst.GetCapp(k8sClient, desiredCapp.Name, desiredCapp.Namespace)
		Expect(assertionCapp.Name).ShouldNot(Equal(baseCapp.Name))

		By("Checks if got the site in status and in annotation")
		Eventually(func() bool {
			assertionCapp = utilst.GetCapp(k8sClient, assertionCapp.Name, assertionCapp.Namespace)
			status, err := utilst.IsSiteInPlacement(k8sClient, assertionCapp.Status.ApplicationLinks.Site, assertionCapp.Spec.Site, "test")
			Expect(err).Should(BeNil())
			return status
		}, testconsts.Timeout, testconsts.Interval).Should(BeTrue(), "Should fetch capp.")
		Eventually(func() bool {
			assertionCapp = utilst.GetCapp(k8sClient, assertionCapp.Name, assertionCapp.Namespace)
			status, err := utilst.IsSiteInPlacement(k8sClient, assertionCapp.Annotations[testconsts.AnnotationKeyHasPlacement], assertionCapp.Spec.Site, "test")
			Expect(err).Should(BeNil())
			return status
		}, testconsts.Timeout, testconsts.Interval).Should(BeTrue(), "Should fetch capp.")
	})

	It("Should update the selected cluster in status and an annotation ", func() {
		baseCapp := mock.CreateBaseCapp()
		baseCapp.Spec.Site = testconsts.Cluster
		desiredCapp := utilst.CreateCapp(k8sClient, baseCapp)

		By("Checks unique creation of Capp")
		assertionCapp := utilst.GetCapp(k8sClient, desiredCapp.Name, desiredCapp.Namespace)
		Expect(assertionCapp.Name).ShouldNot(Equal(baseCapp.Name))

		By("Checks if got the cluster in status and in annotation")
		Eventually(func() bool {
			assertionCapp = utilst.GetCapp(k8sClient, assertionCapp.Name, assertionCapp.Namespace)
			return assertionCapp.Status.ApplicationLinks.Site == assertionCapp.Spec.Site
		}, testconsts.Timeout, testconsts.Interval).Should(BeTrue(), "Should fetch capp.")
		Eventually(func() bool {
			assertionCapp = utilst.GetCapp(k8sClient, assertionCapp.Name, assertionCapp.Namespace)
			return assertionCapp.Annotations[testconsts.AnnotationKeyHasPlacement] == assertionCapp.Spec.Site
		}, testconsts.Timeout, testconsts.Interval).Should(BeTrue(), "Should fetch capp.")
	})
})
