package builders

import (
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	workv1 "open-cluster-management.io/api/work/v1"
)

// BuildRole creates a workv1.Manifest with a Role for pod log access in a specific namespace.
func BuildRole(cappName string, namespace string) workv1.Manifest {
	role := rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cappName + "-logs-reader",
			Namespace: namespace,
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
	return workv1.Manifest{RawExtension: runtime.RawExtension{Object: &role}}
}

// BuildRoleBinding constructs a workv1.Manifest containing a RoleBinding for given subjects, scoped to a specific capp name and namespace.
func BuildRoleBinding(cappName string, namespace string, subjects []rbacv1.Subject) workv1.Manifest {
	roleBinding := rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cappName + "-logs-reader",
			Namespace: namespace,
		},
		RoleRef: rbacv1.RoleRef{
			Name:     cappName + "-logs-reader",
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
		},
		Subjects: subjects,
	}
	return workv1.Manifest{RawExtension: runtime.RawExtension{Object: &roleBinding}}
}
