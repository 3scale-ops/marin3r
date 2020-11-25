package controllers

import (
	"context"
	"time"

	marin3rv1alpha1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator.marin3r/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("EnvoyConfigRevision controller", func() {
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
				Name: "instance",
			},
			Spec: operatorv1alpha1.DiscoveryServiceSpec{
				Image:                     "image",
				DiscoveryServiceNamespace: namespace,
				EnabledNamespaces:         []string{namespace},
			},
		}
		err = k8sClient.Create(context.Background(), ds)
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() bool {
			err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "instance"}, ds)
			if err != nil {
				return false
			}
			return true
		}, 30*time.Second, 5*time.Second).Should(BeTrue())

	})

	Context("DiscoveryService", func() {

		It("Create the required resources", func() {

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

			By("waiting for the discovery service ClusterRole to be created")

			{
				cr := &rbacv1.ClusterRole{}
				Eventually(func() bool {
					if err := k8sClient.Get(
						context.Background(),
						types.NamespacedName{Name: OwnedObjectName(ds)},
						cr,
					); err != nil {
						return false
					}
					return true
				}, 30*time.Second, 5*time.Second).Should(BeTrue())
			}

			By("waiting for the discovery service ClusterRoleBinding to be created")

			{
				crb := &rbacv1.ClusterRoleBinding{}
				Eventually(func() bool {
					if err := k8sClient.Get(
						context.Background(),
						types.NamespacedName{Name: OwnedObjectName(ds)},
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

			By("waiting for the discovery service MutatingWebhookConfiguration to be created")
			{

				mwc := &admissionregistrationv1beta1.MutatingWebhookConfiguration{}
				Eventually(func() bool {
					if err := k8sClient.Get(
						context.Background(),
						types.NamespacedName{Name: OwnedObjectName(ds)},
						mwc,
					); err != nil {
						return false
					}
					return true
				}, 30*time.Second, 5*time.Second).Should(BeTrue())
			}

			By("checking the namespaces in 'spec.enabledNamespaces' have the required label")

			{
				for _, name := range ds.Spec.EnabledNamespaces {
					ns := &corev1.Namespace{}
					Eventually(func() bool {
						if err := k8sClient.Get(context.Background(), types.NamespacedName{Name: name}, ns); err != nil {
							return false
						}
						return true
					}, 30*time.Second, 5*time.Second).Should(BeTrue())
					Expect(ns.Labels[operatorv1alpha1.DiscoveryServiceLabelKey]).To(Equal(ds.GetName()))
				}
			}

			By("checking the namespaces in 'spec.enabledNamespaces' have an EnvoyBootstrap resource each")

			{
				for _, ns := range ds.Spec.EnabledNamespaces {
					eb := &marin3rv1alpha1.EnvoyBootstrap{}
					Eventually(func() bool {
						if err := k8sClient.Get(context.Background(), types.NamespacedName{Name: ds.GetName(), Namespace: ns}, eb); err != nil {
							return false
						}
						return true
					}, 30*time.Second, 5*time.Second).Should(BeTrue())
				}
			}
		})
	})

})
