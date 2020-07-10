package secret

import (
	"context"
	"testing"

	marin3rv1alpha1 "github.com/3scale/marin3r/pkg/apis/marin3r/v1alpha1"
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
	s.AddKnownTypes(marin3rv1alpha1.SchemeGroupVersion,
		&marin3rv1alpha1.EnvoyConfigRevision{},
		&marin3rv1alpha1.EnvoyConfigRevisionList{},
		&marin3rv1alpha1.EnvoyConfig{},
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
		ecr := &marin3rv1alpha1.EnvoyConfigRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ecr",
				Namespace: "default",
			},
			Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
				NodeID:  "node1",
				Version: "xxxx",
				EnvoyResources: &marin3rv1alpha1.EnvoyResources{
					Secrets: []marin3rv1alpha1.EnvoySecretResource{{
						Name: "secret",
						Ref: corev1.SecretReference{
							Name:      "secret",
							Namespace: "default",
						}}}},
			},
			Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
				Conditions: []status.Condition{{Type: marin3rv1alpha1.RevisionPublishedCondition, Status: corev1.ConditionTrue}},
			},
		}

		cl := fake.NewFakeClient(secret, ecr)
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

		r.client.Get(context.TODO(), types.NamespacedName{Name: "ecr", Namespace: "default"}, ecr)
		if !ecr.Status.Conditions.IsTrueFor(marin3rv1alpha1.ResourcesOutOfSyncCondition) {
			t.Errorf("TestReconcileSecret_Reconcile() condition 'ResourcesOutOfSyncCondition' was not set in NodeCacheRevision")
		}
	})
}
