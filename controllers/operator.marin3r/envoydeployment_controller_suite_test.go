package controllers

import (
	"context"
	"time"

	marin3rv1alpha1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	operatorv1alpha1 "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
)

var _ = Describe("EnvoyDeployment controller", func() {
	var namespace string
	var ed *operatorv1alpha1.EnvoyDeployment

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
		Eventually(func() error {
			return k8sClient.Get(context.Background(), types.NamespacedName{Name: namespace}, n)
		}, 30*time.Second, 5*time.Second).ShouldNot(HaveOccurred())

		By("creating a DiscoveryService instance")
		ds := &operatorv1alpha1.DiscoveryService{
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
		Eventually(func() error {
			return k8sClient.Get(context.Background(), types.NamespacedName{Name: "instance", Namespace: namespace}, ds)
		}, 30*time.Second, 5*time.Second).ShouldNot(HaveOccurred())

		By("creating an EnvoyConfig instance")
		ec := &marin3rv1alpha1.EnvoyConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "config", Namespace: namespace},
			Spec: marin3rv1alpha1.EnvoyConfigSpec{
				EnvoyAPI:       pointer.StringPtr("v3"),
				NodeID:         "test-node",
				EnvoyResources: &marin3rv1alpha1.EnvoyResources{},
			},
		}
		err = k8sClient.Create(context.Background(), ec)
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() bool {
			err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "config", Namespace: namespace}, ec)
			return err == nil
		}, 30*time.Second, 5*time.Second).Should(BeTrue())

		By("creating a EnvoyDeployment instance")
		ed = &operatorv1alpha1.EnvoyDeployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "instance",
				Namespace: namespace,
			},
			Spec: operatorv1alpha1.EnvoyDeploymentSpec{
				DiscoveryServiceRef: ds.GetName(),
				EnvoyConfigRef:      ec.GetName(),
			},
		}
		err = k8sClient.Create(context.Background(), ed)
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() error {
			return k8sClient.Get(context.Background(), types.NamespacedName{Name: "instance", Namespace: namespace}, ed)
		}, 30*time.Second, 5*time.Second).ShouldNot(HaveOccurred())

	})

	Context("EnvoyDeployment", func() {

		It("adds a finalizer to the resource", func() {

			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "instance", Namespace: namespace}, ed)
				Expect(err).ToNot(HaveOccurred())
				return len(ed.GetFinalizers()) > 0
			}, 30*time.Second, 5*time.Second).Should(BeTrue())
		})

		It("creates the required resources", func() {

			By("waiting for the EnvoyBootstrap resource to be created")
			{
				eb := &marin3rv1alpha1.EnvoyBootstrap{}
				Eventually(func() error {
					return k8sClient.Get(
						context.Background(),
						types.NamespacedName{Name: "marin3r-envoydeployment-instance", Namespace: namespace},
						eb,
					)
				}, 30*time.Second, 5*time.Second).ShouldNot(HaveOccurred())
			}

			By("waiting for the envoy Deployment to be created")
			{
				dep := &appsv1.Deployment{}
				key := types.NamespacedName{Name: "marin3r-envoydeployment-instance", Namespace: namespace}
				Eventually(func() error {
					return k8sClient.Get(context.Background(), key, dep)
				}, 30*time.Second, 5*time.Second).ShouldNot(HaveOccurred())
			}
		})
	})

})
