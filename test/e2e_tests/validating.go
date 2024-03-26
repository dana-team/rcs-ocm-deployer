package e2e_tests

import (
	"context"

	"github.com/dana-team/rcs-ocm-deployer/test/e2e_tests/testconsts"
	utilst "github.com/dana-team/rcs-ocm-deployer/test/e2e_tests/utils"

	mock "github.com/dana-team/rcs-ocm-deployer/test/e2e_tests/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
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
		baseCapp.Name = utilst.GenerateUniqueCappName(baseCapp.Name)
		baseCapp.Spec.Site = UnsupportedCluster
		Expect(k8sClient.Create(context.Background(), baseCapp)).ShouldNot(Succeed())
	})

	It("Should deny the use of a non-existing placement", func() {
		baseCapp := mock.CreateBaseCapp()
		baseCapp.Name = utilst.GenerateUniqueCappName(baseCapp.Name)
		baseCapp.Spec.Site = UnsupportedSite
		Expect(k8sClient.Create(context.Background(), baseCapp)).ShouldNot(Succeed())
	})

	It("Should deny the use of an invalid hostname", func() {
		baseCapp := mock.CreateBaseCapp()
		baseCapp.Name = utilst.GenerateUniqueCappName(baseCapp.Name)
		baseCapp.Spec.RouteSpec.Hostname = UnsupportedHostname
		Expect(k8sClient.Create(context.Background(), baseCapp)).ShouldNot(Succeed())
	})

	It("Should deny the use of an invalid log type", func() {
		baseCapp := mock.CreateBaseCapp()
		baseCapp.Name = utilst.GenerateUniqueCappName(baseCapp.Name)
		baseCapp.Spec.LogSpec.Type = UnsupportedLogType
		Expect(k8sClient.Create(context.Background(), baseCapp)).ShouldNot(Succeed())
	})

	It("Should deny the use of an incomplete log spec", func() {
		baseCapp := mock.CreateBaseCapp()
		baseCapp.Name = utilst.GenerateUniqueCappName(baseCapp.Name)
		baseCapp.Spec.LogSpec.Type = SplunkLogType
		baseCapp.Spec.LogSpec.Host = SplunkHostExample
		Expect(k8sClient.Create(context.Background(), baseCapp)).ShouldNot(Succeed())
	})

	It("Should allow the use of a complete and supported log spec", func() {
		baseCapp := mock.CreateBaseCapp()
		baseCapp.Name = utilst.GenerateUniqueCappName(baseCapp.Name)
		baseCapp.Spec.LogSpec.Type = SplunkLogType
		baseCapp.Spec.LogSpec.Host = SplunkHostExample
		baseCapp.Spec.LogSpec.Index = MainIndex
		baseCapp.Spec.LogSpec.HecTokenSecretName = SplunkSecretNameExample
		Expect(k8sClient.Create(context.Background(), baseCapp)).Should(Succeed())
	})

	It("should deny the use of Opaque secret in tls spec", func() {
		baseCapp := mock.CreateBaseCapp()
		baseCapp.Name = utilst.GenerateUniqueCappName(baseCapp.Name)
		baseSecret := mock.CreateSecret()
		baseSecret.Type = corev1.SecretTypeOpaque
		utilst.CreateSecret(k8sClient, baseSecret)
		Eventually(func() bool {
			return utilst.DoesResourceExist(k8sClient, baseSecret)
		}, testconsts.Timeout, testconsts.Interval).Should(BeTrue())
		baseCapp.Spec.RouteSpec.TlsEnabled = true
		baseCapp.Spec.RouteSpec.TlsSecret = baseSecret.Name
		Expect(k8sClient.Create(context.Background(), baseCapp)).ShouldNot(Succeed())
	})

	It("should allow the use of tls secret in tls spec", func() {
		baseCapp := mock.CreateBaseCapp()
		baseCapp.Name = utilst.GenerateUniqueCappName(baseCapp.Name)
		baseSecret := mock.CreateSecret()
		baseSecret = utilst.CreateTlsSecret(k8sClient, baseSecret)
		baseCapp.Spec.RouteSpec.TlsEnabled = true
		baseCapp.Spec.RouteSpec.TlsSecret = baseSecret.Name
		Expect(k8sClient.Create(context.Background(), baseCapp)).Should(Succeed())
	})

	It("should deny the option of giving non existent secret in tls spec", func() {
		baseCapp := mock.CreateBaseCapp()
		baseCapp.Name = utilst.GenerateUniqueCappName(baseCapp.Name)
		baseCapp.Spec.RouteSpec.TlsEnabled = true
		baseCapp.Spec.RouteSpec.TlsSecret = "notexistsecret"
		Expect(k8sClient.Create(context.Background(), baseCapp)).ShouldNot(Succeed())
	})

})
