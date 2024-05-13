package mocks

import (
	cappv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	knativev1 "knative.dev/serving/pkg/apis/serving/v1"
)

var (
	NSName            = "capp-e2e-tests"
	cappSecretName    = "capp-secret"
	cappConfigMapName = "capp-config"
	cappAdmin         = "capp-admin"
	cappName          = "capp-default-test"
	cappBaseImage     = "ghcr.io/knative/autoscale-go:latest"
	DataKey           = "password"
	DataValue         = "password"
	KindConfigMap     = "ConfigMap"
	KindSecret        = "Secret"
)

// CreateBaseCapp is responsible for making the most lean version of Capp, so we can manipulate it in the tests.
func CreateBaseCapp() *cappv1alpha1.Capp {
	return &cappv1alpha1.Capp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cappName,
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
											Value: cappName,
										},
									},
									Image: cappBaseImage,
									Name:  cappName,
								},
							},
						},
					},
				},
			},
		},
	}
}

// CreateSecret creates a basic secret.
func CreateSecret() *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       KindSecret,
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cappSecretName,
			Namespace: NSName,
		},
		Data: map[string][]byte{
			DataKey: []byte(DataValue),
		},
	}
}

// CreateConfigMap creates a basic configMap.
func CreateConfigMap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       KindConfigMap,
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cappConfigMapName,
			Namespace: NSName,
		},
		BinaryData: map[string][]byte{
			DataKey: []byte(DataValue),
		},
	}
}

// CreateCappRole creates a role with basic permissions for pod logs.
func CreateCappRole() *rbacv1.Role {
	rules := []rbacv1.PolicyRule{
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
	}

	return CreateRole(cappAdmin+"-role", rules)
}

// CreateCappRoleBinding creates a binding for the pod reader role.
func CreateCappRoleBinding() *rbacv1.RoleBinding {
	roleRef := rbacv1.RoleRef{
		Name:     cappAdmin + "-role",
		Kind:     "Role",
		APIGroup: "rbac.authorization.k8s.io",
	}

	subjects := []rbacv1.Subject{
		{
			Kind:     "User",
			Name:     cappAdmin,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}

	return CreateRoleBinding(cappAdmin+"-role-binding", roleRef, subjects)
}

// CreateRole creates a role with the specified name and rules.
func CreateRole(name string, rules []rbacv1.PolicyRule) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: NSName,
		},
		Rules: rules,
	}
}

// CreateRoleBinding creates a role binding with the specified name, role reference, and subjects.
func CreateRoleBinding(name string, roleRef rbacv1.RoleRef, subjects []rbacv1.Subject) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: NSName,
		},
		RoleRef:  roleRef,
		Subjects: subjects,
	}
}

// CreateServiceAccount creates a service account with the specified name.
func CreateServiceAccount(name string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: NSName,
		},
	}
}
