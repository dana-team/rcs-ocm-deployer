package utils

import (
	"context"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	clusterv1beta1 "open-cluster-management.io/api/cluster/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IsSiteInPlacement checks if a given site is part of a clusterset mentioned in a placement
func IsSiteInPlacement(k8sClient client.Client, site string, placementName string, placementNamespace string) (bool, error) {
	placement := clusterv1beta1.Placement{}
	clustersets := []string{}
	managedClusterList := clusterv1.ManagedClusterList{}
	err := k8sClient.Get(context.Background(), client.ObjectKey{Name: placementName, Namespace: placementNamespace}, &placement)
	if err != nil {
		return false, err
	}
	clustersets = append(clustersets, placement.Spec.ClusterSets...)
	err = k8sClient.List(context.Background(), &managedClusterList)
	if err != nil {
		return false, err
	}
	for _, managedCluster := range managedClusterList.Items {
		for _, clusterset := range clustersets {
			if managedCluster.GetLabels()["cluster.open-cluster-management.io/clusterset"] == clusterset {
				return true, nil
			}
		}
	}
	return false, nil

}

// CreateSecret creates a corev1.secret object with a random suffix in its name and returns it
func CreateSecret(k8sClient client.Client, secret *corev1.Secret) *corev1.Secret {
	randString := generateRandomString(RandStrLength)
	secretName := secret.Name + "-" + randString
	secret.Name = secretName
	Expect(k8sClient.Create(context.Background(), secret)).To(Succeed())
	return secret
}

// CreateRole creates a rbac1.Role object and returns it
func CreateRole(k8sClient client.Client, role *rbacv1.Role) *rbacv1.Role {
	Expect(k8sClient.Create(context.Background(), role)).To(Succeed())
	return role
}

// CreateRoleBinding creates a rbacv1.RoleBinding and returns it
func CreateRoleBinding(k8sClient client.Client, roleBinding *rbacv1.RoleBinding) *rbacv1.RoleBinding {
	Expect(k8sClient.Create(context.Background(), roleBinding)).To(Succeed())
	return roleBinding
}
