package utils

import (
	"context"
	"math/rand"
	"time"

	"github.com/dana-team/rcs-ocm-deployer/test/e2e_tests/testconsts"

	cappv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	charset              = "abcdefghijklmnopqrstuvwxyz0123456789"
	RandStrLength        = 10
	TimeoutCapp          = 60 * time.Second
	CappCreationInterval = 2 * time.Second
)

var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))

// generateRandomString returns a random string of the specified length using characters from the charset.
func generateRandomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

// CreateCapp creates a new Capp instance with a unique name and returns it.
func CreateCapp(k8sClient client.Client, capp *cappv1alpha1.Capp) *cappv1alpha1.Capp {
	cappName := GenerateUniqueCappName(capp.Name)
	newCapp := capp.DeepCopy()
	newCapp.Name = cappName
	Expect(k8sClient.Create(context.Background(), newCapp)).To(Succeed())
	Eventually(func() string {
		return GetCapp(k8sClient, newCapp.Name, newCapp.Namespace).Status.StateStatus.State
	}, TimeoutCapp, CappCreationInterval).Should(Equal("enabled"), "Should fetch capp")
	return newCapp
}

// GenerateUniqueCappName generates a unique Capp name.
func GenerateUniqueCappName(baseCappName string) string {
	randString := generateRandomString(RandStrLength)
	return baseCappName + "-" + randString
}

// DeleteCapp deletes an existing Capp instance.
func DeleteCapp(k8sClient client.Client, capp *cappv1alpha1.Capp) {
	Expect(k8sClient.Delete(context.Background(), capp)).To(Succeed())
}

// GetCapp fetches an existing Capp and return the instance.
func GetCapp(k8sClient client.Client, name string, namespace string) *cappv1alpha1.Capp {
	capp := &cappv1alpha1.Capp{}
	Expect(k8sClient.Get(context.Background(), client.ObjectKey{Name: name, Namespace: namespace}, capp)).To(Succeed())
	return capp
}

// UpdateCapp updates the provided Capp instance in the Kubernetes cluster, and returns it.
func UpdateCapp(k8sClient client.Client, capp *cappv1alpha1.Capp) {
	Expect(k8sClient.Update(context.Background(), capp)).To(Succeed())
}

// DoesFinalizerExist checks if a finalizer exists on a Capp.
func DoesFinalizerExist(k8sClient client.Client, cappName string, cappNamespace string, finalizerName string) bool {
	capp := GetCapp(k8sClient, cappName, cappNamespace)
	for _, finalizer := range capp.GetFinalizers() {
		if finalizer == finalizerName {
			return true
		}
	}
	return false
}

// GetCappWithPlacementAnnotation checks whether a Capp eventually has a placement annotation.
// If it exists then it returns an up-to-date Capp which contains the annotation.
func GetCappWithPlacementAnnotation(k8sClient client.Client, name, namespace string) *cappv1alpha1.Capp {
	capp := &cappv1alpha1.Capp{}
	Eventually(func() string {
		capp = GetCapp(k8sClient, name, namespace)
		return capp.Annotations[testconsts.AnnotationKeyHasPlacement]
	}, testconsts.Timeout, testconsts.Interval).ShouldNot(Equal(""), "Should have placement annotation on Capp.")

	return capp
}
