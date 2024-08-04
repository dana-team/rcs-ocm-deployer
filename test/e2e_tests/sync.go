package e2e_tests

import (
	"context"

	cappv1alpha1 "github.com/dana-team/container-app-operator/api/v1alpha1"
	"github.com/dana-team/rcs-ocm-deployer/test/e2e_tests/testconsts"

	mock "github.com/dana-team/rcs-ocm-deployer/test/e2e_tests/mocks"
	utilst "github.com/dana-team/rcs-ocm-deployer/test/e2e_tests/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	workv1 "open-cluster-management.io/api/work/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	namespaceManifestWorkPrefix = "mw-create-"
	envVarName                  = "E2E-TEST"
)

// verifySecretOrConfigMapCopy makes sure a Secret or a ConfigMap is eventually copied to the ManifestWork,
// by creating a Capp with the needed configuration and getting the created ManifestWork resource.
func verifySecretOrConfigMapCopy(baseCapp *cappv1alpha1.Capp, name, namespace, kind string) {
	desiredCapp := utilst.CreateCapp(k8sClient, baseCapp)

	assertionCapp := utilst.GetCappWithPlacementAnnotation(k8sClient, desiredCapp.Name, desiredCapp.Namespace)
	mwNamespace := assertionCapp.Annotations[testconsts.AnnotationKeyHasPlacement]

	manifestWork := &workv1.ManifestWork{}

	resourceFactory := map[string]client.Object{
		mock.KindSecret:    &corev1.Secret{},
		mock.KindConfigMap: &corev1.ConfigMap{},
	}

	Eventually(func() bool {
		mwName := namespaceManifestWorkPrefix + assertionCapp.Namespace + "-" + assertionCapp.Name
		_ = k8sClient.Get(context.Background(), client.ObjectKey{Name: mwName, Namespace: mwNamespace}, manifestWork)
		object, err := utilst.IsObjInManifestWork(k8sClient, *manifestWork, name, namespace, resourceFactory[kind], kind)
		Expect(err).Should(BeNil())
		return object
	}, testconsts.Timeout, testconsts.Interval).Should(BeTrue())
}

