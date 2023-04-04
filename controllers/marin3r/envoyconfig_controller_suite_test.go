package controllers

import (
	"context"
	"fmt"
	"time"

	reconcilerutil "github.com/3scale-ops/basereconciler/util"
	marin3rv1alpha1 "github.com/3scale-ops/marin3r/apis/marin3r/v1alpha1"
	xdss_v3 "github.com/3scale-ops/marin3r/pkg/discoveryservice/xdss/v3"
	envoy "github.com/3scale-ops/marin3r/pkg/envoy"
	testutil "github.com/3scale-ops/marin3r/pkg/util/test"
	envoy_config_endpoint_v3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("EnvoyConfig controller", func() {
	var namespace string
	var nodeID string

	BeforeEach(func() {
		// Create a namespace for each block
		namespace = "test-ns-" + nameGenerator.Generate()
		// Create a nodeID for each block
		nodeID = nameGenerator.Generate()
		// Add any setup steps that needs to be executed before each test
		testNamespace := &corev1.Namespace{
			TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Namespace"},
			ObjectMeta: metav1.ObjectMeta{Name: namespace},
		}

		err := k8sClient.Create(context.Background(), testNamespace)
		Expect(err).ToNot(HaveOccurred())

		n := &corev1.Namespace{}
		Eventually(func() error {
			return k8sClient.Get(context.Background(), types.NamespacedName{Name: namespace}, n)
		}, 60*time.Second, 5*time.Second).ShouldNot(HaveOccurred())

	})

	AfterEach(func() {

		// Delete the namespace
		testNamespace := &corev1.Namespace{
			TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Namespace"},
			ObjectMeta: metav1.ObjectMeta{Name: namespace},
		}
		// Add any teardown steps that needs to be executed after each test
		err := k8sClient.Delete(context.Background(), testNamespace, client.PropagationPolicy(metav1.DeletePropagationForeground))
		Expect(err).ToNot(HaveOccurred())

		n := &corev1.Namespace{}
		Eventually(func() bool {
			err := k8sClient.Get(context.Background(), types.NamespacedName{Name: namespace}, n)
			if err != nil && errors.IsNotFound(err) {
				return false
			}
			return true
		}, 60*time.Second, 5*time.Second).Should(BeTrue())
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
							{Name: pointer.String("endpoint"), Value: "{\"cluster_name\": \"endpoint\"}"},
						}}},
			}
			err := k8sClient.Create(context.Background(), ec)
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() error {
				return k8sClient.Get(context.Background(), types.NamespacedName{Name: "ec", Namespace: namespace}, ec)
			}, 60*time.Second, 5*time.Second).ShouldNot(HaveOccurred())
		})

		When("EnvoyConfig is created", func() {

			It("should create a matching EnvoyConfigRevision and resources should be in the xDS cache", func() {

				Eventually(func() bool {
					err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "ec", Namespace: namespace}, ec)
					Expect(err).ToNot(HaveOccurred())
					if ec.Status.PublishedVersion == nil || *ec.Status.PublishedVersion == "" {
						return false
					}
					return true
				}, 60*time.Second, 5*time.Second).Should(BeTrue())

				// Get the EnvoyConfigRevision that should have been created
				ecr := &marin3rv1alpha1.EnvoyConfigRevision{}
				ecrKey := types.NamespacedName{Name: ec.Status.ConfigRevisions[len(ec.Status.ConfigRevisions)-1].Ref.Name, Namespace: namespace}
				err := k8sClient.Get(context.Background(), ecrKey, ecr)
				Expect(err).ToNot(HaveOccurred())

				// Validate the cache for the nodeID
				wantRevision := reconcilerutil.Hash(ec.Spec.EnvoyResources)
				wantSnap := xdss_v3.NewSnapshot().SetResources(envoy.Endpoint, []envoy.Resource{
					&envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint"},
				})

				// Wait for the revision to get written to the xDS cache
				Eventually(func() bool {
					gotV3Snap, err := ecrV3Reconciler.XdsCache.GetSnapshot(ec.Spec.NodeID)
					if err != nil {
						return false
					}
					return testutil.SnapshotsAreEqual(gotV3Snap, wantSnap)
				}, 60*time.Second, 5*time.Second).Should(BeTrue())

				err = k8sClient.Get(context.Background(), types.NamespacedName{Name: "ec", Namespace: namespace}, ec)
				Expect(err).ToNot(HaveOccurred())
				Expect(*ec.Status.PublishedVersion).To(Equal(wantRevision))
				Expect(*ec.Status.DesiredVersion).To(Equal(wantRevision))
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
							{Name: pointer.String("endpoint"), Value: "{\"cluster_name\": \"endpoint\"}"},
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
				if ec.Status.CacheState == nil || *ec.Status.CacheState != marin3rv1alpha1.InSyncState {
					return false
				}
				return true
			}, 60*time.Second, 5*time.Second).Should(BeTrue())
		})

		When("EnvoyConfig is updated with wrong resources", func() {

			BeforeEach(func() {

				By("updating the EnvoyConfig with a wrong envoy v3 resource")
				patch := client.MergeFrom(ec.DeepCopy())
				ec.Spec.EnvoyResources = &marin3rv1alpha1.EnvoyResources{
					Endpoints: []marin3rv1alpha1.EnvoyResource{
						{Name: pointer.String("endpoint"), Value: "{\"wrong_key\": \"wrong_value\"}"},
					}}
				err := k8sClient.Patch(context.Background(), ec, patch)
				Expect(err).ToNot(HaveOccurred())

				By("waiting for a rollback to occur")
				Eventually(func() bool {
					err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "ec", Namespace: namespace}, ec)
					Expect(err).ToNot(HaveOccurred())
					if ec.Status.CacheState != nil && *ec.Status.CacheState == marin3rv1alpha1.RollbackState {
						return true
					}
					return false
				}, 60*time.Second, 5*time.Second).Should(BeTrue())
			})

			It("should set the CacheOutOfSync condition", func() {

				Expect(meta.IsStatusConditionTrue(ec.Status.Conditions, marin3rv1alpha1.CacheOutOfSyncCondition)).To(BeTrue())
			})

			When("resources are fixed in EnvoyConfig", func() {

				BeforeEach(func() {

					By("updating again the EnvoyConfig with a correct envoy v3 resource")
					patch := client.MergeFrom(ec.DeepCopy())
					ec.Spec.EnvoyResources = &marin3rv1alpha1.EnvoyResources{
						Endpoints: []marin3rv1alpha1.EnvoyResource{
							{Name: pointer.String("endpoint"), Value: "{\"cluster_name\": \"correct_endpoint\"}"},
						}}
					err := k8sClient.Patch(context.Background(), ec, patch)
					Expect(err).ToNot(HaveOccurred())

					By("waiting for status.cacheState to go back to 'InSync'")
					Eventually(func() bool {
						err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "ec", Namespace: namespace}, ec)
						Expect(err).ToNot(HaveOccurred())
						if ec.Status.CacheState != nil && *ec.Status.CacheState == marin3rv1alpha1.InSyncState {
							return true
						}
						return false
					}, 60*time.Second, 5*time.Second).Should(BeTrue())
				})

				It("should clear the CacheOutOfSync condition", func() {

					Expect(meta.IsStatusConditionTrue(ec.Status.Conditions, marin3rv1alpha1.CacheOutOfSyncCondition)).To(BeFalse())
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
				meta.SetStatusCondition(&ecr.Status.Conditions, metav1.Condition{
					Type:   marin3rv1alpha1.RevisionTaintedCondition,
					Status: metav1.ConditionTrue,
					Reason: "test",
				})
				err = k8sClient.Status().Patch(context.Background(), ecr, patch)
				Expect(err).ToNot(HaveOccurred())

				By("waiting for status.cacheState to be 'RollbackFailed'")
				Eventually(func() bool {
					err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "ec", Namespace: namespace}, ec)
					Expect(err).ToNot(HaveOccurred())
					if ec.Status.CacheState == nil || *ec.Status.CacheState != marin3rv1alpha1.RollbackFailedState {
						return false
					}
					return true
				}, 60*time.Second, 5*time.Second).Should(BeTrue())
			})

			It("should set the CacheOutOfSync and the RollbackFailed conditions", func() {
				Expect(meta.IsStatusConditionTrue(ec.Status.Conditions, marin3rv1alpha1.CacheOutOfSyncCondition)).To(BeTrue())
				Expect(meta.IsStatusConditionTrue(ec.Status.Conditions, marin3rv1alpha1.RollbackFailedCondition)).To(BeTrue())
			})

			When("EnvoyConfig gets updated with a new set of correct resources", func() {

				BeforeEach(func() {

					By("updating again the EnvoyConfig with a correct envoy v3 resource")
					patch := client.MergeFrom(ec.DeepCopy())
					ec.Spec.EnvoyResources = &marin3rv1alpha1.EnvoyResources{
						Endpoints: []marin3rv1alpha1.EnvoyResource{
							{Name: pointer.String("endpoint"), Value: "{\"cluster_name\": \"correct_endpoint\"}"},
						}}
					err := k8sClient.Patch(context.Background(), ec, patch)
					Expect(err).ToNot(HaveOccurred())

					By("waiting for status.cacheState to be 'InSync'")
					Eventually(func() bool {
						err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "ec", Namespace: namespace}, ec)
						Expect(err).ToNot(HaveOccurred())
						if ec.Status.CacheState != nil && *ec.Status.CacheState == marin3rv1alpha1.InSyncState {
							return true
						}
						return false
					}, 60*time.Second, 5*time.Second).Should(BeTrue())
				})

				It("should set status.cacheState back to InSync", func() {

					Expect(meta.IsStatusConditionTrue(ec.Status.Conditions, marin3rv1alpha1.CacheOutOfSyncCondition)).To(BeFalse())
					Expect(meta.IsStatusConditionTrue(ec.Status.Conditions, marin3rv1alpha1.RollbackFailedCondition)).To(BeFalse())

				})

			})
		})
	})

})
