package utils

import (
	"context"
	"fmt"

	rcsv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// This Go package contains utility functions for managing Kubernetes RBAC Roles and RoleBindings for a Container Application.

// PrepareAdminsRolesForCapp prepares a new RBAC Role and RoleBinding for a
// container application. It grants permission to read logs of the container's
// pods. It takes a context.Context, a Kubernetes client.Client, and a
// rcsv1alpha1.Capp object representing the Container Application for which the
// Role and RoleBinding are being prepared, and returns the new Role,
// RoleBinding, and an error (if any).
func PrepareAdminsRolesForCapp(ctx context.Context, r client.Client, capp rcsv1alpha1.Capp) (rbacv1.Role, rbacv1.RoleBinding, error) {
	role := rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      capp.Name + "-logs-reader",
			Namespace: capp.Namespace,
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
	users, err := GetUsersfromNamespace(ctx, r, capp)
	subjects := GenerateSubjectsFromUsers(users)
	if err != nil {
		return rbacv1.Role{}, rbacv1.RoleBinding{}, err
	}

	rolebinding := rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      capp.Name + "-logs-reader",
			Namespace: capp.Namespace,
		},
		RoleRef: rbacv1.RoleRef{
			Name:     capp.Name + "-logs-reader",
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
		},
		Subjects: subjects,
	}
	return role, rolebinding, nil
}

// GenerateSubjectsFromUsers generates a list of Kubernetes Subject objects from
// a list of user names. It takes a slice of string representing the user names
// and returns a slice of rbacv1.Subject objects.
func GenerateSubjectsFromUsers(users []string) []rbacv1.Subject {
	subjects := []rbacv1.Subject{}
	for _, user := range users {
		subjects = append(subjects, rbacv1.Subject{
			Name:     user,
			Kind:     "User",
			APIGroup: "rbac.authorization.k8s.io",
		})
	}
	return subjects
}

// GetUsersfromNamespace returns a list of all user names with admin or logs-reader Roles It takes a context.Context, a Kubernetes client.Client, and a rcsv1alpha1.Capp object representing the Container Application for which the user names are being retrieved,
// and returns a slice of string representing the user names, and an error (if any).
func GetUsersfromNamespace(ctx context.Context, r client.Client, capp rcsv1alpha1.Capp) ([]string, error) {
	rolebindings := rbacv1.RoleBindingList{}
	users := []string{}
	listOps := &client.ListOptions{
		Namespace: capp.GetNamespace(),
	}
	if err := r.List(ctx, &rolebindings, listOps); err != nil {
		return users, fmt.Errorf("failed to list roleBindings in the namespace: %s", err.Error())
	}
	for _, rb := range rolebindings.Items {
		if rb.RoleRef.Name != "admin" && rb.RoleRef.Name != "logs-reader" {
			continue
		}
		for _, user := range rb.Subjects {
			users = append(users, user.Name)
		}
	}
	return users, nil
}
