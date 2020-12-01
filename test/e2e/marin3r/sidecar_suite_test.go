package e2e

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"time"

	marin3rv1alpha1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator/v1alpha1"
	"github.com/3scale/marin3r/pkg/envoy"
	testutil "github.com/3scale/marin3r/test/e2e/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/phayes/freeport"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Envpoy sidecars", func() {
	var testNamespace string

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
				Name:      "instance",
				Namespace: testNamespace,
			},
			Spec: operatorv1alpha1.DiscoveryServiceSpec{
				Image: pointer.StringPtr(image),
			},
		}
		err = k8sClient.Create(context.Background(), ds)
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() int {
			dep := &appsv1.Deployment{}
			key := types.NamespacedName{
				Name:      "marin3r-instance",
				Namespace: testNamespace,
			}
			if err := k8sClient.Get(context.Background(), key, dep); err != nil {
				return 0
			}
			return int(dep.Status.ReadyReplicas)
		}, 60*time.Second, 5*time.Second).Should(Equal(1))
	})

	AfterEach(func() {

		// Delete the namespace
		ns := &corev1.Namespace{
			TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Namespace"},
			ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
		}
		logger.Info("Cleanup", "Namespace", testNamespace)
		err := k8sClient.Delete(context.Background(), ns, client.PropagationPolicy(metav1.DeletePropagationForeground))
		Expect(err).ToNot(HaveOccurred())
	})

	Context("Sidecar injection", func() {
		var localPort int
		var nodeID string
		var ec *marin3rv1alpha1.EnvoyConfig

		BeforeEach(func() {

			var err error
			localPort, err = freeport.GetFreePort()
			Expect(err).ToNot(HaveOccurred())
			nodeID = nameGenerator.Generate()

		})

		It("injects an envoy sidecar container using v2 config", func() {

			By("applaying an EnvoyConfig that will configure the envoy sidecar through service discovery")
			key := types.NamespacedName{Name: "nginx-envoyconfig", Namespace: testNamespace}
			ec = testutil.GenerateEnvoyConfig(key, nodeID, envoy.APIv2,
				func() map[string]envoy.Resource {
					k, v := testutil.EndpointV2("nginx", "127.0.0.1", 80)
					return map[string]envoy.Resource{k: v}
				},
				func() map[string]envoy.Resource {
					k, v := testutil.ClusterWithEdsV2("nginx")
					return map[string]envoy.Resource{k: v}
				},
				func() map[string]envoy.Resource {
					k, v := testutil.ProxyPassRouteV2("proxypass", "nginx")
					return map[string]envoy.Resource{k: v}
				},
				func() map[string]envoy.Resource {
					k, v := testutil.HTTPListenerWithRdsV2("http", "proxypass", testutil.GetAddressV2("0.0.0.0", envoyListenerPort), nil)
					return map[string]envoy.Resource{k: v}
				},
				nil,
			)
			err := k8sClient.Create(context.Background(), ec)
			Expect(err).NotTo(HaveOccurred())

			By("creating a Deployment with the required labels and annotations")
			key = types.NamespacedName{Name: "nginx", Namespace: testNamespace}
			dep := testutil.GenerateDeploymentWithInjection(key, nodeID, "v2", "v1.16.0", envoyListenerPort)
			err = k8sClient.Create(context.Background(), dep)
			Expect(err).ToNot(HaveOccurred())

			selector := client.MatchingLabels{testutil.DeploymentLabelKey: testutil.DeploymentLabelValue}
			Eventually(func() int {
				return testutil.ReadyReplicas(k8sClient, testNamespace, selector)
			}, 60*time.Second, 5*time.Second).Should(Equal(1))

			By("checking that the Pods were mutated to add the envoy sidecar")
			podList := &corev1.PodList{}
			err = k8sClient.List(context.Background(), podList,
				[]client.ListOption{selector, client.InNamespace(testNamespace)}...)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(podList.Items)).To(Equal(1))
			Expect(len(podList.Items[0].Spec.Containers)).To(Equal(2))

			By(fmt.Sprintf("forwarding the Pod's port to localhost: %v", localPort))
			stopCh := make(chan struct{})
			readyCh := make(chan struct{})
			defer close(stopCh)
			go func() {
				defer GinkgoRecover()
				fw, err := testutil.NewTestPortForwarder(cfg, podList.Items[0], uint32(localPort), envoyListenerPort, GinkgoWriter, stopCh, readyCh)
				Expect(err).ToNot(HaveOccurred())
				err = fw.ForwardPorts()
				Expect(err).ToNot(HaveOccurred())
			}()

			ticker := time.NewTimer(10 * time.Second)
			select {
			case <-ticker.C:
				Fail("timed out while waiting for port forward")
			case <-readyCh:
				ticker.Stop()
				break
			}

			By("doing a request against envoy sidecar, that should be forwarded to the nginx container")
			var resp *http.Response
			Eventually(func() error {
				resp, err = http.Get(fmt.Sprintf("http://localhost:%v", localPort))
				return err
			}, 60*time.Second, 5*time.Second).ShouldNot(HaveOccurred())

			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			scanner := bufio.NewScanner(resp.Body)
			Expect(scanner.Scan()).To(BeTrue())
			Expect(scanner.Text()).To(Equal("Server address: 127.0.0.1:80"))
			Expect(scanner.Scan()).To(BeTrue())
			Expect(scanner.Text()).To(Equal(fmt.Sprintf("Server name: %s", podList.Items[0].GetName())))

		})

		It("injects an envoy sidecar container using v3 config", func() {

			By("applaying an EnvoyConfig that will configure the envoy sidecar through service discovery")
			key := types.NamespacedName{Name: "nginx-envoyconfig", Namespace: testNamespace}
			ec = testutil.GenerateEnvoyConfig(key, nodeID, envoy.APIv3,
				func() map[string]envoy.Resource {
					k, v := testutil.EndpointV3("nginx", "127.0.0.1", 80)
					return map[string]envoy.Resource{k: v}
				},
				func() map[string]envoy.Resource {
					k, v := testutil.ClusterWithEdsV3("nginx")
					return map[string]envoy.Resource{k: v}
				},
				func() map[string]envoy.Resource {
					k, v := testutil.ProxyPassRouteV3("proxypass", "nginx")
					return map[string]envoy.Resource{k: v}
				},
				func() map[string]envoy.Resource {
					k, v := testutil.HTTPListenerWithRdsV3("http", "proxypass", testutil.GetAddressV3("0.0.0.0", envoyListenerPort), nil)
					return map[string]envoy.Resource{k: v}
				},
				nil,
			)
			err := k8sClient.Create(context.Background(), ec)
			Expect(err).NotTo(HaveOccurred())

			By("creating a Deployment with the required labels and annotations")
			key = types.NamespacedName{Name: "nginx", Namespace: testNamespace}
			dep := testutil.GenerateDeploymentWithInjection(key, nodeID, "v3", "v1.16.0", 8080)
			err = k8sClient.Create(context.Background(), dep)
			Expect(err).ToNot(HaveOccurred())

			selector := client.MatchingLabels{testutil.DeploymentLabelKey: testutil.DeploymentLabelValue}
			Eventually(func() int {
				return testutil.ReadyReplicas(k8sClient, testNamespace, selector)
			}, 60*time.Second, 5*time.Second).Should(Equal(1))

			By("checking that the Pods were mutated to add the envoy sidecar")
			podList := &corev1.PodList{}
			err = k8sClient.List(context.Background(), podList,
				[]client.ListOption{selector, client.InNamespace(testNamespace)}...)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(podList.Items)).To(Equal(1))
			Expect(len(podList.Items[0].Spec.Containers)).To(Equal(2))

			By(fmt.Sprintf("forwarding the Pod's port to localhost: %v", localPort))
			stopCh := make(chan struct{})
			readyCh := make(chan struct{})
			defer close(stopCh)
			go func() {
				defer GinkgoRecover()
				fw, err := testutil.NewTestPortForwarder(cfg, podList.Items[0], uint32(localPort), envoyListenerPort, GinkgoWriter, stopCh, readyCh)
				Expect(err).ToNot(HaveOccurred())
				err = fw.ForwardPorts()
				Expect(err).ToNot(HaveOccurred())
			}()

			ticker := time.NewTimer(10 * time.Second)
			select {
			case <-ticker.C:
				Fail("timed out while waiting for port forward")
			case <-readyCh:
				ticker.Stop()
				break
			}

			By("doing a request against envoy sidecar, that should be forwarded to the nginx container")
			var resp *http.Response
			Eventually(func() error {
				resp, err = http.Get(fmt.Sprintf("http://localhost:%v", localPort))
				return err
			}, 60*time.Second, 5*time.Second).ShouldNot(HaveOccurred())

			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			scanner := bufio.NewScanner(resp.Body)
			Expect(scanner.Scan()).To(BeTrue())
			Expect(scanner.Text()).To(Equal("Server address: 127.0.0.1:80"))
			Expect(scanner.Scan()).To(BeTrue())
			Expect(scanner.Text()).To(Equal(fmt.Sprintf("Server name: %s", podList.Items[0].GetName())))

		})

		It("upgrade from v2 sidecars to v3 sidecars", func() {
			var stopCh chan struct{}
			var readyCh chan struct{}
			var pod corev1.Pod

			By("applying an EnvoyConfig that will configure the envoy sidecar through service discovery")
			{
				key := types.NamespacedName{Name: "nginx-envoyconfig", Namespace: testNamespace}
				ec = testutil.GenerateEnvoyConfig(key, nodeID, envoy.APIv2,
					func() map[string]envoy.Resource { return nil },
					func() map[string]envoy.Resource { return nil },
					func() map[string]envoy.Resource {
						k, v := testutil.DirectResponseRouteV2("direct_response", "This is a v2 route")
						return map[string]envoy.Resource{k: v}
					},
					func() map[string]envoy.Resource {
						k, v := testutil.HTTPListenerWithRdsV2("http", "direct_response", testutil.GetAddressV2("0.0.0.0", envoyListenerPort), nil)
						return map[string]envoy.Resource{k: v}
					},
					nil,
				)
				err := k8sClient.Create(context.Background(), ec)
				Expect(err).NotTo(HaveOccurred())
			}

			By("creating a Deployment with the required labels and annotations")
			{
				key := types.NamespacedName{Name: "nginx", Namespace: testNamespace}
				dep := testutil.GenerateDeploymentWithInjection(key, nodeID, "v2", "v1.16.0", envoyListenerPort)
				err := k8sClient.Create(context.Background(), dep)
				Expect(err).ToNot(HaveOccurred())

				selector := client.MatchingLabels{testutil.DeploymentLabelKey: testutil.DeploymentLabelValue}
				Eventually(func() int {
					return testutil.ReadyReplicas(k8sClient, testNamespace, selector)
				}, 60*time.Second, 5*time.Second).Should(Equal(1))
			}

			By("checking that the Pod was mutated to add the envoy sidecar")
			{
				podList := &corev1.PodList{}
				selector := client.MatchingLabels{testutil.DeploymentLabelKey: testutil.DeploymentLabelValue}
				err := k8sClient.List(context.Background(), podList,
					[]client.ListOption{selector, client.InNamespace(testNamespace)}...)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(podList.Items)).To(Equal(1))
				Expect(len(podList.Items[0].Spec.Containers)).To(Equal(2))
				pod = podList.Items[0]
			}

			By(fmt.Sprintf("forwarding the Pod's port to localhost: %v", localPort))
			{
				stopCh = make(chan struct{})
				readyCh = make(chan struct{})
				go func() {
					defer GinkgoRecover()
					fw, err := testutil.NewTestPortForwarder(cfg, pod, uint32(localPort), envoyListenerPort, GinkgoWriter, stopCh, readyCh)
					Expect(err).ToNot(HaveOccurred())
					err = fw.ForwardPorts()
					Expect(err).ToNot(HaveOccurred())
				}()

				ticker := time.NewTimer(10 * time.Second)
				select {
				case <-ticker.C:
					Fail("timed out while waiting for port forward")
				case <-readyCh:
					ticker.Stop()
					break
				}
			}

			By("doing a request against envoy sidecar, that should answered directly by envoy with a 200")
			{
				var resp *http.Response
				var err error

				Eventually(func() error {
					resp, err = http.Get(fmt.Sprintf("http://localhost:%v", localPort))
					return err
				}, 60*time.Second, 5*time.Second).ShouldNot(HaveOccurred())

				defer resp.Body.Close()
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				scanner := bufio.NewScanner(resp.Body)
				Expect(scanner.Scan()).To(BeTrue())
				Expect(scanner.Text()).To(Equal("This is a v2 route"))
			}

			By("updating the EnvoyConfig to v3")
			{
				key := types.NamespacedName{Name: "nginx-envoyconfig", Namespace: testNamespace}
				patch := client.MergeFrom(ec.DeepCopy())
				ec.Spec = testutil.GenerateEnvoyConfig(key, nodeID, envoy.APIv3,
					func() map[string]envoy.Resource { return nil },
					func() map[string]envoy.Resource { return nil },
					func() map[string]envoy.Resource {
						k, v := testutil.DirectResponseRouteV3("direct_response", "This is a v3 route")
						return map[string]envoy.Resource{k: v}
					},
					func() map[string]envoy.Resource {
						k, v := testutil.HTTPListenerWithRdsV3("http", "direct_response", testutil.GetAddressV3("0.0.0.0", envoyListenerPort), nil)
						return map[string]envoy.Resource{k: v}
					},
					nil,
				).Spec
				err := k8sClient.Patch(context.Background(), ec, patch)
				Expect(err).ToNot(HaveOccurred())
			}

			By("validating the envoy sidecar still returns the v2 message (still using v2 config)")
			{
				var resp *http.Response
				var err error

				resp, err = http.Get(fmt.Sprintf("http://localhost:%v", localPort))
				Expect(err).ToNot(HaveOccurred())
				scanner := bufio.NewScanner(resp.Body)
				Expect(scanner.Scan()).To(BeTrue())
				Expect(scanner.Text()).To(Equal("This is a v2 route"))
			}

			By("updating the sidecar to v3")
			{
				key := types.NamespacedName{Name: "nginx", Namespace: testNamespace}
				dep := &appsv1.Deployment{}
				err := k8sClient.Get(context.Background(), key, dep)
				Expect(err).ToNot(HaveOccurred())

				dep.Spec = testutil.GenerateDeploymentWithInjection(key, nodeID, "v3", "v1.16.0", envoyListenerPort).Spec
				err = k8sClient.Update(context.Background(), dep)
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() bool {
					err := k8sClient.Get(context.Background(), key, dep)
					Expect(err).ToNot(HaveOccurred())
					if dep.Status.ObservedGeneration == 2 && dep.Status.Replicas == 1 && dep.Status.ReadyReplicas == 1 {
						return true
					}
					return false
				}, 60*time.Second, 5*time.Second).Should(BeTrue())
			}

			By("waiting until the old v2 ReplicaSet is completely drained")
			{
				Eventually(func() int {
					podList := &corev1.PodList{}
					selector := client.MatchingLabels{testutil.DeploymentLabelKey: testutil.DeploymentLabelValue}
					err := k8sClient.List(context.Background(), podList,
						[]client.ListOption{selector, client.InNamespace(testNamespace)}...)
					Expect(err).ToNot(HaveOccurred())
					pod = podList.Items[0]
					return len(podList.Items)
				}, 60*time.Second, 5*time.Second).Should(Equal(1))
			}

			By("reopening the port forward to the new pod")
			{
				close(stopCh)
				stopCh = make(chan struct{})
				readyCh = make(chan struct{})
				defer close(stopCh)
				go func() {
					defer GinkgoRecover()
					fw, err := testutil.NewTestPortForwarder(cfg, pod, uint32(localPort), envoyListenerPort, GinkgoWriter, stopCh, readyCh)
					Expect(err).ToNot(HaveOccurred())
					err = fw.ForwardPorts()
					Expect(err).ToNot(HaveOccurred())
				}()

				ticker := time.NewTimer(10 * time.Second)
				select {
				case <-ticker.C:
					Fail("timed out while waiting for port forward")
				case <-readyCh:
					ticker.Stop()
					break
				}
			}

			By("validating that the envoy container returns a direct response with the v3 message")
			{
				var resp *http.Response
				var err error

				Eventually(func() error {
					resp, err = http.Get(fmt.Sprintf("http://localhost:%v", localPort))
					return err
				}, 60*time.Second, 5*time.Second).ShouldNot(HaveOccurred())

				defer resp.Body.Close()
				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				scanner := bufio.NewScanner(resp.Body)
				Expect(scanner.Scan()).To(BeTrue())
				Expect(scanner.Text()).To(Equal("This is a v3 route"))
			}
		})
	})
})
