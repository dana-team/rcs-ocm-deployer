package e2e_tests

import (
	"k8s.io/client-go/rest"

	cappv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
	loggingv1beta1 "github.com/kube-logging/logging-operator/pkg/sdk/logging/api/v1beta1"
	networkingv1 "github.com/openshift/api/network/v1"
	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	knativev1 "knative.dev/serving/pkg/apis/serving/v1"
	knativev1alphav1 "knative.dev/serving/pkg/apis/serving/v1alpha1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	clusterv1alpha1 "open-cluster-management.io/api/cluster/v1alpha1"
	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"
	clusterv1beta2 "open-cluster-management.io/api/cluster/v1beta2"
	workv1 "open-cluster-management.io/api/work/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	k8sClient client.Client
	cfg       *rest.Config
)

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
