package rollback

import (
	"context"
	"fmt"

	marin3rv1alpha1 "github.com/3scale/marin3r/apis/marin3r/v1alpha1"
	envoy "github.com/3scale/marin3r/pkg/envoy"
	"github.com/3scale/marin3r/pkg/reconcilers/marin3r/envoyconfig/filters"
	"github.com/3scale/marin3r/pkg/reconcilers/marin3r/envoyconfig/revisions"

	"github.com/operator-framework/operator-lib/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	previousVersionPrefix string = "ReceivedPreviousVersion_"
)

// OnError returns a function that should be called when the envoy xDS server receives
// a NACK to a discovery response from any of the gateways
func OnError(cl client.Client) func(nodeID, version, msg string, envoyAPI envoy.APIVersion) error {

	return func(nodeID, version, msg string, envoyAPI envoy.APIVersion) error {

		// Get the envoyconfig that corresponds to the envoy node that returned the error
		ecr, err := revisions.Get(context.Background(), cl, "",
			filters.ByNodeID(nodeID), filters.ByVersion(version), filters.ByEnvoyAPI(envoyAPI))
		if err != nil {
			return err
		}

		if !ecr.Status.Conditions.IsTrueFor(marin3rv1alpha1.RevisionTaintedCondition) {
			patch := client.MergeFrom(ecr.DeepCopy())
			ecr.Status.Conditions.SetCondition(status.Condition{
				Type:    marin3rv1alpha1.RevisionTaintedCondition,
				Status:  "True",
				Reason:  status.ConditionReason("GatewayReturnedNACK"),
				Message: fmt.Sprintf("A gateway returned NACK to the discovery response: '%s'", msg),
			})

			if err := cl.Status().Patch(context.Background(), ecr, patch); err != nil {
				return err
			}
		}

		return nil
	}
}
