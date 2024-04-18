package e2e_tests

import (
	"context"
	"testing"

	cappv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
	mock "github.com/dana-team/rcs-ocm-deployer/test/e2e_tests/mocks"
	"github.com/dana-team/rcs-ocm-deployer/test/e2e_tests/testconsts"
	utilst "github.com/dana-team/rcs-ocm-deployer/test/e2e_tests/utils"
	loggingv1beta1 "github.com/kube-logging/logging-operator/pkg/sdk/logging/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	networkingv1 "github.com/openshift/api/network/v1"
	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	knativev1 "knative.dev/serving/pkg/apis/serving/v1"
	knativev1alphav1 "knative.dev/serving/pkg/apis/serving/v1alpha1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"
	clusterv1beta2 "open-cluster-management.io/api/cluster/v1beta2"
	workv1 "open-cluster-management.io/api/work/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var k8sClient client.Client
var cfg *rest.Config

func newScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = corev1.AddToScheme(s)
	_ = cappv1alpha1.AddToScheme(s)
	_ = loggingv1beta1.AddToScheme(s)
	_ = knativev1alphav1.AddToScheme(s)
	_ = knativev1.AddToScheme(s)
	_ = networkingv1.Install(s)
	_ = routev1.Install(s)
	_ = clusterv1.Install(s)
	_ = clusterv1alpha1.Install(s)
	_ = clusterv1beta1.Install(s)
	_ = clusterv1beta2.Install(s)
	_ = workv1.Install(s)
	_ = scheme.AddToScheme(s)
	return s
}

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
}, func() {
	initClient()
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	utilst.DeleteTestUser(k8sClient, mock.NSName)
	cleanUp()
})

// initClient initializes a k8s client.
func initClient() {
	var err error
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

	Expect(k8sClient.Create(context.Background(), namespace)).To(Succeed())
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
