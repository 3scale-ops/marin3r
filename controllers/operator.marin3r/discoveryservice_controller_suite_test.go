package controllers

import (
	"context"
	"time"

	marin3rv1alpha1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator.marin3r/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
)

var _ = Describe("DiscoveryService controller", func() {
	var namespace string
	var ds *operatorv1alpha1.DiscoveryService

	BeforeEach(func() {
		// Create a namespace for each block
		namespace = "test-ns-" + nameGenerator.Generate()

		// Add any setup steps that needs to be executed before each test
		testNamespace := &v1.Namespace{
			TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Namespace"},
			ObjectMeta: metav1.ObjectMeta{Name: namespace},
		}

		err := k8sClient.Create(context.Background(), testNamespace)
		Expect(err).ToNot(HaveOccurred())

		n := &v1.Namespace{}
		Eventually(func() bool {
			err := k8sClient.Get(context.Background(), types.NamespacedName{Name: namespace}, n)
			if err != nil {
				return false
			}
			return true
		}, 30*time.Second, 5*time.Second).Should(BeTrue())

		By("creating a DiscoveryService instance")
		ds = &operatorv1alpha1.DiscoveryService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "instance",
				Namespace: namespace,
			},
			Spec: operatorv1alpha1.DiscoveryServiceSpec{
				Image: pointer.StringPtr("image"),
			},
		}
		err = k8sClient.Create(context.Background(), ds)
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() bool {
			err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "instance", Namespace: namespace}, ds)
			if err != nil {
				return false
			}
			return true
		}, 30*time.Second, 5*time.Second).Should(BeTrue())

	})

	Context("DiscoveryService", func() {

		It("creates the required resources", func() {

			By("waiting for the root CA DiscoveryServiceCertificate to be created")
			{
				dsc := &operatorv1alpha1.DiscoveryServiceCertificate{}
				Eventually(func() bool {
					if err := k8sClient.Get(
						context.Background(),
						types.NamespacedName{Name: getCACertName(ds), Namespace: namespace},
						dsc,
					); err != nil {
						return false
					}
					return true
				}, 30*time.Second, 5*time.Second).Should(BeTrue())

				Expect(dsc.Spec.SecretRef.Name).To(Equal(ds.GetRootCertificateAuthorityOptions().SecretName))
				Expect(dsc.Spec.ValidFor).To(Equal(int64(ds.GetRootCertificateAuthorityOptions().Duration.Seconds())))
			}

			By("waiting for the root CA DiscoveryServiceCertificate to be created")
			{
				dsc := &operatorv1alpha1.DiscoveryServiceCertificate{}
				Eventually(func() bool {
					if err := k8sClient.Get(
						context.Background(),
						types.NamespacedName{Name: getServerCertName(ds), Namespace: namespace},
						dsc,
					); err != nil {
						return false
					}
					return true
				}, 30*time.Second, 5*time.Second).Should(BeTrue())

				Expect(dsc.Spec.SecretRef.Name).To(Equal(ds.GetServerCertificateOptions().SecretName))
				Expect(dsc.Spec.ValidFor).To(Equal(int64(ds.GetServerCertificateOptions().Duration.Seconds())))
			}

			By("waiting for the discovery service ServiceAccount to be created")

			{
				sa := &corev1.ServiceAccount{}
				Eventually(func() bool {
					if err := k8sClient.Get(
						context.Background(),
						types.NamespacedName{Name: OwnedObjectName(ds), Namespace: namespace},
						sa,
					); err != nil {
						return false
					}
					return true
				}, 30*time.Second, 5*time.Second).Should(BeTrue())
			}

			By("waiting for the discovery service Role to be created")

			{
				cr := &rbacv1.Role{}
				Eventually(func() bool {
					if err := k8sClient.Get(
						context.Background(),
						types.NamespacedName{Name: OwnedObjectName(ds), Namespace: namespace},
						cr,
					); err != nil {
						return false
					}
					return true
				}, 30*time.Second, 5*time.Second).Should(BeTrue())
			}

			By("waiting for the discovery service RoleBinding to be created")

			{
				crb := &rbacv1.RoleBinding{}
				Eventually(func() bool {
					if err := k8sClient.Get(
						context.Background(),
						types.NamespacedName{Name: OwnedObjectName(ds), Namespace: namespace},
						crb,
					); err != nil {
						return false
					}
					return true
				}, 30*time.Second, 5*time.Second).Should(BeTrue())
			}

			By("waiting for the discovery service Deployment to be created")

			{
				dep := &appsv1.Deployment{}
				Eventually(func() bool {
					if err := k8sClient.Get(
						context.Background(),
						types.NamespacedName{Name: OwnedObjectName(ds), Namespace: namespace},
						dep,
					); err != nil {
						return false
					}
					return true
				}, 30*time.Second, 5*time.Second).Should(BeTrue())
			}

			By("waiting for the discovery service Service to be created")

			{
				svc := &corev1.Service{}
				Eventually(func() bool {
					if err := k8sClient.Get(
						context.Background(),
						types.NamespacedName{Name: ds.GetServiceConfig().Name, Namespace: namespace},
						svc,
					); err != nil {
						return false
					}
					return true
				}, 30*time.Second, 5*time.Second).Should(BeTrue())
			}

			By("checking the namespaces in 'spec.enabledNamespaces' have an EnvoyBootstrap resource each")

			{
				eb := &marin3rv1alpha1.EnvoyBootstrap{}
				Eventually(func() bool {
					if err := k8sClient.Get(context.Background(), types.NamespacedName{Name: ds.GetName(), Namespace: namespace}, eb); err != nil {
						return false
					}
					return true
				}, 30*time.Second, 5*time.Second).Should(BeTrue())
			}
		})
	})

})
