package e2e_tests

import (
	"context"

	mock "github.com/dana-team/rcs-ocm-deployer/test/e2e_tests/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	UnsupportedCluster      = "my-cluster"
	UnsupportedSite         = "my-site"
	UnsupportedHostname     = "...aaa.a...."
	UnsupportedLogType      = "laber"
	SplunkLogType           = "splunk"
	SplunkHostExample       = "74.234.208.141"
	MainIndex               = "main"
	SplunkSecretNameExample = "splunk-single-standalone-secrets"
)

var _ = Describe("Validate the validating webhook", func() {

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

	It("Should deny the use of an invalid log type", func() {
		baseCapp := mock.CreateBaseCapp()
		baseCapp.Spec.LogSpec.Type = UnsupportedLogType
		Expect(k8sClient.Create(context.Background(), baseCapp)).ShouldNot(Succeed())
	})

	It("Should deny the use of an incomplete log spec", func() {
		baseCapp := mock.CreateBaseCapp()
		baseCapp.Spec.LogSpec.Type = SplunkLogType
		baseCapp.Spec.LogSpec.Host = SplunkHostExample
		Expect(k8sClient.Create(context.Background(), baseCapp)).ShouldNot(Succeed())
	})

	It("Should allow the use of a complete and supported log spec", func() {
		baseCapp := mock.CreateBaseCapp()
		baseCapp.Spec.LogSpec.Type = SplunkLogType
		baseCapp.Spec.LogSpec.Host = SplunkHostExample
		baseCapp.Spec.LogSpec.Index = MainIndex
		baseCapp.Spec.LogSpec.HecTokenSecretName = SplunkSecretNameExample
		Expect(k8sClient.Create(context.Background(), baseCapp)).Should(Succeed())
	})
})
