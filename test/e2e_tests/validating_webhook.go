package e2e_tests

import (
	"context"

	mock "github.com/dana-team/rcs-ocm-deployer/test/e2e_tests/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	UnsupportedScaleMetric = "storage"
	UnsupportedCluster     = "my-cluster"
	UnsupportedSite        = "my-site"
	UnsupportedHostname    = "...aaa.a...."
)

var _ = Describe("Validate the validating webhook", func() {

	It("Should deny the use of an unsupported scale metric", func() {
		baseCapp := mock.CreateBaseCapp()
		baseCapp.Spec.ScaleMetric = UnsupportedScaleMetric
		Expect(k8sClient.Create(context.Background(), baseCapp)).ShouldNot(Succeed())

	})

	It("Should deny the use of a non-existing cluster", func() {
		baseCapp := mock.CreateBaseCapp()
		baseCapp.Spec.Site = UnsupportedCluster
		Expect(k8sClient.Create(context.Background(), baseCapp)).ShouldNot(Succeed())
	})

	It("Should deny the use of a non-existing placement", func() {
		baseCapp := mock.CreateBaseCapp()
		baseCapp.Spec.Site = UnsupportedSite
		Expect(k8sClient.Create(context.Background(), baseCapp)).ShouldNot(Succeed())
	})

	It("Should deny the use of an invalid hostname", func() {
		baseCapp := mock.CreateBaseCapp()
		baseCapp.Spec.RouteSpec.Hostname = UnsupportedHostname
		Expect(k8sClient.Create(context.Background(), baseCapp)).ShouldNot(Succeed())
	})
})
