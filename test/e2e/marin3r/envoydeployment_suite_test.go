package e2e

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"time"

	marin3rv1alpha1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	operatorv1alpha1 "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	"github.com/3scale-ops/marin3r/pkg/envoy"
	"github.com/3scale-ops/marin3r/pkg/envoy/container/defaults"
	"github.com/3scale-ops/marin3r/pkg/util/pointer"
	testutil "github.com/3scale-ops/marin3r/test/e2e/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/phayes/freeport"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("EnvoyDeployment", func() {
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
			return err == nil
		}, timeout, poll).Should(BeTrue())

		By("creating a DiscoveryService instance")
		ds = &operatorv1alpha1.DiscoveryService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "instance",
				Namespace: testNamespace,
			},
			Spec: operatorv1alpha1.DiscoveryServiceSpec{
				Image: pointer.New(image),
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
		}, timeout, poll).Should(Equal(1))
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

	Context("Using v3 API", func() {
		var localPort int
		var nodeID string
		var ec *marin3rv1alpha1.EnvoyConfig

		BeforeEach(func() {
			var err error
			localPort, err = freeport.GetFreePort()
			Expect(err).ToNot(HaveOccurred())
			nodeID = nameGenerator.Generate()
		})

		It("deploys Envoy as an EnvoyDeployment", func() {

			By("applying an EnvoyConfig that will configure the envoy Deployment through service discovery")
			key := types.NamespacedName{Name: "envoyconfig", Namespace: testNamespace}
			ec = testutil.GenerateEnvoyConfig(key, nodeID, envoy.APIv3,
				nil,
				[]envoy.Resource{testutil.ClusterWithEdsV3("nginx")},
				[]envoy.Resource{testutil.ProxyPassRouteV3("proxypass", "nginx")},
				[]envoy.Resource{testutil.HTTPListener("http", "proxypass", "router_filter", testutil.GetAddressV3("0.0.0.0", envoyListenerPort), nil)},
				[]envoy.Resource{testutil.HTTPFilterRouter("router_filter")},
				nil,
				[]testutil.EndpointDiscovery{{ClusterName: "nginx", PortName: "http", LabelKey: "kubernetes.io/service-name", LabelValue: "nginx"}},
			)
			Eventually(func() error {
				return k8sClient.Create(context.Background(), ec)
			}, timeout, poll).ShouldNot(HaveOccurred())

			By("creating an nginx Deployment")
			key = types.NamespacedName{Name: "nginx", Namespace: testNamespace}
			dep := testutil.GenerateDeployment(key)
			err := k8sClient.Create(context.Background(), dep)
			Expect(err).ToNot(HaveOccurred())

			selector := client.MatchingLabels{testutil.DeploymentLabelKey: testutil.DeploymentLabelValue}
			Eventually(func() int {
				return testutil.ReadyReplicas(k8sClient, testNamespace, selector)
			}, timeout, poll).Should(Equal(1))

			nginxPodList := &corev1.PodList{}
			err = k8sClient.List(context.Background(), nginxPodList,
				[]client.ListOption{selector, client.InNamespace(testNamespace)}...)
			Expect(err).ToNot(HaveOccurred())

			By("creating a headless service pointing to the nginx Deployment")
			key = types.NamespacedName{Name: "nginx", Namespace: testNamespace}
			svc := testutil.GenerateHeadlessService(key)
			err = k8sClient.Create(context.Background(), svc)
			Expect(err).ToNot(HaveOccurred())

			By("creating an EnvoyDeployment")
			key = types.NamespacedName{Name: "envoy", Namespace: testNamespace}
			edep := &operatorv1alpha1.EnvoyDeployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: operatorv1alpha1.EnvoyDeploymentSpec{
					DiscoveryServiceRef: ds.GetName(),
					EnvoyConfigRef:      ec.GetName(),
					Image:               pointer.New(defaults.ImageRepo + ":" + envoyVersionV3),
					InitManager:         &operatorv1alpha1.InitManager{Image: pointer.New(image)},
				},
			}

			Eventually(func() error {
				return k8sClient.Create(context.Background(), edep)
			}, timeout, poll).ShouldNot(HaveOccurred())

			selector = client.MatchingLabels{
				"app.kubernetes.io/name":       "marin3r",
				"app.kubernetes.io/managed-by": "marin3r-operator",
				"app.kubernetes.io/component":  "envoy-deployment",
				"app.kubernetes.io/instance":   "envoy",
			}
			Eventually(func() int {
				return testutil.ReadyReplicas(k8sClient, testNamespace, selector)
			}, timeout, poll).Should(Equal(1))

			By("getting the Envoy Pod")
			podList := &corev1.PodList{}
			Eventually(func() int {
				err = k8sClient.List(context.Background(), podList,
					[]client.ListOption{selector, client.InNamespace(testNamespace)}...)
				Expect(err).ToNot(HaveOccurred())
				return len(podList.Items)
			}, timeout, poll).Should(Equal(1))

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

			By("doing a request against envoy Deployment, that should be forwarded to the nginx Pod")
			var resp *http.Response
			Eventually(func() error {
				resp, err = http.Get(fmt.Sprintf("http://localhost:%v/test", localPort))
				return err
			}, timeout, poll).ShouldNot(HaveOccurred())

			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			scanner := bufio.NewScanner(resp.Body)
			Expect(scanner.Scan()).To(BeTrue())
			Expect(scanner.Scan()).To(BeTrue())
			Expect(scanner.Text()).To(Equal(fmt.Sprintf("Server name: %s", nginxPodList.Items[0].GetName())))
		})
	})
})
