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
	unsupportedCluster   = "my-cluster"
	unsupportedSite      = "my-site"
	validDomainSuffix    = ".com"
	unsupportedHostname  = "...aaa.a...."
	clusterLocalHostname = "invalid.svc.cluster.local"
	existingHostname     = "google.com"
	unsupportedLogType   = "unsupported"
	elasticLogType       = "elastic"
	elasticUser          = "user"
	elasticHostExample   = "https://elasticsearch.dana.com/_bulk"
	index                = "main"
	secretName           = "elastic-secret"
)

var _ = Describe("Validate the validating webhook", func() {
	It("Should deny the use of a non-existing cluster", func() {
		baseCapp := mock.CreateBaseCapp()
		baseCapp.Name = utilst.GenerateUniqueCappName(baseCapp.Name)
		baseCapp.Spec.Site = unsupportedCluster
		Expect(k8sClient.Create(context.Background(), baseCapp)).ShouldNot(Succeed())
	})

	It("Should deny the use of a non-existing placement", func() {
		baseCapp := mock.CreateBaseCapp()
		baseCapp.Name = utilst.GenerateUniqueCappName(baseCapp.Name)
		baseCapp.Spec.Site = unsupportedSite
		Expect(k8sClient.Create(context.Background(), baseCapp)).ShouldNot(Succeed())
	})

	It("Should deny the use of an invalid hostname", func() {
		baseCapp := mock.CreateBaseCapp()
		baseCapp.Name = utilst.GenerateUniqueCappName(baseCapp.Name)
		baseCapp.Spec.RouteSpec.Hostname = unsupportedHostname
		Expect(k8sClient.Create(context.Background(), baseCapp)).ShouldNot(Succeed())
	})

	It("Should deny the use of an existing hostname", func() {
		baseCapp := mock.CreateBaseCapp()
		baseCapp.Name = utilst.GenerateUniqueCappName(baseCapp.Name)
		baseCapp.Spec.RouteSpec.Hostname = existingHostname
		Expect(k8sClient.Create(context.Background(), baseCapp)).ShouldNot(Succeed())
	})

	It("Should deny the use of a hostname in cluster local", func() {
		baseCapp := mock.CreateBaseCapp()
		baseCapp.Name = utilst.GenerateUniqueCappName(baseCapp.Name)
		baseCapp.Spec.RouteSpec.Hostname = clusterLocalHostname
		Expect(k8sClient.Create(context.Background(), baseCapp)).ShouldNot(Succeed())
	})

	It("Should allow the use of a unique and valid hostname", func() {
		baseCapp := mock.CreateBaseCapp()
		baseCapp.Name = utilst.GenerateUniqueCappName(baseCapp.Name)
		validHostName := baseCapp.Name + validDomainSuffix
		baseCapp.Spec.RouteSpec.Hostname = validHostName
		Expect(k8sClient.Create(context.Background(), baseCapp)).Should(Succeed())
	})

	It("Should deny the use of an invalid log type", func() {
		baseCapp := mock.CreateBaseCapp()
		baseCapp.Name = utilst.GenerateUniqueCappName(baseCapp.Name)
		baseCapp.Spec.LogSpec.Type = unsupportedLogType
		Expect(k8sClient.Create(context.Background(), baseCapp)).ShouldNot(Succeed())
	})

	It("Should deny the use of an incomplete log spec", func() {
		baseCapp := mock.CreateBaseCapp()
		baseCapp.Name = utilst.GenerateUniqueCappName(baseCapp.Name)
		baseCapp.Spec.LogSpec.Type = elasticLogType
		baseCapp.Spec.LogSpec.Host = elasticHostExample
		Expect(k8sClient.Create(context.Background(), baseCapp)).ShouldNot(Succeed())
	})

	It("Should allow the use of a complete and supported log spec", func() {
		baseCapp := mock.CreateBaseCapp()
		baseCapp.Name = utilst.GenerateUniqueCappName(baseCapp.Name)
		baseCapp.Spec.LogSpec.Type = elasticLogType
		baseCapp.Spec.LogSpec.Host = elasticHostExample
		baseCapp.Spec.LogSpec.Index = index
		baseCapp.Spec.LogSpec.User = elasticUser
		baseCapp.Spec.LogSpec.PasswordSecret = secretName
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
