package utils

import (
	"context"

	rcsv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func PrepareAdminsRolesForCapp(ctx context.Context, r client.Client, capp rcsv1alpha1.Capp) (rbacv1.Role, rbacv1.RoleBinding, error) {
	role := rbacv1.Role{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      string(capp.Name + "-logs-reader"),
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
	subjects := generateSubjectsFromUsers(users)
	if err != nil {
		return rbacv1.Role{}, rbacv1.RoleBinding{}, err
	}

	rolebinding := rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      string(capp.Name + "-logs-reader"),
			Namespace: capp.Namespace,
		},
		RoleRef: rbacv1.RoleRef{
			Name:     string(capp.Name + "-logs-reader"),
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
		},
		Subjects: subjects,
	}
	return role, rolebinding, nil
}

func generateSubjectsFromUsers(users []string) []rbacv1.Subject {
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

func GetUsersfromNamespace(ctx context.Context, r client.Client, capp rcsv1alpha1.Capp) ([]string, error) {
	rolebindings := rbacv1.ClusterRoleBindingList{}
	users := []string{}
	if err := r.List(ctx, &rolebindings); err != nil {
		return users, err
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
