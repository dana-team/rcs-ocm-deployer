package e2e_tests

import (
	"context"

	rcsv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
	"github.com/dana-team/rcs-ocm-deployer/internals/utils"
	mock "github.com/dana-team/rcs-ocm-deployer/test/e2e_tests/mocks"
	utilst "github.com/dana-team/rcs-ocm-deployer/test/e2e_tests/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	workv1 "open-cluster-management.io/api/work/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Validate the placement sync controller", func() {

	It("Should add a cleanup finalizer to created capp", func() {
		baseCapp := mock.CreateBaseCapp()
		desiredCapp := utilst.CreateCapp(k8sClient, baseCapp)

		By("Checks unique creation of Capp")
		assertionCapp := utilst.GetCapp(k8sClient, desiredCapp.Name, desiredCapp.Namespace)
		Expect(assertionCapp.Name).ShouldNot(Equal(baseCapp.Name))

		By("Checks if Capp got capp-cleanup annotation")
		Eventually(func() bool {
			assertionCapp = utilst.GetCapp(k8sClient, assertionCapp.Name, assertionCapp.Namespace)
			return utilst.DoesFinalizerExist(k8sClient, assertionCapp.Name, assertionCapp.Namespace, "dana.io/capp-cleanup")
		}, TimeoutCapp, CappCreationInterval).Should(BeTrue(), "Should fetch capp.")

	})

	It("Should delete all capp dependent resources when capp is deleted", func() {
		baseCapp := mock.CreateBaseCapp()
		desiredCapp := utilst.CreateCapp(k8sClient, baseCapp)

		By("Checks unique creation of Capp")
		assertionCapp := utilst.GetCapp(k8sClient, desiredCapp.Name, desiredCapp.Namespace)
		Expect(assertionCapp.Name).ShouldNot(Equal(baseCapp.Name))

		By("Waiting for placement to be set on Capp")
		Eventually(func() string {
			assertionCapp = utilst.GetCapp(k8sClient, assertionCapp.Name, assertionCapp.Namespace)
			return assertionCapp.Annotations["dana.io/has-placement"]
		}, TimeoutCapp, CappCreationInterval).ShouldNot(Equal(""), "Should fetch capp.")
		mwNamespace := assertionCapp.Annotations["dana.io/has-placement"]

		By("Deletes capp")
		utilst.DeleteCapp(k8sClient, assertionCapp)

		By("Checks if ManifestWork was deleted")
		manifestWork := &workv1.ManifestWork{}
		Eventually(func() bool {
			mwName := utils.NamespaceManifestWorkPrefix + assertionCapp.Namespace + "-" + assertionCapp.Name
			Expect(k8sClient.Get(context.Background(), client.ObjectKey{Name: mwName, Namespace: mwNamespace}, manifestWork)).Should(Succeed())
			return utilst.DoesResourceExist(k8sClient, manifestWork)
		}, TimeoutCapp, CappCreationInterval).ShouldNot(BeFalse())
	})

	It("Should copy the secret from volumes to ManifestWork ", func() {
		baseSecret := mock.CreateSecret()
		secret := utilst.CreateSecret(k8sClient, baseSecret)
		baseCapp := mock.CreateBaseCapp()
		baseCapp.Spec.ConfigurationSpec.Template.Spec.Volumes = append(baseCapp.Spec.ConfigurationSpec.Template.Spec.Volumes, corev1.Volume{
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secret.Name,
				},
			},
		})
		desiredCapp := utilst.CreateCapp(k8sClient, baseCapp)

		By("Checks unique creation of Capp")
		assertionCapp := utilst.GetCapp(k8sClient, desiredCapp.Name, desiredCapp.Namespace)
		Expect(assertionCapp.Name).ShouldNot(Equal(baseCapp.Name))

		By("Waiting for placement to be set on Capp")
		Eventually(func() string {
			assertionCapp = utilst.GetCapp(k8sClient, assertionCapp.Name, assertionCapp.Namespace)
			return assertionCapp.Annotations["dana.io/has-placement"]
		}, TimeoutCapp, CappCreationInterval).ShouldNot(Equal(""), "Should fetch capp.")
		mwNamespace := assertionCapp.Annotations["dana.io/has-placement"]

		By("Checks ManifestWork was synced with secret")
		manifestWork := &workv1.ManifestWork{}
		Eventually(func() bool {
			mwName := utils.NamespaceManifestWorkPrefix + assertionCapp.Namespace + "-" + assertionCapp.Name
			_ = k8sClient.Get(context.Background(), client.ObjectKey{Name: mwName, Namespace: mwNamespace}, manifestWork)
			secret, err := utilst.IsObjInManifestWork(k8sClient, *manifestWork, secret.Name, secret.Namespace, &corev1.Secret{}, "Secret")
			Expect(err).Should(BeNil())
			return secret
		}, TimeoutCapp, CappCreationInterval).Should(BeTrue())
	})

	It("Should copy the secret from environment variables to ManifestWork ", func() {
		baseSecret := mock.CreateSecret()
		secret := utilst.CreateSecret(k8sClient, baseSecret)
		baseCapp := mock.CreateBaseCapp()
		baseCapp.Spec.ConfigurationSpec.Template.Spec.Containers[0].EnvFrom = append(baseCapp.Spec.ConfigurationSpec.Template.Spec.Containers[0].EnvFrom, corev1.EnvFromSource{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: secret.Name,
				},
			},
		})
		desiredCapp := utilst.CreateCapp(k8sClient, baseCapp)

		By("Checks unique creation of Capp")
		assertionCapp := utilst.GetCapp(k8sClient, desiredCapp.Name, desiredCapp.Namespace)
		Expect(assertionCapp.Name).ShouldNot(Equal(baseCapp.Name))

		By("Waiting for placement to be set on Capp")
		Eventually(func() string {
			assertionCapp = utilst.GetCapp(k8sClient, assertionCapp.Name, assertionCapp.Namespace)
			return assertionCapp.Annotations["dana.io/has-placement"]
		}, TimeoutCapp, CappCreationInterval).ShouldNot(Equal(""), "Should fetch capp.")
		mwNamespace := assertionCapp.Annotations["dana.io/has-placement"]

		By("Checks ManifestWork was synced with secret")
		manifestWork := &workv1.ManifestWork{}
		Eventually(func() bool {
			mwName := utils.NamespaceManifestWorkPrefix + assertionCapp.Namespace + "-" + assertionCapp.Name
			Expect(k8sClient.Get(context.Background(), client.ObjectKey{Name: mwName, Namespace: mwNamespace}, manifestWork)).Should(Succeed())
			secret, err := utilst.IsObjInManifestWork(k8sClient, *manifestWork, secret.Name, secret.Namespace, &corev1.Secret{}, "Secret")
			Expect(err).Should(BeNil())
			return secret
		}, TimeoutCapp, CappCreationInterval).Should(BeTrue())
	})
	It("Should copy the secret from RouteSpec to ManifestWork ", func() {
		baseSecret := mock.CreateSecret()
		secret := utilst.CreateSecret(k8sClient, baseSecret)
		baseCapp := mock.CreateBaseCapp()
		baseCapp.Spec.RouteSpec.TlsEnabled = true
		baseCapp.Spec.RouteSpec.TlsSecret = secret.Name
		desiredCapp := utilst.CreateCapp(k8sClient, baseCapp)

		By("Checks unique creation of Capp")
		assertionCapp := utilst.GetCapp(k8sClient, desiredCapp.Name, desiredCapp.Namespace)
		Expect(assertionCapp.Name).ShouldNot(Equal(baseCapp.Name))

		By("Waiting for placement to be set on Capp")
		Eventually(func() string {
			assertionCapp = utilst.GetCapp(k8sClient, assertionCapp.Name, assertionCapp.Namespace)
			return assertionCapp.Annotations["dana.io/has-placement"]
		}, TimeoutCapp, CappCreationInterval).ShouldNot(Equal(""), "Should fetch capp.")
		mwNamespace := assertionCapp.Annotations["dana.io/has-placement"]

		By("Checks ManifestWork was synced with secret")
		manifestWork := &workv1.ManifestWork{}
		Eventually(func() bool {
			mwName := utils.NamespaceManifestWorkPrefix + assertionCapp.Namespace + "-" + assertionCapp.Name
			_ = k8sClient.Get(context.Background(), client.ObjectKey{Name: mwName, Namespace: mwNamespace}, manifestWork)
			secret, err := utilst.IsObjInManifestWork(k8sClient, *manifestWork, secret.Name, secret.Namespace, &corev1.Secret{}, "Secret")
			Expect(err).Should(BeNil())
			return secret
		}, TimeoutCapp, CappCreationInterval).Should(BeTrue())
	})

	It("Should copy the secret from imagePullSecrets to ManifestWork ", func() {
		baseSecret := mock.CreateSecret()
		secret := utilst.CreateSecret(k8sClient, baseSecret)
		baseCapp := mock.CreateBaseCapp()
		baseCapp.Spec.ConfigurationSpec.Template.Spec.ImagePullSecrets = append(baseCapp.Spec.ConfigurationSpec.Template.Spec.ImagePullSecrets, corev1.LocalObjectReference{
			Name: secret.Name,
		})
		desiredCapp := utilst.CreateCapp(k8sClient, baseCapp)

		By("Checks unique creation of Capp")
		assertionCapp := utilst.GetCapp(k8sClient, desiredCapp.Name, desiredCapp.Namespace)
		Expect(assertionCapp.Name).ShouldNot(Equal(baseCapp.Name))

		By("Waiting for placement to be set on Capp")
		Eventually(func() string {
			assertionCapp = utilst.GetCapp(k8sClient, assertionCapp.Name, assertionCapp.Namespace)
			return assertionCapp.Annotations["dana.io/has-placement"]
		}, TimeoutCapp, CappCreationInterval).ShouldNot(Equal(""), "Should fetch capp.")
		mwNamespace := assertionCapp.Annotations["dana.io/has-placement"]

		By("Checks ManifestWork was synced with secret")
		manifestWork := &workv1.ManifestWork{}
		Eventually(func() bool {
			mwName := utils.NamespaceManifestWorkPrefix + assertionCapp.Namespace + "-" + assertionCapp.Name
			_ = k8sClient.Get(context.Background(), client.ObjectKey{Name: mwName, Namespace: mwNamespace}, manifestWork)
			secret, err := utilst.IsObjInManifestWork(k8sClient, *manifestWork, secret.Name, secret.Namespace, &corev1.Secret{}, "Secret")
			Expect(err).Should(BeNil())
			return secret
		}, TimeoutCapp, CappCreationInterval).Should(BeTrue())
	})

	It("Should copy the rolebindings from capp's namespace to ManifestWork ", func() {
		baseRole := mock.CreateRole()
		role := utilst.CreateRole(k8sClient, baseRole)
		baseRoleBinding := mock.CreateRoleBinding()
		roleBinding := utilst.CreateRoleBinding(k8sClient, baseRoleBinding)
		baseCapp := mock.CreateBaseCapp()
		desiredCapp := utilst.CreateCapp(k8sClient, baseCapp)

		By("Checks unique creation of Capp")
		assertionCapp := utilst.GetCapp(k8sClient, desiredCapp.Name, desiredCapp.Namespace)
		Expect(assertionCapp.Name).ShouldNot(Equal(baseCapp.Name))

		By("Waiting for placement to be set on Capp")
		Eventually(func() string {
			assertionCapp = utilst.GetCapp(k8sClient, assertionCapp.Name, assertionCapp.Namespace)
			return assertionCapp.Annotations["dana.io/has-placement"]
		}, TimeoutCapp, CappCreationInterval).ShouldNot(Equal(""), "Should fetch capp.")
		mwNamespace := assertionCapp.Annotations["dana.io/has-placement"]

		By("Checks ManifestWork was synced with role and rolebinding")
		manifestWork := &workv1.ManifestWork{}
		Eventually(func() bool {
			mwName := utils.NamespaceManifestWorkPrefix + assertionCapp.Namespace + "-" + assertionCapp.Name
			_ = k8sClient.Get(context.Background(), client.ObjectKey{Name: mwName, Namespace: mwNamespace}, manifestWork)
			return utilst.IsRbacObjInManifestWork(k8sClient, *manifestWork, assertionCapp.Name, role.Namespace, "Role") &&
				utilst.IsRbacObjInManifestWork(k8sClient, *manifestWork, assertionCapp.Name, roleBinding.Namespace, "RoleBinding")
		}, TimeoutCapp, CappCreationInterval).Should(BeTrue())
	})

	It("Should copy the capp and capp's namespace to ManifestWork ", func() {
		baseCapp := mock.CreateBaseCapp()
		desiredCapp := utilst.CreateCapp(k8sClient, baseCapp)

		By("Checks unique creation of Capp")
		assertionCapp := utilst.GetCapp(k8sClient, desiredCapp.Name, desiredCapp.Namespace)
		Expect(assertionCapp.Name).ShouldNot(Equal(baseCapp.Name))

		By("Waiting for placement to be set on Capp")
		Eventually(func() string {
			assertionCapp = utilst.GetCapp(k8sClient, assertionCapp.Name, assertionCapp.Namespace)
			return assertionCapp.Annotations["dana.io/has-placement"]
		}, TimeoutCapp, CappCreationInterval).ShouldNot(Equal(""), "Should fetch capp.")
		mwNamespace := assertionCapp.Annotations["dana.io/has-placement"]

		By("Checks ManifestWork was synced with capp and namespace")
		manifestWork := &workv1.ManifestWork{}
		Eventually(func() bool {
			mwName := utils.NamespaceManifestWorkPrefix + assertionCapp.Namespace + "-" + assertionCapp.Name
			_ = k8sClient.Get(context.Background(), client.ObjectKey{Name: mwName, Namespace: mwNamespace}, manifestWork)
			ns, err := utilst.IsObjInManifestWork(k8sClient, *manifestWork, assertionCapp.Namespace, "", &corev1.Namespace{}, "Namespace")
			Expect(err).Should(BeNil())
			capp, err := utilst.IsObjInManifestWork(k8sClient, *manifestWork, assertionCapp.Name, assertionCapp.Namespace, &rcsv1alpha1.Capp{}, "Capp")
			Expect(err).Should(BeNil())
			return ns && capp
		}, TimeoutCapp, CappCreationInterval).Should(BeTrue())
	})

	It("Should copy the updated capp to ManifestWork ", func() {
		baseCapp := mock.CreateBaseCapp()
		desiredCapp := utilst.CreateCapp(k8sClient, baseCapp)

		By("Checks unique creation of Capp")
		assertionCapp := utilst.GetCapp(k8sClient, desiredCapp.Name, desiredCapp.Namespace)
		Expect(assertionCapp.Name).ShouldNot(Equal(baseCapp.Name))

		By("Waiting for placement to be set on Capp")
		Eventually(func() string {
			assertionCapp = utilst.GetCapp(k8sClient, assertionCapp.Name, assertionCapp.Namespace)
			return assertionCapp.Annotations["dana.io/has-placement"]
		}, TimeoutCapp, CappCreationInterval).ShouldNot(Equal(""), "Should fetch capp.")
		mwNamespace := assertionCapp.Annotations["dana.io/has-placement"]

		By("Update Capp")
		assertionCapp.Spec.Site = "cluster2"
		Expect(k8sClient.Update(context.Background(), assertionCapp)).Should(Succeed())

		By("Checks Capp site is not nil")
		manifestWork := &workv1.ManifestWork{}
		mwName := utils.NamespaceManifestWorkPrefix + assertionCapp.Namespace + "-" + assertionCapp.Name
		Eventually(func() interface{} {
			_ = k8sClient.Get(context.Background(), client.ObjectKey{Name: mwName, Namespace: mwNamespace}, manifestWork)
			return utilst.GetCappFromManifestWork(k8sClient, *manifestWork).Object["spec"].(map[string]interface{})["site"]
		}, TimeoutCapp, CappCreationInterval).ShouldNot(BeNil())

		By("Checks ManifestWork was synced with new capp")
		Eventually(func() string {
			_ = k8sClient.Get(context.Background(), client.ObjectKey{Name: mwName, Namespace: mwNamespace}, manifestWork)
			return utilst.GetCappFromManifestWork(k8sClient, *manifestWork).Object["spec"].(map[string]interface{})["site"].(string)
		}, TimeoutCapp, CappCreationInterval).Should(Equal("cluster2"))
	})
})