var _ = Describe("Validate the placement sync controller", func() {
	It("Should add a cleanup finalizer to created Capp", func() {
		baseCapp := mock.CreateBaseCapp()
		desiredCapp := utilst.CreateCapp(k8sClient, baseCapp)

		By("Checks unique creation of Capp")
		assertionCapp := utilst.GetCapp(k8sClient, desiredCapp.Name, desiredCapp.Namespace)
		Expect(assertionCapp.Name).ShouldNot(Equal(baseCapp.Name))

		By("Checks if Capp got capp-cleanup annotation")
		Eventually(func() bool {
			assertionCapp = utilst.GetCapp(k8sClient, assertionCapp.Name, assertionCapp.Namespace)
			return utilst.DoesFinalizerExist(k8sClient, assertionCapp.Name, assertionCapp.Namespace, "dana.io/capp-cleanup")
		}, testconsts.Timeout, testconsts.Interval).Should(BeTrue(), "Should fetch capp.")
	})

	It("Should delete all Capp dependent resources when Capp is deleted", func() {
		baseCapp := mock.CreateBaseCapp()
		desiredCapp := utilst.CreateCapp(k8sClient, baseCapp)

		assertionCapp := utilst.GetCappWithPlacementAnnotation(k8sClient, desiredCapp.Name, desiredCapp.Namespace)
		mwNamespace := assertionCapp.Annotations[testconsts.AnnotationKeyHasPlacement]

		By("Deletes capp")
		utilst.DeleteCapp(k8sClient, assertionCapp)

		By("Checks if ManifestWork was deleted")
		manifestWork := &workv1.ManifestWork{}
		Eventually(func() bool {
			mwName := namespaceManifestWorkPrefix + assertionCapp.Namespace + "-" + assertionCapp.Name
			Expect(k8sClient.Get(context.Background(), client.ObjectKey{Name: mwName, Namespace: mwNamespace}, manifestWork)).Should(Succeed())
			return utilst.DoesResourceExist(k8sClient, manifestWork)
		}, testconsts.Timeout, testconsts.Interval).ShouldNot(BeFalse())
	})

	It("Should copy the secret from volumes to ManifestWork", func() {
		baseSecret := mock.CreateSecret()
		secret := utilst.CreateSecret(k8sClient, baseSecret)
		baseCapp := mock.CreateBaseCapp()
		baseCapp.Spec.ConfigurationSpec.Template.Spec.Volumes = append(
			baseCapp.Spec.ConfigurationSpec.Template.Spec.Volumes,
			corev1.Volume{
				Name: secret.Name,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: secret.Name,
					},
				},
			})

		baseCapp.Spec.ConfigurationSpec.Template.Spec.Containers[0].VolumeMounts = append(
			baseCapp.Spec.ConfigurationSpec.Template.Spec.Containers[0].VolumeMounts,
			corev1.VolumeMount{
				Name:      secret.Name,
				MountPath: "/test/mount",
			})

		verifySecretOrConfigMapCopy(baseCapp, secret.Name, secret.Namespace, baseSecret.TypeMeta.Kind)
	})

	It("Should copy the secret from environment variables to ManifestWork (envFrom)", func() {
		baseSecret := mock.CreateSecret()
		secret := utilst.CreateSecret(k8sClient, baseSecret)
		baseCapp := mock.CreateBaseCapp()
		baseCapp.Spec.ConfigurationSpec.Template.Spec.Containers[0].EnvFrom = append(
			baseCapp.Spec.ConfigurationSpec.Template.Spec.Containers[0].EnvFrom,
			corev1.EnvFromSource{
				SecretRef: &corev1.SecretEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: secret.Name,
					},
				},
			})

		verifySecretOrConfigMapCopy(baseCapp, secret.Name, secret.Namespace, baseSecret.TypeMeta.Kind)
	})

	It("Should copy the secret from environment variables to ManifestWork (valueFrom)", func() {
		baseSecret := mock.CreateSecret()
		secret := utilst.CreateSecret(k8sClient, baseSecret)
		baseCapp := mock.CreateBaseCapp()
		baseCapp.Spec.ConfigurationSpec.Template.Spec.Containers[0].Env = append(
			baseCapp.Spec.ConfigurationSpec.Template.Spec.Containers[0].Env,
			corev1.EnvVar{
				Name: envVarName,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: secret.Name,
						},
						Key: mock.DataKey,
					},
				},
			})

		verifySecretOrConfigMapCopy(baseCapp, secret.Name, secret.Namespace, baseSecret.TypeMeta.Kind)
	})

	It("Should copy the configMap from volumes to ManifestWork", func() {
		baseConfigMap := mock.CreateConfigMap()
		configmap := utilst.CreateConfigMap(k8sClient, baseConfigMap)
		baseCapp := mock.CreateBaseCapp()
		baseCapp.Spec.ConfigurationSpec.Template.Spec.Volumes = append(
			baseCapp.Spec.ConfigurationSpec.Template.Spec.Volumes,
			corev1.Volume{
				Name: configmap.Name,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: configmap.Name,
						},
					},
				},
			})

		baseCapp.Spec.ConfigurationSpec.Template.Spec.Containers[0].VolumeMounts = append(
			baseCapp.Spec.ConfigurationSpec.Template.Spec.Containers[0].VolumeMounts,
			corev1.VolumeMount{
				Name:      configmap.Name,
				MountPath: "/test/mount",
			})

		verifySecretOrConfigMapCopy(baseCapp, configmap.Name, configmap.Namespace, baseConfigMap.TypeMeta.Kind)
	})

	It("Should copy the ConfigMap from environment variables to ManifestWork (envFrom)", func() {
		baseConfigMap := mock.CreateConfigMap()
		configmap := utilst.CreateConfigMap(k8sClient, baseConfigMap)
		baseCapp := mock.CreateBaseCapp()
		baseCapp.Spec.ConfigurationSpec.Template.Spec.Containers[0].EnvFrom = append(
			baseCapp.Spec.ConfigurationSpec.Template.Spec.Containers[0].EnvFrom,
			corev1.EnvFromSource{
				ConfigMapRef: &corev1.ConfigMapEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: configmap.Name,
					},
				},
			})

		verifySecretOrConfigMapCopy(baseCapp, configmap.Name, configmap.Namespace, baseConfigMap.TypeMeta.Kind)
	})

	It("Should copy the ConfigMap from environment variables to ManifestWork (valueFrom)", func() {
		baseConfigMap := mock.CreateConfigMap()
		configmap := utilst.CreateConfigMap(k8sClient, baseConfigMap)
		baseCapp := mock.CreateBaseCapp()
		baseCapp.Spec.ConfigurationSpec.Template.Spec.Containers[0].Env = append(
			baseCapp.Spec.ConfigurationSpec.Template.Spec.Containers[0].Env,
			corev1.EnvVar{
				Name: envVarName,
				ValueFrom: &corev1.EnvVarSource{
					ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: configmap.Name,
						},
						Key: mock.DataKey,
					},
				},
			})

		verifySecretOrConfigMapCopy(baseCapp, configmap.Name, configmap.Namespace, baseConfigMap.TypeMeta.Kind)
	})

	It("Should copy the secret from imagePullSecrets to ManifestWork ", func() {
		baseSecret := mock.CreateSecret()
		secret := utilst.CreateSecret(k8sClient, baseSecret)
		baseCapp := mock.CreateBaseCapp()
		baseCapp.Spec.ConfigurationSpec.Template.Spec.ImagePullSecrets = append(
			baseCapp.Spec.ConfigurationSpec.Template.Spec.ImagePullSecrets,
			corev1.LocalObjectReference{
				Name: secret.Name,
			})

		verifySecretOrConfigMapCopy(baseCapp, secret.Name, secret.Namespace, baseSecret.TypeMeta.Kind)
	})

	It("Should copy the RoleBindings from Capp's namespace to ManifestWork ", func() {
		baseRole := mock.CreateCappRole()
		role := utilst.CreateRole(k8sClient, baseRole)
		baseRoleBinding := mock.CreateCappRoleBinding()
		roleBinding := utilst.CreateRoleBinding(k8sClient, baseRoleBinding)
		baseCapp := mock.CreateBaseCapp()
		desiredCapp := utilst.CreateCapp(k8sClient, baseCapp)

		assertionCapp := utilst.GetCappWithPlacementAnnotation(k8sClient, desiredCapp.Name, desiredCapp.Namespace)
		mwNamespace := assertionCapp.Annotations[testconsts.AnnotationKeyHasPlacement]

		By("Checks ManifestWork was synced with Role and RoleBinding")
		manifestWork := &workv1.ManifestWork{}

		Eventually(func() bool {
			mwName := namespaceManifestWorkPrefix + assertionCapp.Namespace + "-" + assertionCapp.Name
			_ = k8sClient.Get(context.Background(), client.ObjectKey{Name: mwName, Namespace: mwNamespace}, manifestWork)
			return utilst.IsRbacObjInManifestWork(*manifestWork, assertionCapp.Name, role.Namespace, "Role") &&
				utilst.IsRbacObjInManifestWork(*manifestWork, assertionCapp.Name, roleBinding.Namespace, "RoleBinding")
		}, testconsts.Timeout, testconsts.Interval).Should(BeTrue())
	})

	It("Should copy the Capp and Capp's namespace to ManifestWork ", func() {
		baseCapp := mock.CreateBaseCapp()
		desiredCapp := utilst.CreateCapp(k8sClient, baseCapp)

		assertionCapp := utilst.GetCappWithPlacementAnnotation(k8sClient, desiredCapp.Name, desiredCapp.Namespace)
		mwNamespace := assertionCapp.Annotations[testconsts.AnnotationKeyHasPlacement]

		By("Checks ManifestWork was synced with Capp and namespace")
		manifestWork := &workv1.ManifestWork{}

		Eventually(func() bool {
			mwName := namespaceManifestWorkPrefix + assertionCapp.Namespace + "-" + assertionCapp.Name
			_ = k8sClient.Get(context.Background(), client.ObjectKey{Name: mwName, Namespace: mwNamespace}, manifestWork)
			ns, err := utilst.IsObjInManifestWork(k8sClient, *manifestWork, assertionCapp.Namespace, "", &corev1.Namespace{}, "Namespace")
			Expect(err).Should(BeNil())
			capp, err := utilst.IsObjInManifestWork(k8sClient, *manifestWork, assertionCapp.Name, assertionCapp.Namespace, &cappv1alpha1.Capp{}, "Capp")
			Expect(err).Should(BeNil())
			return ns && capp
		}, testconsts.Timeout, testconsts.Interval).Should(BeTrue())
	})

	It("Should copy the updated Capp to ManifestWork ", func() {
		baseCapp := mock.CreateBaseCapp()
		desiredCapp := utilst.CreateCapp(k8sClient, baseCapp)

		assertionCapp := utilst.GetCappWithPlacementAnnotation(k8sClient, desiredCapp.Name, desiredCapp.Namespace)
		mwNamespace := assertionCapp.Annotations[testconsts.AnnotationKeyHasPlacement]

		By("Update Capp")
		assertionCapp.Spec.Site = testconsts.Cluster2
		Expect(k8sClient.Update(context.Background(), assertionCapp)).Should(Succeed())

		By("Checks Capp site is not nil")
		manifestWork := &workv1.ManifestWork{}
		mwName := namespaceManifestWorkPrefix + assertionCapp.Namespace + "-" + assertionCapp.Name
		Eventually(func() interface{} {
			_ = k8sClient.Get(context.Background(), client.ObjectKey{Name: mwName, Namespace: mwNamespace}, manifestWork)
			return utilst.GetCappFromManifestWork(*manifestWork).Object["spec"].(map[string]interface{})["site"]
		}, testconsts.Timeout, testconsts.Interval).ShouldNot(BeNil())

		By("Checks ManifestWork was synced with new Capp")
		Eventually(func() string {
			_ = k8sClient.Get(context.Background(), client.ObjectKey{Name: mwName, Namespace: mwNamespace}, manifestWork)
			return utilst.GetCappFromManifestWork(*manifestWork).Object["spec"].(map[string]interface{})["site"].(string)
		}, testconsts.Timeout, testconsts.Interval).Should(Equal(testconsts.Cluster2))

		By("Checks if managed by label exists")
		Eventually(func() bool {
			_ = k8sClient.Get(context.Background(), client.ObjectKey{Name: mwName, Namespace: mwNamespace}, manifestWork)
			cappObject := utilst.GetCappFromManifestWork(*manifestWork)
			cappLables := cappObject.GetLabels()
			if cappLables != nil {
				rcsLabel, ok := cappLables[testconsts.MangedByLableKey]
				if ok {
					return rcsLabel == testconsts.MangedByLabelValue
				}
			}
			return false
		}, testconsts.Timeout, testconsts.Interval).Should(BeTrue())

	})
})
