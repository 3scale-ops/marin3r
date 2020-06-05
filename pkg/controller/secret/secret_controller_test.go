package secret

import (
	"context"
	"testing"

	cachesv1alpha1 "github.com/3scale/marin3r/pkg/apis/caches/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var s *runtime.Scheme = scheme.Scheme

func init() {
	s.AddKnownTypes(cachesv1alpha1.SchemeGroupVersion,
		&cachesv1alpha1.NodeConfigRevision{},
		&cachesv1alpha1.NodeConfigRevisionList{},
		&cachesv1alpha1.NodeConfigCache{},
	)
}

func TestReconcileSecret_Reconcile(t *testing.T) {
	t.Run("Adds ResourcesOutOfSyncCondition to NCR when a refered secret changes", func(t *testing.T) {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret",
				Namespace: "default",
			},
			Type: corev1.SecretTypeTLS,
			Data: map[string][]byte{
				"tls.key": []byte("xxxxx"),
				"tls.crt": []byte("xxxxx"),
			},
		}
		ncr := &cachesv1alpha1.NodeConfigRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ncr",
				Namespace: "default",
			},
			Spec: cachesv1alpha1.NodeConfigRevisionSpec{
				NodeID:  "node1",
				Version: "xxxx",
				Resources: &cachesv1alpha1.EnvoyResources{
					Secrets: []cachesv1alpha1.EnvoySecretResource{{
						Name: "secret",
						Ref: corev1.SecretReference{
							Name:      "secret",
							Namespace: "default",
						}}}},
			},
			Status: cachesv1alpha1.NodeConfigRevisionStatus{
				Conditions: []status.Condition{{Type: cachesv1alpha1.RevisionPublishedCondition, Status: corev1.ConditionTrue}},
			},
		}

		cl := fake.NewFakeClient(secret, ncr)
		r := &ReconcileSecret{client: cl, scheme: s}

		_, gotErr := r.Reconcile(reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      "secret",
				Namespace: "default",
			},
		})

		if gotErr != nil {
			t.Errorf("TestReconcileSecret_Reconcile() returned error: '%v'", gotErr)
			return
		}

		r.client.Get(context.TODO(), types.NamespacedName{Name: "ncr", Namespace: "default"}, ncr)
		if !ncr.Status.Conditions.IsTrueFor(cachesv1alpha1.ResourcesOutOfSyncCondition) {
			t.Errorf("TestReconcileSecret_Reconcile() condition 'ResourcesOutOfSyncCondition' was not set in NodeCacheRevision")
		}
	})
}
