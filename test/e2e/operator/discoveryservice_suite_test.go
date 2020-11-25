package e2e

import (
	"context"
	"time"

	testutil "github.com/3scale/marin3r/test/e2e/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator.marin3r/v1alpha1"
)

const (
	image           string = "quay.io/3scale/marin3r:test"
	targetNamespace string = "default"
)

var _ = Describe("DiscoveryService intall and lifecycle", func() {
	var testNamespace string
	var ds *operatorv1alpha1.DiscoveryService

	BeforeEach(func() {
		// Create a namespace for each block
		testNamespace = "test-ns-" + nameGenerator.Generate()

		// Add any setup steps that needs to be executed before each test
		ns := &corev1.Namespace{
			TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Namespace"},
			ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
		}

		err := k8sClient.Create(context.Background(), ns)
		Expect(err).ToNot(HaveOccurred())

		n := &corev1.Namespace{}
		Eventually(func() bool {
			err := k8sClient.Get(context.Background(), types.NamespacedName{Name: testNamespace}, n)
			if err != nil {
				return false
			}
			return true
		}, 60*time.Second, 5*time.Second).Should(BeTrue())

		By("creating a DiscoveryService instance")
		ds = &operatorv1alpha1.DiscoveryService{
			ObjectMeta: metav1.ObjectMeta{
				Name: "instance",
			},
			Spec: operatorv1alpha1.DiscoveryServiceSpec{
				Image:                     image,
				DiscoveryServiceNamespace: targetNamespace,
				EnabledNamespaces:         []string{testNamespace},
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
		}, 60*time.Second, 5*time.Second).Should(BeTrue())

	})

	AfterEach(func() {

		// Delete DiscoveryService instance
		logger.Info("Cleanup", "DiscoveryService", ds.GetName())
		err := k8sClient.Delete(context.Background(), ds)
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() int {
			return testutil.ReadyReplicas(
				k8sClient,
				targetNamespace,
				client.MatchingLabels{
					"app.kubernetes.io/name":       "marin3r",
					"app.kubernetes.io/managed-by": "marin3r-operator",
					"app.kubernetes.io/component":  "discovery-service",
					"app.kubernetes.io/instance":   ds.GetName(),
				},
			)
		}, 60*time.Second, 5*time.Second).Should(Equal(0))

		// Delete the namespace
		ns := &corev1.Namespace{
			TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Namespace"},
			ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
		}
		logger.Info("Cleanup", "Namespace", testNamespace)
		err = k8sClient.Delete(context.Background(), ns, client.PropagationPolicy(metav1.DeletePropagationForeground))
		Expect(err).ToNot(HaveOccurred())
	})

	Context("DiscoveryService installation and lifecycle", func() {

		BeforeEach(func() {
			Eventually(func() int {
				return testutil.ReadyReplicas(
					k8sClient,
					targetNamespace,
					client.MatchingLabels{
						"app.kubernetes.io/name":       "marin3r",
						"app.kubernetes.io/managed-by": "marin3r-operator",
						"app.kubernetes.io/component":  "discovery-service",
						"app.kubernetes.io/instance":   ds.GetName(),
					},
				)
			}, 60*time.Second, 5*time.Second).Should(Equal(1))
		})

		It("triggers a rollout on certificate change", func() {

			By("deleting the Secret that holds the current certificate to force recreation")
			serverCert := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: ds.GetServerCertificateOptions().SecretName,
					// Name:      "marin3r-server-cert-instance",
					Namespace: targetNamespace,
				},
			}
			err := k8sClient.Delete(context.Background(), serverCert)
			Expect(err).ToNot(HaveOccurred())

			By("waiting for the Deployment generation to increase")
			Eventually(func() int {
				dep := &appsv1.Deployment{}
				key := types.NamespacedName{
					Name:      "marin3r-instance",
					Namespace: targetNamespace,
				}
				err := k8sClient.Get(context.Background(), key, dep)
				Expect(err).ToNot(HaveOccurred())
				return int(dep.Status.ObservedGeneration)
			}, 60*time.Second, 5*time.Second).Should(Equal(2))

			By("waiting for the ready replicas to be 1")
			Eventually(func() int {
				return testutil.ReadyReplicas(
					k8sClient,
					targetNamespace,
					client.MatchingLabels{
						"app.kubernetes.io/name":       "marin3r",
						"app.kubernetes.io/managed-by": "marin3r-operator",
						"app.kubernetes.io/component":  "discovery-service",
						"app.kubernetes.io/instance":   ds.GetName(),
					},
				)
			}, 60*time.Second, 5*time.Second).Should(Equal(1))
		})

		It("reconciles the discovery service deployment", func() {

			ds.Spec.Debug = true
			err := k8sClient.Update(context.Background(), ds)
			Expect(err).ToNot(HaveOccurred())

			By("waiting for the Deployment generation to increase")
			Eventually(func() int {
				dep := &appsv1.Deployment{}
				key := types.NamespacedName{
					Name:      "marin3r-instance",
					Namespace: targetNamespace,
				}
				err := k8sClient.Get(context.Background(), key, dep)
				Expect(err).ToNot(HaveOccurred())
				return int(dep.Status.ObservedGeneration)
			}, 60*time.Second, 5*time.Second).Should(Equal(2))
		})

	})
})
