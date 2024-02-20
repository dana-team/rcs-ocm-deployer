package directors

import (
	"context"
	"fmt"

	rcsv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
	builder "github.com/dana-team/rcs-ocm-deployer/internals/sync/builders"
	"github.com/go-logr/logr"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/client-go/tools/record"
	workv1 "open-cluster-management.io/api/work/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type AuthDirector struct {
	Ctx           context.Context
	K8sclient     client.Client
	Log           logr.Logger
	EventRecorder record.EventRecorder
}

// AssembleManifests constructs a slice of workv1.Manifest objects for a given capp, including a Role and a RoleBinding manifest.
// The Role manifest is created for pod log access, and the RoleBinding manifest associates the Role with subjects derived from users in the capp's namespace.
func (d AuthDirector) AssembleManifests(capp rcsv1alpha1.Capp) ([]workv1.Manifest, error) {
	var manifests []workv1.Manifest
	roleManifest := builder.BuildRole(capp.Name, capp.Namespace)
	manifests = append(manifests, roleManifest)
	users, err := getUsersfromNamespace(d.Ctx, d.K8sclient, capp)
	if err != nil {
		d.Log.Error(err, "could not create auth manifest for capp")
		return []workv1.Manifest{}, err
	}
	subjects := generateSubjectsFromUsers(users)
	roleBindingManifest := builder.BuildRoleBinding(capp.Name, capp.Namespace, subjects)
	manifests = append(manifests, roleBindingManifest)
	return manifests, nil
}

// generateSubjectsFromUsers generates a list of Kubernetes Subject objects from
// a list of user names. It takes a slice of string representing the user names
// and returns a slice of rbacv1.Subject objects.
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

// getUsersfromNamespace returns a list of all user names with admin or logs-reader Roles It takes a context.Context, a Kubernetes client.Client, and a rcsv1alpha1.Capp object representing the Container Application for which the user names are being retrieved,
// and returns a slice of string representing the user names, and an error (if any).
func getUsersfromNamespace(ctx context.Context, r client.Client, capp rcsv1alpha1.Capp) ([]string, error) {
	rolebindings := rbacv1.RoleBindingList{}
	users := []string{}
	listOps := &client.ListOptions{
		Namespace: capp.GetNamespace(),
	}
	if err := r.List(ctx, &rolebindings, listOps); err != nil {
		return users, fmt.Errorf("failed to list roleBindings in the namespace: %v", err.Error())
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
