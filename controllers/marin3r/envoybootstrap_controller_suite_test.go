package controllers

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	marin3rv1alpha1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator/v1alpha1"
)

var _ = Describe("EnvoyBootstrap controller", func() {
	var namespace string

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

	})

	AfterEach(func() {

		// Delete the namespace
		testNamespace := &v1.Namespace{
			TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Namespace"},
			ObjectMeta: metav1.ObjectMeta{Name: namespace},
		}
		// Add any teardown steps that needs to be executed after each test
		err := k8sClient.Delete(context.Background(), testNamespace, client.PropagationPolicy(metav1.DeletePropagationForeground))
		Expect(err).ToNot(HaveOccurred())

		n := &v1.Namespace{}
		Eventually(func() bool {
			err := k8sClient.Get(context.Background(), types.NamespacedName{Name: namespace}, n)
			if err != nil && errors.IsNotFound(err) {
				return false
			}
			return true
		}, 30*time.Second, 5*time.Second).Should(BeTrue())
	})

	Context("an EnvoyBootstrap is created", func() {
		var eb *marin3rv1alpha1.EnvoyBootstrap
		var ds *operatorv1alpha1.DiscoveryService

		BeforeEach(func() {

			By("Creating a DiscoveryService instance")
			ds = &operatorv1alpha1.DiscoveryService{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "instance",
					Namespace: namespace,
				},
				Spec: operatorv1alpha1.DiscoveryServiceSpec{
					Image: pointer.StringPtr("image"),
				},
			}
			err := k8sClient.Create(context.Background(), ds)
			Expect(err).ToNot(HaveOccurred())

			eb = &marin3rv1alpha1.EnvoyBootstrap{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: namespace},
				Spec: marin3rv1alpha1.EnvoyBootstrapSpec{
					DiscoveryService: "instance",
					ClientCertificate: &marin3rv1alpha1.ClientCertificate{
						Directory:  "/tls",
						SecretName: "my-cert",
						Duration: metav1.Duration{
							Duration: func() time.Duration {
								d, _ := time.ParseDuration("5m")
								return d
							}(),
						},
					},
					EnvoyStaticConfig: &marin3rv1alpha1.EnvoyStaticConfig{
						ConfigMapNameV2:       "bootstrap-v2",
						ConfigMapNameV3:       "bootstrap-v3",
						ConfigFile:            "/config.json",
						ResourcesDir:          "/resources",
						RtdsLayerResourceName: "runtime",
						AdminBindAddress:      "127.0.0.1:9901",
						AdminAccessLogPath:    "/dev/null",
					},
				},
			}
			err = k8sClient.Create(context.Background(), eb)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "test", Namespace: namespace}, eb)
				if err != nil {
					return false
				}
				return true
			}, 30*time.Second, 5*time.Second).Should(BeTrue())

		})

		It("should create the ConfigMaps and the DiscoveryServiceCertificate for the envoy client", func() {

			By("Checking that the DiscoveryServiceCertificate has been created")
			{
				dsc := &operatorv1alpha1.DiscoveryServiceCertificate{}
				Eventually(func() bool {
					if err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "my-cert", Namespace: namespace}, dsc); err != nil {
						return false
					}
					return true
				}, 30*time.Second, 5*time.Second).Should(BeTrue())

				Expect(dsc.Spec.SecretRef.Name).To(Equal(eb.Spec.ClientCertificate.SecretName))
				Expect(dsc.Spec.ValidFor).To(Equal(int64(eb.Spec.ClientCertificate.Duration.Seconds())))
			}

			By("Checking that the v2 bootstrap ConfigMap has been created")
			{
				cm := &corev1.ConfigMap{}
				Eventually(func() bool {
					if err := k8sClient.Get(context.Background(), types.NamespacedName{Name: eb.Spec.EnvoyStaticConfig.ConfigMapNameV2, Namespace: namespace}, cm); err != nil {
						return false
					}
					return true
				}, 30*time.Second, 5*time.Second).Should(BeTrue())
			}

			By("Checking that the v3 bootstrap ConfigMap has been created")
			{
				cm := &corev1.ConfigMap{}
				Eventually(func() bool {
					if err := k8sClient.Get(context.Background(), types.NamespacedName{Name: eb.Spec.EnvoyStaticConfig.ConfigMapNameV3, Namespace: namespace}, cm); err != nil {
						return false
					}
					return true
				}, 30*time.Second, 5*time.Second).Should(BeTrue())
			}
		})
	})
})
