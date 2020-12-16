package rollback

import (
	"context"
	"fmt"

	marin3rv1alpha1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
	envoy "github.com/3scale/marin3r/pkg/envoy"
	"github.com/3scale/marin3r/pkg/reconcilers/marin3r/envoyconfig/filters"

	"github.com/operator-framework/operator-lib/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	previousVersionPrefix string = "ReceivedPreviousVersion_"
)

// OnError returns a function that should be called when the envoy control plane receives
// a NACK to a discovery response from any of the gateways
func OnError(cl client.Client) func(nodeID, version, msg string, envoyAPI envoy.APIVersion) error {

	return func(nodeID, version, msg string, envoyAPI envoy.APIVersion) error {

		// Get the envoyconfig that corresponds to the envoy node that returned the error
		ecrList := &marin3rv1alpha1.EnvoyConfigRevisionList{}
		selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
			MatchLabels: map[string]string{
				filters.NodeIDTag:   nodeID,
				filters.VersionTag:  version,
				filters.EnvoyAPITag: string(envoyAPI),
			},
		})
		if err != nil {
			return err
		}
		err = cl.List(context.TODO(), ecrList, &client.ListOptions{LabelSelector: selector})
		if err != nil {
			return err
		}
		if len(ecrList.Items) != 1 {
			return fmt.Errorf("Got %v envoyconfigrevision objects when only 1 expected", len(ecrList.Items))
		}

		// Add the "ResourcesUpdateUnsuccessful" condition to the EnvoyConfigRevision object
		// unless the condition is already set
		ecr := &ecrList.Items[0]
		if !ecr.Status.Conditions.IsTrueFor(marin3rv1alpha1.RevisionTaintedCondition) {
			patch := client.MergeFrom(ecr.DeepCopy())
			ecr.Status.Conditions.SetCondition(status.Condition{
				Type:    marin3rv1alpha1.RevisionTaintedCondition,
				Status:  "True",
				Reason:  status.ConditionReason("GatewayReturnedNACK"),
				Message: fmt.Sprintf("A gateway returned NACK to the discovery response: '%s'", msg),
			})

			if err := cl.Status().Patch(context.TODO(), ecr, patch); err != nil {
				return err
			}
		}

		return nil
	}
}
