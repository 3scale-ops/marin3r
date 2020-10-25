package controllers

import (
	"context"

	operatorv1alpha1 "github.com/3scale/marin3r/apis/operator/v1alpha1"
	"github.com/3scale/marin3r/pkg/webhook"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	MutatingWebhookTimeout int32 = 5
)

// reconcileMutatingWebhook keeps the marin3r MutatingWebhookConfiguration object in sync with the desired state
func (r *DiscoveryServiceReconciler) reconcileMutatingWebhook(ctx context.Context) (reconcile.Result, error) {

	r.Log.V(1).Info("Reconciling MutatingWebhookConfiguration")

	caBundle, err := r.getCABundle(ctx)
	if err != nil {
		return reconcile.Result{}, err
	}

	existent := &admissionregistrationv1.MutatingWebhookConfiguration{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: OwnedObjectName(r.ds)}, existent)

	if err != nil {
		if errors.IsNotFound(err) {
			existent = r.genMutatingWebhookConfigurationObject(caBundle)
			if err := controllerutil.SetControllerReference(r.ds, existent, r.Scheme); err != nil {
				return reconcile.Result{}, err
			}
			if err := r.Client.Create(ctx, existent); err != nil {
				return reconcile.Result{}, err
			}
			r.Log.Info("Created MutatingWebhookConfiguration")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// We just reconcile the "Webhooks" field
	desired := r.genMutatingWebhookConfigurationObject(caBundle)

	if !equality.Semantic.DeepEqual(existent.Webhooks, desired.Webhooks) {
		patch := client.MergeFrom(existent.DeepCopy())
		existent.Webhooks = desired.Webhooks
		if err := r.Client.Patch(ctx, existent, patch); err != nil {
			return reconcile.Result{}, err
		}
		r.Log.Info("Patched MutatingWebhookConfiguration")
	}

	return reconcile.Result{}, nil
}

func (r *DiscoveryServiceReconciler) genMutatingWebhookConfigurationObject(caBundle []byte) *admissionregistrationv1.MutatingWebhookConfiguration {

	return &admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:   OwnedObjectName(r.ds),
			Labels: Labels(r.ds),
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{
			{
				Name: "sidecar-injector.marin3r.3scale.net",
				NamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						operatorv1alpha1.DiscoveryServiceLabelKey: r.ds.GetName(),
					},
				},
				ObjectSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						operatorv1alpha1.DiscoveryServiceEnabledKey: operatorv1alpha1.DiscoveryServiceEnabledValue,
					},
				},
				SideEffects: func() *admissionregistrationv1.SideEffectClass {
					s := admissionregistrationv1.SideEffectClassNone
					return &s
				}(),
				AdmissionReviewVersions: []string{
					admissionregistrationv1.SchemeGroupVersion.Version,
					admissionregistrationv1beta1.SchemeGroupVersion.Version,
				},
				TimeoutSeconds: pointer.Int32Ptr(MutatingWebhookTimeout),
				FailurePolicy: func() *admissionregistrationv1.FailurePolicyType {
					s := admissionregistrationv1.Fail
					return &s
				}(),
				Rules: []admissionregistrationv1.RuleWithOperations{
					{
						Operations: []admissionregistrationv1.OperationType{
							admissionregistrationv1.Create,
						},
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{corev1.SchemeGroupVersion.Group},
							APIVersions: []string{corev1.SchemeGroupVersion.Version},
							Resources:   []string{"pods"},
							Scope: func() *admissionregistrationv1.ScopeType {
								s := admissionregistrationv1.NamespacedScope
								return &s
							}(),
						},
					},
				},
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Name:      OwnedObjectName(r.ds),
						Namespace: OwnedObjectNamespace(r.ds),
						Path:      pointer.StringPtr(webhook.MutatePath),
						Port:      pointer.Int32Ptr(443),
					},
					CABundle: caBundle,
				},
				MatchPolicy: func() *admissionregistrationv1.MatchPolicyType {
					s := admissionregistrationv1.Equivalent
					return &s
				}(),
				ReinvocationPolicy: func() *admissionregistrationv1.ReinvocationPolicyType {
					s := admissionregistrationv1.NeverReinvocationPolicy
					return &s
				}(),
			},
		},
	}
}

func (r *DiscoveryServiceReconciler) getCABundle(ctx context.Context) ([]byte, error) {

	secret := &corev1.Secret{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: getCACertName(r.ds), Namespace: OwnedObjectNamespace(r.ds)}, secret); err != nil {
		return nil, err
	}

	return secret.Data["tls.crt"], nil
}
