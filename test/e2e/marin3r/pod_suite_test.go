package e2e

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"time"

	"github.com/3scale/marin3r/apis/marin3r/v1alpha1"
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

var _ = Describe("Envoy pods", func() {
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
		}, 600*time.Second, 5*time.Second).Should(Equal(1))

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

	Context("Envoy Pod with xDS configured", func() {
		var t testutil.TestPod
		var ec *v1alpha1.EnvoyConfig
		var localPort int
		var nodeID string
		var stopCh chan struct{}
		var readyCh chan struct{}

		BeforeEach(func() {
			var err error
			localPort, err = freeport.GetFreePort()
			Expect(err).ToNot(HaveOccurred())

			nodeID = nameGenerator.Generate()

			By("applying an EnvoyConfig that configures the Pod with a direct response")
			key := types.NamespacedName{Name: "test-envoyconfig", Namespace: testNamespace}
			ec = testutil.GenerateEnvoyConfig(key, nodeID, envoy.APIv3,
				func() map[string]envoy.Resource { return nil },
				func() map[string]envoy.Resource { return nil },
				func() map[string]envoy.Resource {
					k, v := testutil.DirectResponseRouteV3("direct_response", "OK")
					return map[string]envoy.Resource{k: v}
				},
				func() map[string]envoy.Resource {
					k, v := testutil.HTTPListenerWithRdsV3("http", "direct_response", testutil.GetAddressV3("0.0.0.0", envoyListenerPort), nil)
					return map[string]envoy.Resource{k: v}
				},
				nil,
			)
			err = k8sClient.Create(context.Background(), ec)
			Expect(err).ToNot(HaveOccurred())

			By("deploying a Pod that will consume the EnvoyConfig through xDS")
			key = types.NamespacedName{Name: "test-pod", Namespace: testNamespace}
			t = testutil.GeneratePodWithBootstrap(key, nodeID, "v3", "v1.16.0", "instance")

			err = k8sClient.Create(context.Background(), t.EnvoyBootstrap)
			Expect(err).ToNot(HaveOccurred())

			err = k8sClient.Create(context.Background(), t.Pod)
			Expect(err).ToNot(HaveOccurred())

			selector := client.MatchingLabels{testutil.PodLabelKey: testutil.PodLabelValue}
			Eventually(func() int {
				return testutil.ReadyReplicas(k8sClient, testNamespace, selector)
			}, 60*time.Second, 5*time.Second).Should(Equal(1))

			By(fmt.Sprintf("forwarding the Pod's port to localhost: %v", localPort))
			stopCh = make(chan struct{})
			readyCh = make(chan struct{})
			logger.Info(fmt.Sprintf("%v", cfg))
			go func() {
				defer GinkgoRecover()
				fw, err := testutil.NewTestPortForwarder(cfg, *t.Pod, uint32(localPort), envoyListenerPort, GinkgoWriter, stopCh, readyCh)
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
		})

		AfterEach(func() {
			close(stopCh)
			err := k8sClient.Delete(context.Background(), ec)
			Expect(err).ToNot(HaveOccurred())
		})

		It("will reply with 200 ok to any request", func() {
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
			Expect(scanner.Text()).To(Equal("OK"))
		})

		It("it will rollback the config on an Envoy NACK", func() {

			By("updating the envoy resources with a listener that will fail to update")
			key := types.NamespacedName{Name: "test-envoyconfig", Namespace: testNamespace}
			patch := client.MergeFrom(ec.DeepCopy())
			ec.Spec = testutil.GenerateEnvoyConfig(key, nodeID, envoy.APIv3,
				func() map[string]envoy.Resource { return nil },
				func() map[string]envoy.Resource { return nil },
				func() map[string]envoy.Resource {
					k, v := testutil.DirectResponseRouteV3("direct_response", "OK")
					return map[string]envoy.Resource{k: v}
				},
				func() map[string]envoy.Resource {
					// Envoy listeners don't allow bind address changes
					k, v := testutil.HTTPListenerWithRdsV3("http", "direct_response", testutil.GetAddressV3("0.0.0.0", 30333), nil)
					return map[string]envoy.Resource{k: v}
				},
				nil,
			).Spec
			err := k8sClient.Patch(context.Background(), ec, patch)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() bool {
				err = k8sClient.Get(context.Background(), key, ec)
				return ec.Status.CacheState == marin3rv1alpha1.RollbackState
			}, 60*time.Second, 5*time.Second).ShouldNot(BeTrue())

			By("validating the envoy Pod still replis anything with 200 OK")
			var resp *http.Response
			Eventually(func() error {
				resp, err = http.Get(fmt.Sprintf("http://localhost:%v", localPort))
				return err
			}, 60*time.Second, 5*time.Second).ShouldNot(HaveOccurred())

			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			scanner := bufio.NewScanner(resp.Body)
			Expect(scanner.Scan()).To(BeTrue())
			Expect(scanner.Text()).To(Equal("OK"))

		})

		Context("using certificates from Secrets", func() {

			It("the xDS server feeds certificates from Secrets", func() {
				var secret *corev1.Secret

				By("creating a self-signed certificate with very low validity")
				{
					var err error
					key := types.NamespacedName{Name: "self-signed-cert", Namespace: testNamespace}
					secret, err = testutil.GenerateTLSSecret(key, "localhost", "10m")
					Expect(err).ToNot(HaveOccurred())

					err = k8sClient.Create(context.Background(), secret)
					Expect(err).ToNot(HaveOccurred())
				}

				By("updating the envoy resources with an https listener")
				{
					key := types.NamespacedName{Name: "test-envoyconfig", Namespace: testNamespace}
					patch := client.MergeFrom(ec.DeepCopy())
					ec.Spec = testutil.GenerateEnvoyConfig(key, nodeID, envoy.APIv3,
						func() map[string]envoy.Resource { return nil },
						func() map[string]envoy.Resource { return nil },
						func() map[string]envoy.Resource {
							k, v := testutil.DirectResponseRouteV3("direct_response", "OK")
							return map[string]envoy.Resource{k: v}
						},
						func() map[string]envoy.Resource {
							k, v := testutil.HTTPListenerWithRdsV3("https", "direct_response", testutil.GetAddressV3("0.0.0.0", envoyListenerPort), testutil.TransportSocketV3("localhost"))
							return map[string]envoy.Resource{k: v}
						},
						map[string]string{"localhost": "self-signed-cert"},
					).Spec
					err := k8sClient.Patch(context.Background(), ec, patch)
					Expect(err).ToNot(HaveOccurred())
				}

				By("validating the envoy Pod still replies anything with 200 OK, using the provided certificate")
				{
					var resp *http.Response
					var err error

					tlsClient := &http.Client{
						Transport: &http.Transport{
							TLSClientConfig: &tls.Config{
								RootCAs: func() *x509.CertPool {
									roots := x509.NewCertPool()
									Expect(roots.AppendCertsFromPEM(secret.Data["tls.crt"])).To(BeTrue())
									return roots
								}(),
							}}}
					Eventually(func() error {
						resp, err = tlsClient.Get(fmt.Sprintf("https://localhost:%v", localPort))
						return err
					}, 60*time.Second, 5*time.Second).ShouldNot(HaveOccurred())

					defer resp.Body.Close()
					Expect(resp.StatusCode).To(Equal(http.StatusOK))

					scanner := bufio.NewScanner(resp.Body)
					Expect(scanner.Scan()).To(BeTrue())
					Expect(scanner.Text()).To(Equal("OK"))
					Expect(resp.TLS.VerifiedChains[0][0].Subject.CommonName).To(Equal("localhost"))
				}

				By("updating the Secret with a new certificate")
				{
					patch := client.MergeFrom(secret.DeepCopy())
					secret.Data = func() map[string][]byte {
						key := types.NamespacedName{Name: "self-signed-cert"}
						secret, err := testutil.GenerateTLSSecret(key, "127.0.0.1", "10m")
						Expect(err).ToNot(HaveOccurred())
						return secret.Data
					}()
					err := k8sClient.Patch(context.Background(), secret, patch)
					Expect(err).ToNot(HaveOccurred())
				}

				By("validating the envoy Pod still replis anything with 200 OK, but the certificate common name is different")
				{
					var resp *http.Response
					var err error

					tlsClient := &http.Client{
						Transport: &http.Transport{
							TLSClientConfig: &tls.Config{
								RootCAs: func() *x509.CertPool {
									roots := x509.NewCertPool()
									Expect(roots.AppendCertsFromPEM(secret.Data["tls.crt"])).To(BeTrue())
									return roots
								}(),
							}}}
					Eventually(func() string {
						resp, err = tlsClient.Get(fmt.Sprintf("https://127.0.0.1:%v", localPort))
						if err != nil {
							return ""
						}
						return resp.TLS.VerifiedChains[0][0].Subject.CommonName
					}, 60*time.Second, 5*time.Second).Should(Equal("127.0.0.1"))
				}
			})

		})
	})
})
