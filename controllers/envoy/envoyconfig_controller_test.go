package controllers

import (
	"context"
	"testing"
	"time"

	envoyv1alpha1 "github.com/3scale/marin3r/apis/envoy/v1alpha1"
	xdss_v2 "github.com/3scale/marin3r/pkg/discoveryservice/xdss/v2"
	testutil "github.com/3scale/marin3r/pkg/util/test"
	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	cache_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cache_v2 "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var s *runtime.Scheme = scheme.Scheme

func init() {
	s.AddKnownTypes(envoyv1alpha1.GroupVersion,
		&envoyv1alpha1.EnvoyConfigRevision{},
		&envoyv1alpha1.EnvoyConfigRevisionList{},
		&envoyv1alpha1.EnvoyConfig{},
	)
}

var _ = Describe("EnvoyConfigRevision controller", func() {
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

	Context("Using v2 envoy API version", func() {
		var ec *envoyv1alpha1.EnvoyConfig

		BeforeEach(func() {
			// Create a v2 EnvoyConfigRevision for each block
			ec = &envoyv1alpha1.EnvoyConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "ec", Namespace: namespace},
				Spec: envoyv1alpha1.EnvoyConfigSpec{
					EnvoyAPI: pointer.StringPtr("v2"),
					NodeID:   nodeID,
					EnvoyResources: &envoyv1alpha1.EnvoyResources{
						Endpoints: []envoyv1alpha1.EnvoyResource{
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

			It("Should create a matching EnvoyConfigRevision and resources should be in the xDS cache", func() {

				Eventually(func() bool {
					err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "ec", Namespace: namespace}, ec)
					Expect(err).ToNot(HaveOccurred())
					if ec.Status.PublishedVersion == "" {
						return false
					}
					return true
				}, 30*time.Second, 5*time.Second).Should(BeTrue())

				// Get the EnvoyConfigRevision that should have been created
				ecr := &envoyv1alpha1.EnvoyConfigRevision{}
				ecrKey := types.NamespacedName{Name: ec.Status.ConfigRevisions[len(ec.Status.ConfigRevisions)-1].Ref.Name, Namespace: namespace}
				err := k8sClient.Get(context.Background(), ecrKey, ecr)
				Expect(err).ToNot(HaveOccurred())

				// Validate the cache for the nodeID
				wantRevision := calculateRevisionHash(ec.Spec.EnvoyResources)
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
				Expect(ec.Status.ConfigRevisions[0].Ref.Name).To(Equal(ec.Spec.NodeID + "-" + wantRevision))
			})
		})

		When("When EnvoyConfig is updated", func() {
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
				ec.Spec.EnvoyResources = &envoyv1alpha1.EnvoyResources{
					Clusters: []envoyv1alpha1.EnvoyResource{
						{Name: "cluster", Value: "{\"name\": \"cluster\"}"},
					}}
				err := k8sClient.Update(context.Background(), ec)
				Expect(err).ToNot(HaveOccurred())

				// Wait for the new revision to get published
				wantRevision = calculateRevisionHash(ec.Spec.EnvoyResources)
				Eventually(func() bool {
					err := k8sClient.Get(context.Background(), types.NamespacedName{Name: "ec", Namespace: namespace}, ec)
					Expect(err).ToNot(HaveOccurred())
					if ec.Status.PublishedVersion != wantRevision {
						return false
					}
					return true
				}, 30*time.Second, 5*time.Second).Should(BeTrue())
			})

			It("Should create a new matching EnvoyConfigRevision and new resources should be in the xDS cache", func() {

				Expect(ec.Status.PublishedVersion).To(Equal(wantRevision))
				Expect(ec.Status.DesiredVersion).To(Equal(wantRevision))
				Expect(len(ec.Status.ConfigRevisions)).To(Equal(2))
				Expect(ec.Status.ConfigRevisions[len(ec.Status.ConfigRevisions)-1].Ref.Name).To(Equal(ec.Spec.NodeID + "-" + wantRevision))

				// Get the EnvoyConfigRevision that should have been created
				ecr := &envoyv1alpha1.EnvoyConfigRevision{}
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

			It("Should publish an already existent EnvoyConfigRevision if one already matches the current EnvoyConfig resources", func() {

				// Set the previous set of resources in the EnvoyConfig
				ec.Spec.EnvoyResources = &envoyv1alpha1.EnvoyResources{
					Endpoints: []envoyv1alpha1.EnvoyResource{
						{Name: "endpoint", Value: "{\"cluster_name\": \"endpoint\"}"},
					}}
				err := k8sClient.Update(context.Background(), ec)
				Expect(err).ToNot(HaveOccurred())

				// Wait for the existent revision to get published
				wantRevision = calculateRevisionHash(ec.Spec.EnvoyResources)
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
				Expect(ec.Status.ConfigRevisions[len(ec.Status.ConfigRevisions)-1].Ref.Name).To(Equal(ec.Spec.NodeID + "-" + wantRevision))

				// Get the EnvoyConfigRevision that should have been created
				ecr := &envoyv1alpha1.EnvoyConfigRevision{}
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

	// TODO: "From top to bottom, publishes the first non tainted revision of the ConfigRevisions list"
	// TODO: "Set RollbackFailed state if all versions are tainted"
	// TODO: "Set RollbackFailed state if all versions are tainted"

})

func Test_contains(t *testing.T) {
	type args struct {
		list []string
		s    string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "True -> key in slice",
			args: args{list: []string{"a", "b", "c"}, s: "a"},
			want: true,
		},
		{
			name: "False -> key not in slice",
			args: args{list: []string{"a", "b", "c"}, s: "z"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := contains(tt.args.list, tt.args.s); got != tt.want {
				t.Errorf("contains() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnvoyConfigReconciler_getVersionToPublish(t *testing.T) {

	tests := []struct {
		name    string
		ec      *envoyv1alpha1.EnvoyConfig
		ecrList *envoyv1alpha1.EnvoyConfigRevisionList
		want    string
		wantErr bool
	}{
		{
			name: "Returns the desiredVersion on seeing a new version",
			ec: &envoyv1alpha1.EnvoyConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "ec", Namespace: "default"},
				Spec: envoyv1alpha1.EnvoyConfigSpec{
					NodeID:         "node1",
					EnvoyResources: &envoyv1alpha1.EnvoyResources{},
				},
				Status: envoyv1alpha1.EnvoyConfigStatus{
					ConfigRevisions: []envoyv1alpha1.ConfigRevisionRef{
						{Version: "xxx", Ref: corev1.ObjectReference{Name: "ecr1", Namespace: "default"}},
					},
				},
			},
			ecrList: &envoyv1alpha1.EnvoyConfigRevisionList{
				TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "List"},
				Items: []envoyv1alpha1.EnvoyConfigRevision{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "ecr1",
							Namespace: "default",
							Labels:    map[string]string{nodeIDTag: "node1"},
						},
						Spec: envoyv1alpha1.EnvoyConfigRevisionSpec{
							NodeID:         "node1",
							Version:        "xxx",
							EnvoyResources: &envoyv1alpha1.EnvoyResources{},
						},
					},
				},
			},
			want:    "xxx",
			wantErr: false,
		},
		{
			name: "Returns the highest index untainted revision of the ConfigRevision list",
			ec: &envoyv1alpha1.EnvoyConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "ec", Namespace: "default"},
				Spec: envoyv1alpha1.EnvoyConfigSpec{
					NodeID:         "node1",
					EnvoyResources: &envoyv1alpha1.EnvoyResources{},
				},
				Status: envoyv1alpha1.EnvoyConfigStatus{
					ConfigRevisions: []envoyv1alpha1.ConfigRevisionRef{
						{Version: "xxx", Ref: corev1.ObjectReference{Name: "ecr1", Namespace: "default"}},
						{Version: "zzz", Ref: corev1.ObjectReference{Name: "ecr2", Namespace: "default"}},
					},
				},
			},
			ecrList: &envoyv1alpha1.EnvoyConfigRevisionList{
				TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "List"},
				Items: []envoyv1alpha1.EnvoyConfigRevision{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "ecr1",
							Namespace: "default",
							Labels:    map[string]string{nodeIDTag: "node1"},
						},
						Spec: envoyv1alpha1.EnvoyConfigRevisionSpec{
							NodeID:         "node1",
							Version:        "xxx",
							EnvoyResources: &envoyv1alpha1.EnvoyResources{},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "ecr2",
							Namespace: "default",
							Labels:    map[string]string{nodeIDTag: "node1"},
						},
						Spec: envoyv1alpha1.EnvoyConfigRevisionSpec{
							NodeID:         "node1",
							Version:        "zzz",
							EnvoyResources: &envoyv1alpha1.EnvoyResources{},
						},
						Status: envoyv1alpha1.EnvoyConfigRevisionStatus{
							Conditions: status.NewConditions(status.Condition{
								Type:   envoyv1alpha1.RevisionTaintedCondition,
								Status: corev1.ConditionTrue,
							}),
						},
					},
				},
			},
			want:    "xxx",
			wantErr: false,
		},
		{
			name: "Returns an error if all revisions are tainted",
			ec: &envoyv1alpha1.EnvoyConfig{
				ObjectMeta: metav1.ObjectMeta{Name: "ec", Namespace: "default"},
				Spec: envoyv1alpha1.EnvoyConfigSpec{
					NodeID:         "node1",
					EnvoyResources: &envoyv1alpha1.EnvoyResources{},
				},
				Status: envoyv1alpha1.EnvoyConfigStatus{
					ConfigRevisions: []envoyv1alpha1.ConfigRevisionRef{
						{Version: "xxx", Ref: corev1.ObjectReference{Name: "ecr1", Namespace: "default"}},
						{Version: "zzz", Ref: corev1.ObjectReference{Name: "ecr2", Namespace: "default"}},
					},
				},
			},
			ecrList: &envoyv1alpha1.EnvoyConfigRevisionList{
				TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "List"},
				Items: []envoyv1alpha1.EnvoyConfigRevision{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "ecr1",
							Namespace: "default",
							Labels:    map[string]string{nodeIDTag: "node1"},
						},
						Spec: envoyv1alpha1.EnvoyConfigRevisionSpec{
							NodeID:         "node1",
							Version:        "xxx",
							EnvoyResources: &envoyv1alpha1.EnvoyResources{},
						},
						Status: envoyv1alpha1.EnvoyConfigRevisionStatus{
							Conditions: status.NewConditions(status.Condition{
								Type:   envoyv1alpha1.RevisionTaintedCondition,
								Status: corev1.ConditionTrue,
							}),
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "ecr2",
							Namespace: "default",
							Labels:    map[string]string{nodeIDTag: "node1"},
						},
						Spec: envoyv1alpha1.EnvoyConfigRevisionSpec{
							NodeID:         "node1",
							Version:        "zzz",
							EnvoyResources: &envoyv1alpha1.EnvoyResources{},
						},
						Status: envoyv1alpha1.EnvoyConfigRevisionStatus{
							Conditions: status.NewConditions(status.Condition{
								Type:   envoyv1alpha1.RevisionTaintedCondition,
								Status: corev1.ConditionTrue,
							}),
						},
					},
				},
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &EnvoyConfigReconciler{
				Client: fake.NewFakeClient(tt.ec, tt.ecrList),
				Scheme: s,
				Log:    ctrl.Log.WithName("test"),
			}
			got, err := r.getVersionToPublish(context.TODO(), tt.ec)
			if (err != nil) != tt.wantErr {
				t.Errorf("EnvoyConfigReconciler.getVersionToPublish() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EnvoyConfigReconciler.getVersionToPublish() = %v, want %v", got, tt.want)
			}
		})

	}
}
