package utils

import (
	"context"
	"fmt"

	cappv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
	mock "github.com/dana-team/rcs-ocm-deployer/test/e2e_tests/mocks"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ServiceAccountNameFormat = "system:serviceaccount:%s:%s"
	ServiceAccountName       = "test-user"
)

// CreateTestUser creates a test user with the specified Kubernetes client and namespace.
func CreateTestUser(k8sClient client.Client, namespace string) {
	createTestUserServiceAccount(k8sClient)
	createTestUserRoleAndRoleBinding(k8sClient, namespace)
}

// SwitchUser switches the Kubernetes client's user context to the test user if switchToTestUser is true.
// If switchToTestUser is false, it reverts to the original context.
func SwitchUser(k8sClient *client.Client, cfg *rest.Config, namespace string, scheme *runtime.Scheme, switchToTestUser bool) {
	cfg.Impersonate = rest.ImpersonationConfig{}
	if switchToTestUser {
		cfg.Impersonate = rest.ImpersonationConfig{
			UserName: fmt.Sprintf(ServiceAccountNameFormat, namespace, ServiceAccountName),
		}
	}

	newClient, err := client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())

	// Update k8sClient's configuration with the new client's configuration
	*k8sClient = newClient
}

// DeleteTestUser deletes the test user created in the specified namespace.
func DeleteTestUser(k8sClient client.Client, namespace string) {
	deleteTestUserRoleAndRoleBinding(k8sClient, namespace)
	deleteTestUserServiceAccount(k8sClient, namespace)
}

// createTestUserServiceAccount creates a service account for the test user in the specified namespace.
func createTestUserServiceAccount(k8sClient client.Client) {
	serviceAccount := mock.CreateServiceAccount(ServiceAccountName)

	err := k8sClient.Create(context.Background(), serviceAccount)
	Expect(err).To(SatisfyAny(BeNil(), WithTransform(errors.IsAlreadyExists, BeTrue())))
}

// createTestUserRoleAndRoleBinding creates a role and role binding for the test user in the specified namespace.
func createTestUserRoleAndRoleBinding(k8sClient client.Client, namespace string) {
	rules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{cappv1alpha1.GroupVersion.Group},
			Resources: []string{"capps"},
			Verbs:     []string{"get", "update"},
		},
	}
	role := mock.CreateRole(ServiceAccountName, rules)

	// Create or update the Role object
	err := k8sClient.Create(context.Background(), role)
	Expect(err).To(SatisfyAny(BeNil(), WithTransform(errors.IsAlreadyExists, BeTrue())))

	roleRef := rbacv1.RoleRef{
		Kind:     "Role",
		Name:     ServiceAccountName,
		APIGroup: "rbac.authorization.k8s.io",
	}

	subjects := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      ServiceAccountName,
			Namespace: namespace,
		},
	}

	roleBinding := mock.CreateRoleBinding(ServiceAccountName, roleRef, subjects)

	err = k8sClient.Create(context.Background(), roleBinding)
	Expect(err).To(SatisfyAny(BeNil(), WithTransform(errors.IsAlreadyExists, BeTrue())))
}

// deleteTestUserServiceAccount deletes the service account of the test user in the specified namespace.
func deleteTestUserServiceAccount(k8sClient client.Client, namespace string) {
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ServiceAccountName,
			Namespace: namespace,
		},
	}
	err := k8sClient.Delete(context.Background(), serviceAccount)
	Expect(err).To(SatisfyAny(BeNil(), WithTransform(errors.IsNotFound, BeTrue())))
}

// deleteTestUserRoleAndRoleBinding deletes the role and role binding of the test user in the specified namespace.
func deleteTestUserRoleAndRoleBinding(k8sClient client.Client, namespace string) {
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ServiceAccountName,
			Namespace: namespace,
		},
	}
	err := k8sClient.Delete(context.Background(), roleBinding)
	Expect(err).To(SatisfyAny(BeNil(), WithTransform(errors.IsNotFound, BeTrue())))

	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ServiceAccountName,
			Namespace: namespace,
		},
	}
	err = k8sClient.Delete(context.Background(), role)
	Expect(err).To(SatisfyAny(BeNil(), WithTransform(errors.IsNotFound, BeTrue())))
}
