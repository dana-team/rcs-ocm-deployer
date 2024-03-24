package mocks

import (
	cappv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	knativev1 "knative.dev/serving/pkg/apis/serving/v1"
)

var (
	NSName          = "capp-e2e-tests"
	CappSecretName  = "capp-secret"
	CappAdmin       = "capp-admin"
	CappName        = "capp-default-test"
	CappBaseImage   = "ghcr.io/knative/autoscale-go:latest"
	SecretDataKey   = "password"
	SecretDataValue = "password"
)

// CreateBaseCapp is responsible for making the most lean version of Capp, so we can manipulate it in the tests.
func CreateBaseCapp() *cappv1alpha1.Capp {
	return &cappv1alpha1.Capp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      CappName,
			Namespace: NSName,
		},
		Spec: cappv1alpha1.CappSpec{
			ConfigurationSpec: knativev1.ConfigurationSpec{
				Template: knativev1.RevisionTemplateSpec{
					Spec: knativev1.RevisionSpec{
						PodSpec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Env: []corev1.EnvVar{
										{
											Name:  "APP_NAME",
											Value: CappName,
										},
									},
									Image: CappBaseImage,
									Name:  CappName,
								},
							},
						},
					},
				},
			},
		},
	}
}

// CreateSecret creates a simple secret for our tests' use.
func CreateSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      CappSecretName,
			Namespace: NSName,
		},
		Data: map[string][]byte{
			SecretDataKey: []byte(SecretDataValue),
		},
	}
}

// CreateRole creates a role with basic permissions for pod logs.
func CreateRole() *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      CappAdmin + "-role",
			Namespace: NSName,
		},
		Rules: []rbacv1.PolicyRule{
			{
				Resources: []string{
					"pod/logs",
				},
				APIGroups: []string{
					"",
				},
				Verbs: []string{
					"get", "watch", "list",
				},
			},
		},
	}
}

// CreateRoleBinding creates a binding for the pod reader role.
func CreateRoleBinding() *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      CappAdmin + "-role-binding",
			Namespace: NSName,
		},
		RoleRef: rbacv1.RoleRef{
			Name:     CappAdmin + "-role",
			Kind:     "Role",
			APIGroup: "rbac.authorization.k8s.io",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:     "User",
				Name:     CappAdmin,
				APIGroup: "rbac.authorization.k8s.io",
			},
		},
	}
}
