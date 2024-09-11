package e2e_tests

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/api/errors"

	mock "github.com/dana-team/rcs-ocm-deployer/test/e2e_tests/mocks"
	"github.com/dana-team/rcs-ocm-deployer/test/e2e_tests/testconsts"
	utilst "github.com/dana-team/rcs-ocm-deployer/test/e2e_tests/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)

	SetDefaultEventuallyTimeout(testconsts.DefaultEventually)
	RunSpecs(t, "RCS Suite")
}

var _ = SynchronizedBeforeSuite(func() {
	initClient()
	cleanUp()
	createE2ETestNamespace()
	utilst.CreateTestUser(k8sClient, mock.NSName)
	utilst.CreateExcludedServiceAccount(k8sClient)
}, func() {
	initClient()
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	utilst.DeleteTestUser(k8sClient, mock.NSName)
	utilst.DeleteExcludedServiceAccount(k8sClient)
	cleanUp()
})

// initClient initializes a k8s client.
func initClient() {
	var err error
	log.SetLogger(logger)
	cfg, err = config.GetConfig()
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: newScheme()})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())
}

// createE2ETestNamespace creates a namespace for the e2e tests.
func createE2ETestNamespace() {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: mock.NSName,
		},
	}

	err := k8sClient.Create(context.Background(), namespace)
	Expect(err).To(SatisfyAny(BeNil(), WithTransform(errors.IsAlreadyExists, BeTrue())))
	Eventually(func() bool {
		return utilst.DoesResourceExist(k8sClient, namespace)
	}, testconsts.Timeout, testconsts.Interval).Should(BeTrue(), "The namespace should be created")
}

// cleanUp makes sure the test environment is clean.
func cleanUp() {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: mock.NSName,
		},
	}
	if utilst.DoesResourceExist(k8sClient, namespace) {
		Expect(k8sClient.Delete(context.Background(), namespace)).To(Succeed())
		Eventually(func() error {
			return k8sClient.Get(context.Background(), client.ObjectKey{Name: mock.NSName}, namespace)
		}, testconsts.Timeout, testconsts.Interval).Should(HaveOccurred(), "The namespace should be deleted")
	}
}
