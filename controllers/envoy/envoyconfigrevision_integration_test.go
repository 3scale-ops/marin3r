package controllers

import (
	"context"
	"time"

	envoyv1alpha1 "github.com/3scale/marin3r/apis/envoy/v1alpha1"
	xdss "github.com/3scale/marin3r/pkg/discoveryservice/xdss"
	xdss_v2 "github.com/3scale/marin3r/pkg/discoveryservice/xdss/v2"
	xdss_v3 "github.com/3scale/marin3r/pkg/discoveryservice/xdss/v3"
	"github.com/3scale/marin3r/pkg/envoy"
	testutil "github.com/3scale/marin3r/pkg/util/test"
	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_config_endpoint_v3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	envoy_extensions_transport_sockets_tls_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	cache_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cache_v2 "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	cache_v3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/operator-lib/status"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("EnvoyConfigRevision controller", func() {
	var namespace string

	BeforeEach(func() {
		// Create a namespace for each block
		namespace = "test-namespace-" + uuid.New().String()
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
		// Clear the xDS caches after each test
		ecrV2Reconciler.XdsCache = xdss_v2.NewCache(cache_v2.NewSnapshotCache(true, cache_v2.IDHash{}, nil))
		ecrV3Reconciler.XdsCache = xdss_v3.NewCache(cache_v3.NewSnapshotCache(true, cache_v3.IDHash{}, nil))

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

	Context("Using v2 envoy API version", func() {
		var ecr *envoyv1alpha1.EnvoyConfigRevision

		BeforeEach(func() {
			// Create a v2 EnvoyConfigRevision for each block
			ecr = &envoyv1alpha1.EnvoyConfigRevision{
				ObjectMeta: metav1.ObjectMeta{Name: "ecr", Namespace: namespace},
				Spec: envoyv1alpha1.EnvoyConfigRevisionSpec{
					NodeID:  "node",
					Version: "xxxx",
					EnvoyResources: &envoyv1alpha1.EnvoyResources{
						Endpoints: []envoyv1alpha1.EnvoyResource{
							{Name: "endpoint", Value: "{\"cluster_name\": \"endpoint\"}"},
						}}},
			}
			err := k8sClient.Create(context.Background(), ecr)
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "ecr", Namespace: namespace}, ecr)
				if err != nil {
					return false
				}
				return true
			}, 30*time.Second, 5*time.Second).Should(BeTrue())
		})

		When("RevisionPublished condition is false in EnvoyConfigRevision", func() {

			It("Should not make changes to the xDS cache", func() {

				_, err := ecrV2Reconciler.XdsCache.GetSnapshot(ecr.Spec.NodeID)
				Expect(err).To(HaveOccurred())
			})
		})

		When("RevisionPublished condition is true in EnvoyConfigRevision", func() {

			It("Should update the xDS cache with new snapshot for the nodeID and do not modify the v3 xDS cache", func() {

				// Set ECR RevisionPublished condition to true
				ecr = &envoyv1alpha1.EnvoyConfigRevision{}
				Eventually(func() bool {
					err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "ecr", Namespace: namespace}, ecr)
					if err != nil {
						return false
					}
					return true
				}, 30*time.Second, 5*time.Second).Should(BeTrue())

				patch := client.MergeFrom(ecr.DeepCopy())
				ecr.Status.Conditions.SetCondition(status.Condition{
					Type:   envoyv1alpha1.RevisionPublishedCondition,
					Status: corev1.ConditionTrue,
				})

				err := k8sClient.Status().Patch(context.Background(), ecr, patch)
				Expect(err).ToNot(HaveOccurred())
				Expect(ecr.Status.Conditions.IsTrueFor(envoyv1alpha1.RevisionPublishedCondition)).To(BeTrue())

				// A snapshot for the spec.nodeID should exist in the xDS v2 cache
				var gotV2Snap xdss.Snapshot
				Eventually(func() bool {
					gotV2Snap, err = ecrV2Reconciler.XdsCache.GetSnapshot(ecr.Spec.NodeID)
					if err != nil {
						return false
					}
					return true
				}, 30*time.Second, 5*time.Second).Should(BeTrue())

				wantSnap := xdss_v2.NewSnapshot(&cache_v2.Snapshot{
					Resources: [6]cache_v2.Resources{
						{Version: "xxxx", Items: map[string]cache_types.Resource{
							"endpoint": &envoy_api_v2.ClusterLoadAssignment{ClusterName: "endpoint"}}},
						{Version: "xxxx", Items: map[string]cache_types.Resource{}},
						{Version: "xxxx", Items: map[string]cache_types.Resource{}},
						{Version: "xxxx", Items: map[string]cache_types.Resource{}},
						{Version: "xxxx-557db659d4", Items: map[string]cache_types.Resource{}},
						{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					}})
				Expect(testutil.SnapshotsAreEqual(gotV2Snap, wantSnap)).To(BeTrue())

				// v3 xDS cache should not have an snapshot for spec.nodeID
				_, err = ecrV3Reconciler.XdsCache.GetSnapshot(ecr.Spec.NodeID)
				Expect(err).To(HaveOccurred())

			})

		})
	})

	Context("Using v3 envoy API version", func() {
		var ecr *envoyv1alpha1.EnvoyConfigRevision

		BeforeEach(func() {
			// Create a v3 EnvoyConfigRevision for each block
			ecr = &envoyv1alpha1.EnvoyConfigRevision{
				ObjectMeta: metav1.ObjectMeta{Name: "ecr", Namespace: namespace},
				Spec: envoyv1alpha1.EnvoyConfigRevisionSpec{
					NodeID:   "node",
					Version:  "xxxx",
					EnvoyAPI: pointer.StringPtr(string(envoy.APIv3)),
					EnvoyResources: &envoyv1alpha1.EnvoyResources{
						Endpoints: []envoyv1alpha1.EnvoyResource{
							{Name: "endpoint", Value: "{\"cluster_name\": \"endpoint\"}"},
						}}},
			}
			err := k8sClient.Create(context.Background(), ecr)
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "ecr", Namespace: namespace}, ecr)
				if err != nil {
					return false
				}
				return true
			}, 30*time.Second, 5*time.Second).Should(BeTrue())
		})

		When("RevisionPublished condition is false in EnvoyConfigRevision", func() {

			It("Should not make changes to the xDS cache", func() {

				_, err := ecrV3Reconciler.XdsCache.GetSnapshot(ecr.Spec.NodeID)
				Expect(err).To(HaveOccurred())
			})
		})

		When("RevisionPublished condition is true in EnvoyConfigRevision", func() {

			It("Should update the xDS cache with new snapshot for the nodeID and do not modify the v3 xDS cache", func() {

				// Set ECR RevisionPublished condition to true
				ecr = &envoyv1alpha1.EnvoyConfigRevision{}
				Eventually(func() bool {
					err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "ecr", Namespace: namespace}, ecr)
					if err != nil {
						return false
					}
					return true
				}, 30*time.Second, 5*time.Second).Should(BeTrue())

				patch := client.MergeFrom(ecr.DeepCopy())
				ecr.Status.Conditions.SetCondition(status.Condition{
					Type:   envoyv1alpha1.RevisionPublishedCondition,
					Status: corev1.ConditionTrue,
				})

				err := k8sClient.Status().Patch(context.Background(), ecr, patch)
				Expect(err).ToNot(HaveOccurred())
				Expect(ecr.Status.Conditions.IsTrueFor(envoyv1alpha1.RevisionPublishedCondition)).To(BeTrue())

				// A snapshot for the spec.nodeID should exist in the xDS v2 cache
				var gotV3Snap xdss.Snapshot
				Eventually(func() bool {
					gotV3Snap, err = ecrV3Reconciler.XdsCache.GetSnapshot(ecr.Spec.NodeID)
					if err != nil {
						return false
					}
					return true
				}, 30*time.Second, 5*time.Second).Should(BeTrue())

				wantSnap := xdss_v3.NewSnapshot(&cache_v3.Snapshot{
					Resources: [6]cache_v3.Resources{
						{Version: "xxxx", Items: map[string]cache_types.Resource{
							"endpoint": &envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint"}}},
						{Version: "xxxx", Items: map[string]cache_types.Resource{}},
						{Version: "xxxx", Items: map[string]cache_types.Resource{}},
						{Version: "xxxx", Items: map[string]cache_types.Resource{}},
						{Version: "xxxx-557db659d4", Items: map[string]cache_types.Resource{}},
						{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					}})
				Expect(testutil.SnapshotsAreEqual(gotV3Snap, wantSnap)).To(BeTrue())

				// v2 xDS cache should not have an snapshot for spec.nodeID
				_, err = ecrV2Reconciler.XdsCache.GetSnapshot(ecr.Spec.NodeID)
				Expect(err).To(HaveOccurred())

			})

		})
	})

	Context("Load certificates from secrets", func() {
		var ecr *envoyv1alpha1.EnvoyConfigRevision

		BeforeEach(func() {
			// Create a secret
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "secret", Namespace: namespace},
				Type:       corev1.SecretTypeTLS,
				Data:       map[string][]byte{"tls.crt": []byte("cert"), "tls.key": []byte("key")},
			}
			err := k8sClient.Create(context.Background(), secret)
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "secret", Namespace: namespace}, secret)
				if err != nil {
					return false
				}
				return true
			}, 30*time.Second, 5*time.Second).Should(BeTrue())

			// Create a v3 EnvoyConfigRevision and publish it for each block
			ecr = &envoyv1alpha1.EnvoyConfigRevision{
				ObjectMeta: metav1.ObjectMeta{Name: "ecr", Namespace: namespace},
				Spec: envoyv1alpha1.EnvoyConfigRevisionSpec{
					NodeID:   "node",
					Version:  "xxxx",
					EnvoyAPI: pointer.StringPtr(string(envoy.APIv3)),
					EnvoyResources: &envoyv1alpha1.EnvoyResources{
						Secrets: []envoyv1alpha1.EnvoySecretResource{{
							Name: "secret",
							Ref:  corev1.SecretReference{Name: "secret", Namespace: namespace}},
						},
					},
				},
			}
			err = k8sClient.Create(context.Background(), ecr)
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "ecr", Namespace: namespace}, ecr)
				if err != nil {
					return false
				}
				return true
			}, 30*time.Second, 5*time.Second).Should(BeTrue())

			// Set the ecr as published
			patch := client.MergeFrom(ecr.DeepCopy())
			ecr.Status.Conditions.SetCondition(status.Condition{
				Type:   envoyv1alpha1.RevisionPublishedCondition,
				Status: corev1.ConditionTrue,
			})
			err = k8sClient.Status().Patch(context.Background(), ecr, patch)
			Expect(err).ToNot(HaveOccurred())
			Expect(ecr.Status.Conditions.IsTrueFor(envoyv1alpha1.RevisionPublishedCondition)).To(BeTrue())

			wantSnap := xdss_v3.NewSnapshot(&cache_v3.Snapshot{
				Resources: [6]cache_v3.Resources{
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx-77c9875d7b", Items: map[string]cache_types.Resource{
						"secret": &envoy_extensions_transport_sockets_tls_v3.Secret{
							Name: "secret",
							Type: &envoy_extensions_transport_sockets_tls_v3.Secret_TlsCertificate{
								TlsCertificate: &envoy_extensions_transport_sockets_tls_v3.TlsCertificate{
									PrivateKey: &envoy_config_core_v3.DataSource{
										Specifier: &envoy_config_core_v3.DataSource_InlineBytes{InlineBytes: []byte("key")},
									},
									CertificateChain: &envoy_config_core_v3.DataSource{
										Specifier: &envoy_config_core_v3.DataSource_InlineBytes{InlineBytes: []byte("cert")},
									}}}}}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
				}})

			// Wait for the revision to get written to the xDS cache
			Eventually(func() bool {
				gotV3Snap, err := ecrV3Reconciler.XdsCache.GetSnapshot(ecr.Spec.NodeID)
				if err != nil {
					return false
				}
				return testutil.SnapshotsAreEqual(gotV3Snap, wantSnap)
			}, 30*time.Second, 5*time.Second).Should(BeTrue())
		})

		When("Secret changes", func() {

			It("Should update the xDS cache with new snapshot for the nodeID", func() {
				// Update the certificate
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: "secret", Namespace: namespace},
					Type:       corev1.SecretTypeTLS,
					Data:       map[string][]byte{"tls.crt": []byte("new-cert"), "tls.key": []byte("new-key")},
				}
				err := k8sClient.Update(context.Background(), secret)
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() bool {
					err = k8sClient.Get(context.Background(), types.NamespacedName{Name: "secret", Namespace: namespace}, secret)
					if string(secret.Data["tls.crt"]) == "new-cert" {
						return true
					}
					return false
				}, 30*time.Second, 5*time.Second).Should(BeTrue())

				wantSnap := xdss_v3.NewSnapshot(&cache_v3.Snapshot{
					Resources: [6]cache_v3.Resources{
						{Version: "xxxx", Items: map[string]cache_types.Resource{}},
						{Version: "xxxx", Items: map[string]cache_types.Resource{}},
						{Version: "xxxx", Items: map[string]cache_types.Resource{}},
						{Version: "xxxx", Items: map[string]cache_types.Resource{}},
						{Version: "xxxx-679f7cbbfd", Items: map[string]cache_types.Resource{
							"secret": &envoy_extensions_transport_sockets_tls_v3.Secret{
								Name: "secret",
								Type: &envoy_extensions_transport_sockets_tls_v3.Secret_TlsCertificate{
									TlsCertificate: &envoy_extensions_transport_sockets_tls_v3.TlsCertificate{
										PrivateKey: &envoy_config_core_v3.DataSource{
											Specifier: &envoy_config_core_v3.DataSource_InlineBytes{InlineBytes: []byte("new-key")},
										},
										CertificateChain: &envoy_config_core_v3.DataSource{
											Specifier: &envoy_config_core_v3.DataSource_InlineBytes{InlineBytes: []byte("new-cert")},
										}}}}}},
						{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					}})

				// Wait for the revision to get written to the xDS cache
				Eventually(func() bool {
					gotV3Snap, err := ecrV3Reconciler.XdsCache.GetSnapshot(ecr.Spec.NodeID)
					if err != nil {
						return false
					}
					return testutil.SnapshotsAreEqual(gotV3Snap, wantSnap)
				}, 30*time.Second, 5*time.Second).Should(BeTrue())

			})

		})
	})

})
