package directors

import (
	"context"
	"testing"

	cappv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestPrepareAdminsRolesForCapp(t *testing.T) {
	ctx := context.TODO()

	// Create a test Capp
	capp := cappv1alpha1.Capp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-capp",
			Namespace: "test-namespace",
		},
	}

	// Create a fake client
	fakeClient := fake.NewClientBuilder().Build()

	authDirector := AuthDirector{ctx, fakeClient, logr.Discard(), record.NewFakeRecorder(10)}

	mannifests, err := authDirector.AssembleManifests(capp)

	role, rolebinding := mannifests[0].Object.(*rbacv1.Role), mannifests[1].Object.(*rbacv1.RoleBinding)

	// Assert that there are no errors
	assert.NoError(t, err)

	// Assert that the returned Role and RoleBinding are not nil
	assert.NotNil(t, role)
	assert.NotNil(t, rolebinding)

	// Assert that the Role and RoleBinding have the correct names
	assert.Equal(t, "test-capp-logs-reader", role.Name)
	assert.Equal(t, "test-capp-logs-reader", rolebinding.Name)

	// Assert that the RoleRef in the RoleBinding has the correct name and kind
	assert.Equal(t, "test-capp-logs-reader", rolebinding.RoleRef.Name)
	assert.Equal(t, "Role", rolebinding.RoleRef.Kind)

	// Assert that the Role has a single rule for pod logs with get, watch, and list verbs
	assert.Len(t, role.Rules, 1)
	assert.Equal(t, []string{"pod/logs"}, role.Rules[0].Resources)
	assert.Equal(t, []string{"get", "watch", "list"}, role.Rules[0].Verbs)
}

func TestGenerateSubjectsFromUsers(t *testing.T) {
	// Create a list of user names
	users := []string{"user1", "user2", "user3"}

	// Call generateSubjectsFromUsers with the test user list
	subjects := generateSubjectsFromUsers(users)

	// Assert that the returned subjects have the correct length
	assert.Len(t, subjects, len(users))

	// Assert that each subject has the correct name, kind, and APIGroup
	for i, user := range users {
		assert.Equal(t, user, subjects[i].Name)
		assert.Equal(t, "User", subjects[i].Kind)
		assert.Equal(t, "rbac.authorization.k8s.io", subjects[i].APIGroup)
	}
}

func TestGetUsersfromNamespace(t *testing.T) {
	ctx := context.TODO()

	// Create a test Capp
	capp := cappv1alpha1.Capp{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-namespace",
		},
	}

	// Create a test RoleBinding with admin and logs-reader roles
	rolebinding1 := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-namespace",
			Name:      "admin",
		},
		RoleRef: rbacv1.RoleRef{
			Name: "admin",
		},
		Subjects: []rbacv1.Subject{
			{
				Name: "user1",
			},
			{
				Name: "user2",
			},
		},
	}
	rolebinding2 := rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-namespace",
			Name:      "logs-reader",
		},
		RoleRef: rbacv1.RoleRef{
			Name: "logs-reader",
		},
		Subjects: []rbacv1.Subject{
			{
				Name: "user3",
			},
			{
				Name: "user4",
			},
		},
	}

	// Create a fake client and add the test RoleBindings
	fakeClient := fake.NewClientBuilder().WithObjects(&rolebinding1, &rolebinding2).Build()

	// Call getUsersfromNamespace with the test context, fake client, and test Capp
	users, err := getUsersfromNamespace(ctx, fakeClient, capp)

	// Assert that there are no errors
	assert.NoError(t, err)

	// Assert that the returned user list has the correct length
	assert.Len(t, users, 4)

	// Assert that the user list contains the correct users
	assert.Contains(t, users, "user1")
	assert.Contains(t, users, "user2")
	assert.Contains(t, users, "user3")
	assert.Contains(t, users, "user4")
}
