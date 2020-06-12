package nodeconfigcache

import (
	"context"
	"fmt"

	"github.com/3scale/marin3r/pkg/apis"
	cachesv1alpha1 "github.com/3scale/marin3r/pkg/apis/caches/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	previousVersionPrefix string = "ReceivedPreviousVersion_"
)

// OnError returns a function that should be called when the envoy control plane receives
// a NACK to a discovery response from any of the gateways
func OnError(cfg *rest.Config) func(nodeID, version, msg string) error {

	return func(nodeID, version, msg string) error {

		// Create a client and register CRDs
		s := runtime.NewScheme()
		if err := apis.AddToScheme(s); err != nil {
			return err
		}
		cl, err := client.New(cfg, client.Options{Scheme: s})
		if err != nil {
			return err
		}

		// Get the nodeconfigcache that corresponds to the envoy node that returned the error
		ncrList := &cachesv1alpha1.NodeConfigRevisionList{}
		selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
			MatchLabels: map[string]string{nodeIDTag: nodeID, versionTag: version},
		})
		if err != nil {
			return err
		}
		err = cl.List(context.TODO(), ncrList, &client.ListOptions{LabelSelector: selector})
		if err != nil {
			return err
		}
		if len(ncrList.Items) != 1 {
			return fmt.Errorf("Got %v nodeconfigrevision objects when only 1 expected", len(ncrList.Items))
		}

		// Add the "ResourcesUpdateUnsuccessful" condition to the NodeConfigRevision object
		// unless the condition is already set
		ncr := &ncrList.Items[0]
		if !ncr.Status.Conditions.IsTrueFor(cachesv1alpha1.ResourcesOutOfSyncCondition) {
			patch := client.MergeFrom(ncr.DeepCopy())
			ncr.Status.Conditions.SetCondition(status.Condition{
				Type:    cachesv1alpha1.RevisionTaintedCondition,
				Status:  "True",
				Reason:  status.ConditionReason("GatewayReturnedNACK"),
				Message: fmt.Sprintf("A gateway returned NACK to the discovery response: '%s'", msg),
			})

			if err := cl.Status().Patch(context.TODO(), ncr, patch); err != nil {
				return err
			}
		}

		return nil
	}
}
