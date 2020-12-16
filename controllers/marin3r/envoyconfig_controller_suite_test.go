package controllers

import (
	"context"
	"fmt"
	"strings"
	"time"

	marin3rv1alpha1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
	"github.com/3scale/marin3r/pkg/common"
	xdss_v2 "github.com/3scale/marin3r/pkg/discoveryservice/xdss/v2"
	xdss_v3 "github.com/3scale/marin3r/pkg/discoveryservice/xdss/v3"
	"github.com/3scale/marin3r/pkg/envoy"
	"github.com/3scale/marin3r/pkg/reconcilers/marin3r/envoyconfig/filters"
	"github.com/3scale/marin3r/pkg/reconcilers/marin3r/envoyconfig/revisions"
	rollback "github.com/3scale/marin3r/pkg/reconcilers/marin3r/envoyconfig/rollback"
	testutil "github.com/3scale/marin3r/pkg/util/test"
	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_config_endpoint_v3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	cache_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cache_v2 "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	cache_v3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/operator-lib/status"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var s *runtime.Scheme = scheme.Scheme

func init() {
	s.AddKnownTypes(marin3rv1alpha1.GroupVersion,
		&marin3rv1alpha1.EnvoyConfigRevision{},
		&marin3rv1alpha1.EnvoyConfigRevisionList{},
		&marin3rv1alpha1.EnvoyConfig{},
	)
}

var _ = Describe("EnvoyConfig controller", func() {
	var namespace string
	var nodeID string

	BeforeEach(func() {
		// Create a namespace for each block
		namespace = "test-ns-" + nameGenerator.Generate()
		// Create a nodeID for each block
		nodeID = nameGenerator.Generate()
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

	Context("using v2 envoy API version", func() {
		var ec *marin3rv1alpha1.EnvoyConfig

		BeforeEach(func() {
			By("creating a v2 EnvoyConfig")
			ec = &marin3rv1alpha1.EnvoyConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "ec", Namespace: namespace},
				Spec: marin3rv1alpha1.EnvoyConfigSpec{
					EnvoyAPI: pointer.StringPtr("v2"),
					NodeID:   nodeID,
					EnvoyResources: &marin3rv1alpha1.EnvoyResources{
						Endpoints: []marin3rv1alpha1.EnvoyResource{
							{Name: "endpoint", Value: "{\"cluster_name\": \"endpoint\"}"},
						}}},
			}
			err := k8sClient.Create(context.Background(), ec)
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "ec", Namespace: namespace}, ec)
				if err != nil {
					return false
				}
				return true
			}, 30*time.Second, 5*time.Second).Should(BeTrue())
		})

		When("EnvoyConfig is created", func() {

			It("should create a matching EnvoyConfigRevision and resources should be in the xDS cache", func() {

				Eventually(func() bool {
					err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "ec", Namespace: namespace}, ec)
					Expect(err).ToNot(HaveOccurred())
					if ec.Status.PublishedVersion == "" {
						return false
					}
					return true
				}, 30*time.Second, 5*time.Second).Should(BeTrue())

				// Get the EnvoyConfigRevision that should have been created
				ecr := &marin3rv1alpha1.EnvoyConfigRevision{}
				ecrKey := types.NamespacedName{Name: ec.Status.ConfigRevisions[len(ec.Status.ConfigRevisions)-1].Ref.Name, Namespace: namespace}
				err := k8sClient.Get(context.Background(), ecrKey, ecr)
				Expect(err).ToNot(HaveOccurred())

				// Validate the cache for the nodeID
				wantRevision := common.Hash(ec.Spec.EnvoyResources)
				wantSnap := xdss_v2.NewSnapshot(&cache_v2.Snapshot{
					Resources: [6]cache_v2.Resources{
						{Version: wantRevision, Items: map[string]cache_types.Resource{
							"endpoint": &envoy_api_v2.ClusterLoadAssignment{ClusterName: "endpoint"}}},
						{Version: wantRevision, Items: map[string]cache_types.Resource{}},
						{Version: wantRevision, Items: map[string]cache_types.Resource{}},
						{Version: wantRevision, Items: map[string]cache_types.Resource{}},
						{Version: wantRevision + "-557db659d4", Items: map[string]cache_types.Resource{}},
						{Version: wantRevision, Items: map[string]cache_types.Resource{}},
					}})

				// Wait for the revision to get written to the xDS cache
				Eventually(func() bool {
					gotV2Snap, err := ecrV2Reconciler.XdsCache.GetSnapshot(ec.Spec.NodeID)
					if err != nil {
						return false
					}
					return testutil.SnapshotsAreEqual(gotV2Snap, wantSnap)
				}, 30*time.Second, 5*time.Second).Should(BeTrue())

				err = k8sClient.Get(context.Background(), types.NamespacedName{Name: "ec", Namespace: namespace}, ec)
				Expect(err).ToNot(HaveOccurred())
				Expect(ec.Status.PublishedVersion).To(Equal(wantRevision))
				Expect(ec.Status.DesiredVersion).To(Equal(wantRevision))
				Expect(len(ec.Status.ConfigRevisions)).To(Equal(1))
				Expect(ec.Status.ConfigRevisions[0].Ref.Name).To(Equal(fmt.Sprintf("%s-%s", ec.Spec.NodeID, wantRevision)))
			})
		})

		When("EnvoyConfig is updated", func() {
			var wantRevision string

			BeforeEach(func() {
				// Wait for current EnvoyConfig resources to get published in xDS
				Eventually(func() bool {
					err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "ec", Namespace: namespace}, ec)
					Expect(err).ToNot(HaveOccurred())
					if ec.Status.PublishedVersion == "" {
						return false
					}
					return true
				}, 30*time.Second, 5*time.Second).Should(BeTrue())

				// Update the resources in the spec
				ec.Spec.EnvoyResources = &marin3rv1alpha1.EnvoyResources{
					Clusters: []marin3rv1alpha1.EnvoyResource{
						{Name: "cluster", Value: "{\"name\": \"cluster\"}"},
					}}
				err := k8sClient.Update(context.Background(), ec)
				Expect(err).ToNot(HaveOccurred())

				// Wait for the new revision to get published
				wantRevision = common.Hash(ec.Spec.EnvoyResources)
				Eventually(func() bool {
					err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "ec", Namespace: namespace}, ec)
					Expect(err).ToNot(HaveOccurred())
					if ec.Status.PublishedVersion != wantRevision {
						return false
					}
					return true
				}, 30*time.Second, 5*time.Second).Should(BeTrue())
			})

			It("should create a new matching EnvoyConfigRevision and new resources should be in the xDS cache", func() {

				Expect(ec.Status.PublishedVersion).To(Equal(wantRevision))
				Expect(ec.Status.DesiredVersion).To(Equal(wantRevision))
				Expect(len(ec.Status.ConfigRevisions)).To(Equal(2))
				Expect(ec.Status.ConfigRevisions[len(ec.Status.ConfigRevisions)-1].Ref.Name).To(Equal(fmt.Sprintf("%s-%s", ec.Spec.NodeID, wantRevision)))

				// Get the EnvoyConfigRevision that should have been created
				ecr := &marin3rv1alpha1.EnvoyConfigRevision{}
				ecrKey := types.NamespacedName{Name: ec.Status.ConfigRevisions[len(ec.Status.ConfigRevisions)-1].Ref.Name, Namespace: namespace}
				err := k8sClient.Get(context.Background(), ecrKey, ecr)
				Expect(err).ToNot(HaveOccurred())

				// Validate the cache for the nodeID

				wantSnap := xdss_v2.NewSnapshot(&cache_v2.Snapshot{
					Resources: [6]cache_v2.Resources{
						{Version: wantRevision, Items: map[string]cache_types.Resource{}},
						{Version: wantRevision, Items: map[string]cache_types.Resource{
							"cluster": &envoy_api_v2.Cluster{Name: "cluster"}}},
						{Version: wantRevision, Items: map[string]cache_types.Resource{}},
						{Version: wantRevision, Items: map[string]cache_types.Resource{}},
						{Version: wantRevision + "-557db659d4", Items: map[string]cache_types.Resource{}},
						{Version: wantRevision, Items: map[string]cache_types.Resource{}},
					}})

				// Wait for the revision to get written to the xDS cache
				Eventually(func() bool {
					gotV2Snap, err := ecrV2Reconciler.XdsCache.GetSnapshot(ec.Spec.NodeID)
					if err != nil {
						return false
					}
					return testutil.SnapshotsAreEqual(gotV2Snap, wantSnap)
				}, 30*time.Second, 5*time.Second).Should(BeTrue())
			})

			It("should publish an already existent EnvoyConfigRevision if one already matches the current EnvoyConfig resources", func() {

				// Set the previous set of resources in the EnvoyConfig
				ec.Spec.EnvoyResources = &marin3rv1alpha1.EnvoyResources{
					Endpoints: []marin3rv1alpha1.EnvoyResource{
						{Name: "endpoint", Value: "{\"cluster_name\": \"endpoint\"}"},
					}}
				err := k8sClient.Update(context.Background(), ec)
				Expect(err).ToNot(HaveOccurred())

				// Wait for the existent revision to get published
				wantRevision = common.Hash(ec.Spec.EnvoyResources)
				Eventually(func() bool {
					err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "ec", Namespace: namespace}, ec)
					Expect(err).ToNot(HaveOccurred())
					if ec.Status.PublishedVersion != wantRevision {
						return false
					}
					return true
				}, 30*time.Second, 5*time.Second).Should(BeTrue())

				Expect(ec.Status.PublishedVersion).To(Equal(wantRevision))
				Expect(ec.Status.DesiredVersion).To(Equal(wantRevision))
				Expect(len(ec.Status.ConfigRevisions)).To(Equal(2))
				Expect(ec.Status.ConfigRevisions[len(ec.Status.ConfigRevisions)-1].Ref.Name).To(Equal(fmt.Sprintf("%s-%s", ec.Spec.NodeID, wantRevision)))

				// Get the EnvoyConfigRevision that should have been created
				ecr := &marin3rv1alpha1.EnvoyConfigRevision{}
				ecrKey := types.NamespacedName{Name: ec.Status.ConfigRevisions[len(ec.Status.ConfigRevisions)-1].Ref.Name, Namespace: namespace}
				err = k8sClient.Get(context.Background(), ecrKey, ecr)
				Expect(err).ToNot(HaveOccurred())

				// Validate the cache for the nodeID
				wantSnap := xdss_v2.NewSnapshot(&cache_v2.Snapshot{
					Resources: [6]cache_v2.Resources{
						{Version: wantRevision, Items: map[string]cache_types.Resource{
							"endpoint": &envoy_api_v2.ClusterLoadAssignment{ClusterName: "endpoint"}}},
						{Version: wantRevision, Items: map[string]cache_types.Resource{}},
						{Version: wantRevision, Items: map[string]cache_types.Resource{}},
						{Version: wantRevision, Items: map[string]cache_types.Resource{}},
						{Version: wantRevision + "-557db659d4", Items: map[string]cache_types.Resource{}},
						{Version: wantRevision, Items: map[string]cache_types.Resource{}},
					}})

				// Wait for the revision to get written to the xDS cache
				Eventually(func() bool {
					gotV2Snap, err := ecrV2Reconciler.XdsCache.GetSnapshot(ec.Spec.NodeID)
					if err != nil {
						return false
					}
					return testutil.SnapshotsAreEqual(gotV2Snap, wantSnap)
				}, 30*time.Second, 5*time.Second).Should(BeTrue())
			})
		})
	})

	Context("using v3 envoy API version", func() {
		var ec *marin3rv1alpha1.EnvoyConfig

		BeforeEach(func() {
			// Create a v3 EnvoyConfig for each block
			ec = &marin3rv1alpha1.EnvoyConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "ec", Namespace: namespace},
				Spec: marin3rv1alpha1.EnvoyConfigSpec{
					EnvoyAPI: pointer.StringPtr("v3"),
					NodeID:   nodeID,
					EnvoyResources: &marin3rv1alpha1.EnvoyResources{
						Endpoints: []marin3rv1alpha1.EnvoyResource{
							{Name: "endpoint", Value: "{\"cluster_name\": \"endpoint\"}"},
						}}},
			}
			err := k8sClient.Create(context.Background(), ec)
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "ec", Namespace: namespace}, ec)
				if err != nil {
					return false
				}
				return true
			}, 30*time.Second, 5*time.Second).Should(BeTrue())
		})

		When("EnvoyConfig is created", func() {

			It("should create a matching EnvoyConfigRevision and resources should be in the xDS cache", func() {

				Eventually(func() bool {
					err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "ec", Namespace: namespace}, ec)
					Expect(err).ToNot(HaveOccurred())
					if ec.Status.PublishedVersion == "" {
						return false
					}
					return true
				}, 30*time.Second, 5*time.Second).Should(BeTrue())

				// Get the EnvoyConfigRevision that should have been created
				ecr := &marin3rv1alpha1.EnvoyConfigRevision{}
				ecrKey := types.NamespacedName{Name: ec.Status.ConfigRevisions[len(ec.Status.ConfigRevisions)-1].Ref.Name, Namespace: namespace}
				err := k8sClient.Get(context.Background(), ecrKey, ecr)
				Expect(err).ToNot(HaveOccurred())

				// Validate the cache for the nodeID
				wantRevision := common.Hash(ec.Spec.EnvoyResources)
				wantSnap := xdss_v3.NewSnapshot(&cache_v3.Snapshot{
					Resources: [6]cache_v3.Resources{
						{Version: wantRevision, Items: map[string]cache_types.Resource{
							"endpoint": &envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint"}}},
						{Version: wantRevision, Items: map[string]cache_types.Resource{}},
						{Version: wantRevision, Items: map[string]cache_types.Resource{}},
						{Version: wantRevision, Items: map[string]cache_types.Resource{}},
						{Version: wantRevision + "-557db659d4", Items: map[string]cache_types.Resource{}},
						{Version: wantRevision, Items: map[string]cache_types.Resource{}},
					}})

				// Wait for the revision to get written to the xDS cache
				Eventually(func() bool {
					gotV2Snap, err := ecrV3Reconciler.XdsCache.GetSnapshot(ec.Spec.NodeID)
					if err != nil {
						return false
					}
					return testutil.SnapshotsAreEqual(gotV2Snap, wantSnap)
				}, 30*time.Second, 5*time.Second).Should(BeTrue())

				err = k8sClient.Get(context.Background(), types.NamespacedName{Name: "ec", Namespace: namespace}, ec)
				Expect(err).ToNot(HaveOccurred())
				Expect(ec.Status.PublishedVersion).To(Equal(wantRevision))
				Expect(ec.Status.DesiredVersion).To(Equal(wantRevision))
				Expect(len(ec.Status.ConfigRevisions)).To(Equal(1))
				Expect(ec.Status.ConfigRevisions[0].Ref.Name).To(Equal(fmt.Sprintf("%s-%s-%s", ec.Spec.NodeID, string(ec.GetEnvoyAPIVersion()), wantRevision)))
			})
		})
	})

	Context("self-healing", func() {
		var ec *marin3rv1alpha1.EnvoyConfig

		BeforeEach(func() {
			By("creating a v3 EnvoyConfig")
			ec = &marin3rv1alpha1.EnvoyConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "ec", Namespace: namespace},
				Spec: marin3rv1alpha1.EnvoyConfigSpec{
					EnvoyAPI: pointer.StringPtr("v3"),
					NodeID:   nodeID,
					EnvoyResources: &marin3rv1alpha1.EnvoyResources{
						Endpoints: []marin3rv1alpha1.EnvoyResource{
							{Name: "endpoint", Value: "{\"cluster_name\": \"endpoint\"}"},
						}}},
			}
			err := k8sClient.Create(context.Background(), ec)
			Expect(err).ToNot(HaveOccurred())
			By("waiting for the EnvoyConfig to be 'InSync'")
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "ec", Namespace: namespace}, ec)
				if err != nil {
					return false
				}
				if ec.Status.CacheState != marin3rv1alpha1.InSyncState {
					return false
				}
				return true
			}, 30*time.Second, 5*time.Second).Should(BeTrue())
		})

		When("EnvoyConfig is updated with wrong resources", func() {

			BeforeEach(func() {

				By("updating the EnvoyConfig with a wrong envoy v3 resource")
				patch := client.MergeFrom(ec.DeepCopy())
				ec.Spec.EnvoyResources = &marin3rv1alpha1.EnvoyResources{
					Endpoints: []marin3rv1alpha1.EnvoyResource{
						{Name: "endpoint", Value: "{\"wrong_key\": \"wrong_value\"}"},
					}}
				err := k8sClient.Patch(context.Background(), ec, patch)
				Expect(err).ToNot(HaveOccurred())

				By("waiting for a rollback to occur")
				Eventually(func() bool {
					err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "ec", Namespace: namespace}, ec)
					Expect(err).ToNot(HaveOccurred())
					if ec.Status.CacheState == marin3rv1alpha1.RollbackState {
						return true
					}
					return false
				}, 30*time.Second, 5*time.Second).Should(BeTrue())
			})

			It("should set the CacheOutOfSync condition", func() {

				Expect(ec.Status.Conditions.IsTrueFor(marin3rv1alpha1.CacheOutOfSyncCondition)).To(BeTrue())
			})

			When("resources are fixed in EnvoyConfig", func() {

				BeforeEach(func() {

					By("updating again the EnvoyConfig with a correct envoy v3 resource")
					patch := client.MergeFrom(ec.DeepCopy())
					ec.Spec.EnvoyResources = &marin3rv1alpha1.EnvoyResources{
						Endpoints: []marin3rv1alpha1.EnvoyResource{
							{Name: "endpoint", Value: "{\"cluster_name\": \"correct_endpoint\"}"},
						}}
					err := k8sClient.Patch(context.Background(), ec, patch)
					Expect(err).ToNot(HaveOccurred())

					By("waiting for status.cacheState to go back to 'InSync'")
					Eventually(func() bool {
						err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "ec", Namespace: namespace}, ec)
						Expect(err).ToNot(HaveOccurred())
						if ec.Status.CacheState == marin3rv1alpha1.InSyncState {
							return true
						}
						return false
					}, 30*time.Second, 5*time.Second).Should(BeTrue())
				})

				It("should clear the CacheOutOfSync condition", func() {

					Expect(ec.Status.Conditions.IsTrueFor(marin3rv1alpha1.CacheOutOfSyncCondition)).To(BeFalse())
				})

			})

		})

		Context("all EnvoyConfigRevisions are tainted", func() {
			BeforeEach(func() {

				ecrKey := types.NamespacedName{
					Name:      ec.Status.ConfigRevisions[0].Ref.Name,
					Namespace: namespace,
				}
				ecr := &marin3rv1alpha1.EnvoyConfigRevision{}
				err := k8sClient.Get(context.Background(), ecrKey, ecr)
				Expect(err).ToNot(HaveOccurred())

				By("updating the EnvoyConfigRevision with the RevisionTainted condition")
				patch := client.MergeFrom(ecr.DeepCopy())
				ecr.Status.Conditions.SetCondition(status.Condition{
					Type:   marin3rv1alpha1.RevisionTaintedCondition,
					Status: corev1.ConditionTrue,
				})
				err = k8sClient.Status().Patch(context.Background(), ecr, patch)
				Expect(err).ToNot(HaveOccurred())

				By("waiting for status.cacheState to be 'RollbackFailed'")
				Eventually(func() bool {
					err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "ec", Namespace: namespace}, ec)
					Expect(err).ToNot(HaveOccurred())
					if ec.Status.CacheState != marin3rv1alpha1.RollbackFailedState {
						return false
					}
					return true
				}, 30*time.Second, 5*time.Second).Should(BeTrue())
			})

			It("should set the CacheOutOfSync and the RollbackFailed conditions", func() {
				Expect(ec.Status.Conditions.IsTrueFor(marin3rv1alpha1.CacheOutOfSyncCondition)).To(BeTrue())
				Expect(ec.Status.Conditions.IsTrueFor(marin3rv1alpha1.RollbackFailedCondition)).To(BeTrue())
			})

			When("EnvoyConfig gets updated with a new set of correct resources", func() {

				BeforeEach(func() {

					By("updating again the EnvoyConfig with a correct envoy v3 resource")
					patch := client.MergeFrom(ec.DeepCopy())
					ec.Spec.EnvoyResources = &marin3rv1alpha1.EnvoyResources{
						Endpoints: []marin3rv1alpha1.EnvoyResource{
							{Name: "endpoint", Value: "{\"cluster_name\": \"correct_endpoint\"}"},
						}}
					err := k8sClient.Patch(context.Background(), ec, patch)
					Expect(err).ToNot(HaveOccurred())

					By("waiting for status.cacheState to be 'InSync'")
					Eventually(func() bool {
						err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "ec", Namespace: namespace}, ec)
						Expect(err).ToNot(HaveOccurred())
						if ec.Status.CacheState == marin3rv1alpha1.InSyncState {
							return true
						}
						return false
					}, 30*time.Second, 5*time.Second).Should(BeTrue())
				})

				It("should set status.cacheState back to InSync", func() {

					Expect(ec.Status.Conditions.IsTrueFor(marin3rv1alpha1.CacheOutOfSyncCondition)).To(BeFalse())
					Expect(ec.Status.Conditions.IsTrueFor(marin3rv1alpha1.RollbackFailedCondition)).To(BeFalse())

				})

			})
		})
	})

	Context("Upgrade/downgrade envoy API version", func() {
		var ec *marin3rv1alpha1.EnvoyConfig

		BeforeEach(func() {

			By("creating an EnvoyConfig with v2 resources")
			ec = &marin3rv1alpha1.EnvoyConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "ec", Namespace: namespace},
				Spec: marin3rv1alpha1.EnvoyConfigSpec{
					NodeID: nodeID,
					EnvoyResources: &marin3rv1alpha1.EnvoyResources{
						Endpoints: []marin3rv1alpha1.EnvoyResource{
							{Name: "endpoint", Value: "{\"cluster_name\": \"endpoint\"}"},
						}}},
			}
			err := k8sClient.Create(context.Background(), ec)
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "ec", Namespace: namespace}, ec)
				if err != nil {
					return false
				}
				return true
			}, 30*time.Second, 5*time.Second).Should(BeTrue())

			By("waiting for status.cacheState to be 'InSync'", func() {
				Eventually(func() bool {
					err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "ec", Namespace: namespace}, ec)
					Expect(err).ToNot(HaveOccurred())
					if ec.Status.CacheState == marin3rv1alpha1.InSyncState {
						return true
					}
					return false
				}, 30*time.Second, 5*time.Second).Should(BeTrue())
			})
		})

		Specify("spec.envoyAPI should be automatically set to 'v2' (the default value)", func() {
			Expect(ec.Spec.EnvoyAPI).To(Equal(pointer.StringPtr("v2")))
		})

		When("the EnvoyConfig is updated to v3", func() {

			BeforeEach(func() {

				By("updating the spec.envoyAPI field in the EnvoyConfig")
				patch := client.MergeFrom(ec.DeepCopy())
				ec.Spec.EnvoyAPI = pointer.StringPtr("v3")
				err := k8sClient.Patch(context.Background(), ec, patch)
				Expect(err).ToNot(HaveOccurred())

				By("waiting for a v3 EnvoyConfigRevision to be created and published")
				Eventually(func() bool {
					ecr, err := revisions.Get(context.Background(), k8sClient, namespace, filters.ByNodeID(nodeID), filters.ByEnvoyAPI(envoy.APIv3))
					if err != nil {
						return false
					}
					if ecr.Status.Conditions.IsTrueFor(marin3rv1alpha1.RevisionPublishedCondition) {
						return true
					}
					return false
				}, 30*time.Second, 5*time.Second).Should(BeTrue())

			})

			Specify("v2 EnvoyConfigRevision should not be deleted, but status.revisions should only contain references to v3 EnvoyConfigRevision resources", func() {

				By("getting the list of v2 EnvoyConfigRevisions from the API")
				ecrList, err := revisions.List(context.Background(), k8sClient, namespace, filters.ByNodeID(ec.Spec.NodeID), filters.ByEnvoyAPI(envoy.APIv2))
				Expect(err).ToNot(HaveOccurred())
				Expect(len(ecrList.Items)).To(Equal(1))

				By("checking all references in status.revisions point to v3 EnvoyConfigRevisions")
				Eventually(func() bool {
					err = k8sClient.Get(context.Background(), types.NamespacedName{Name: "ec", Namespace: namespace}, ec)
					Expect(err).NotTo(HaveOccurred())
					for _, ref := range ec.Status.ConfigRevisions {
						if strings.Contains(ref.Ref.Name, "-v2-") {
							return false
						}
					}
					return true
				}, 300*time.Second, 5*time.Second).Should(BeTrue())
			})

			Specify("both v2 and v3 version of the resources should be in the xDS server cache", func() {

				By("checking the v2 xDS server cache")
				{
					wantRevision := common.Hash(ec.Spec.EnvoyResources)
					wantSnap := xdss_v2.NewSnapshot(&cache_v2.Snapshot{
						Resources: [6]cache_v2.Resources{
							{Version: wantRevision, Items: map[string]cache_types.Resource{
								"endpoint": &envoy_api_v2.ClusterLoadAssignment{ClusterName: "endpoint"}}},
							{Version: wantRevision, Items: map[string]cache_types.Resource{}},
							{Version: wantRevision, Items: map[string]cache_types.Resource{}},
							{Version: wantRevision, Items: map[string]cache_types.Resource{}},
							{Version: wantRevision + "-557db659d4", Items: map[string]cache_types.Resource{}},
							{Version: wantRevision, Items: map[string]cache_types.Resource{}},
						}})
					gotV2Snap, err := ecrV2Reconciler.XdsCache.GetSnapshot(ec.Spec.NodeID)
					Expect(err).ToNot(HaveOccurred())
					Expect(testutil.SnapshotsAreEqual(gotV2Snap, wantSnap)).To(BeTrue())
				}

				By("checking the v3 xDS server cache")
				{
					wantRevision := common.Hash(ec.Spec.EnvoyResources)
					wantSnap := xdss_v3.NewSnapshot(&cache_v3.Snapshot{
						Resources: [6]cache_v3.Resources{
							{Version: wantRevision, Items: map[string]cache_types.Resource{
								"endpoint": &envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint"}}},
							{Version: wantRevision, Items: map[string]cache_types.Resource{}},
							{Version: wantRevision, Items: map[string]cache_types.Resource{}},
							{Version: wantRevision, Items: map[string]cache_types.Resource{}},
							{Version: wantRevision + "-557db659d4", Items: map[string]cache_types.Resource{}},
							{Version: wantRevision, Items: map[string]cache_types.Resource{}},
						}})
					gotV3Snap, err := ecrV3Reconciler.XdsCache.GetSnapshot(ec.Spec.NodeID)
					Expect(err).ToNot(HaveOccurred())
					Expect(testutil.SnapshotsAreEqual(gotV3Snap, wantSnap)).To(BeTrue())
				}

			})

			When("the EnvoyConfig is updated back to v2", func() {

				BeforeEach(func() {
					By("updating the spec.envoyAPI field in the EnvoyConfig")
					patch := client.MergeFrom(ec.DeepCopy())
					ec.Spec.EnvoyAPI = pointer.StringPtr("v2")
					err := k8sClient.Patch(context.Background(), ec, patch)
					Expect(err).ToNot(HaveOccurred())
				})

				Specify("v3 EnvoyConfigRevision should not be deleted, but status.revisions should only contain references to v2 EnvoyConfigRevision resources", func() {

					By("getting the list of v3 EnvoyConfigRevisions from the API")
					ecrList, err := revisions.List(context.Background(), k8sClient, namespace, filters.ByNodeID(ec.Spec.NodeID), filters.ByEnvoyAPI(envoy.APIv3))
					Expect(err).ToNot(HaveOccurred())
					Expect(len(ecrList.Items)).To(Equal(1))

					By("checking all references in status.revisions point to v2 EnvoyConfigRevisions")
					Eventually(func() bool {
						err = k8sClient.Get(context.Background(), types.NamespacedName{Name: "ec", Namespace: namespace}, ec)
						Expect(err).NotTo(HaveOccurred())
						for _, ref := range ec.Status.ConfigRevisions {
							if strings.Contains(ref.Ref.Name, "-v3-") {
								return false
							}
						}
						return true
					}, 30*time.Second, 5*time.Second).Should(BeTrue())
				})
			})
		})

	})

	Context("calling OnError() triggers a rollback", func() {
		var ec *marin3rv1alpha1.EnvoyConfig

		BeforeEach(func() {
			By("creating an EnvoyConfig")
			ec = &marin3rv1alpha1.EnvoyConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "ec", Namespace: namespace},
				Spec: marin3rv1alpha1.EnvoyConfigSpec{
					NodeID: nodeID,
					EnvoyResources: &marin3rv1alpha1.EnvoyResources{
						Endpoints: []marin3rv1alpha1.EnvoyResource{
							{Name: "endpoint", Value: "{\"cluster_name\": \"endpoint\"}"},
						}}},
			}
			err := k8sClient.Create(context.Background(), ec)
			Expect(err).ToNot(HaveOccurred())

			By("waiting for the EnvoyConfig to be 'InSync'")
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "ec", Namespace: namespace}, ec)
				if err != nil {
					return false
				}
				if ec.Status.CacheState != marin3rv1alpha1.InSyncState {
					return false
				}
				return true
			}, 30*time.Second, 5*time.Second).Should(BeTrue())
		})

		When("OnError is called", func() {

			BeforeEach(func() {
				OnErrorFn := rollback.OnError(cfg)
				version := common.Hash(ec.Spec.EnvoyResources)
				err := OnErrorFn(nodeID, version, "msg", envoy.APIv2)
				Expect(err).ToNot(HaveOccurred())
			})

			Specify("EnvoyConfig should have RollbackFailed condition", func() {

				Eventually(func() bool {
					err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "ec", Namespace: namespace}, ec)
					Expect(err).ToNot(HaveOccurred())
					return ec.Status.Conditions.IsTrueFor(marin3rv1alpha1.RollbackFailedCondition)
				}, 30*time.Second, 5*time.Second).Should(BeTrue())
			})
		})
	})
})
