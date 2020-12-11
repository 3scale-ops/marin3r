package controllers

import (
	"testing"

	marin3rv1alpha1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
	"github.com/operator-framework/operator-lib/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func TestIsInitialized(t *testing.T) {
	tests := []struct {
		name                       string
		envoyConfigRevisionFactory func() *marin3rv1alpha1.EnvoyConfigRevision
		want                       bool
	}{
		{
			"Initializes the resource",
			func() *marin3rv1alpha1.EnvoyConfigRevision {
				return &marin3rv1alpha1.EnvoyConfigRevision{}
			},
			false,
		},
		{
			"Returns true if already initialized",
			func() *marin3rv1alpha1.EnvoyConfigRevision {
				return &marin3rv1alpha1.EnvoyConfigRevision{
					ObjectMeta: metav1.ObjectMeta{
						Finalizers: []string{marin3rv1alpha1.EnvoyConfigRevisionFinalizer},
					},
					Spec: marin3rv1alpha1.EnvoyConfigRevisionSpec{
						EnvoyAPI:      pointer.StringPtr("v2"),
						Serialization: pointer.StringPtr("json"),
					},
					Status: marin3rv1alpha1.EnvoyConfigRevisionStatus{
						Conditions: status.Conditions{
							{
								Type:    marin3rv1alpha1.RevisionPublishedCondition,
								Reason:  "Initialized",
								Status:  corev1.ConditionFalse,
								Message: "EnvoyConfigRevision is not marked as the published revision",
							},
							{
								Type:    marin3rv1alpha1.ResourcesOutOfSyncCondition,
								Reason:  "Initialized",
								Status:  corev1.ConditionTrue,
								Message: "Resources are not in sync with the xDS server cache",
							},
						},
					},
				}
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsInitialized(tt.envoyConfigRevisionFactory()); got != tt.want {
				t.Errorf("IsInitialized() = %v, want %v", got, tt.want)
			}
		})
	}
}
