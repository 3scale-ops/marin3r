package resource_extensions

import (
	"context"
	"fmt"

	"github.com/3scale-ops/basereconciler/reconciler"
	operatorv1alpha1 "github.com/3scale-ops/marin3r/apis/operator.marin3r/v1alpha1"
	discoveryservicecertificate "github.com/3scale-ops/marin3r/pkg/reconcilers/operator/discoveryservicecertificate"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var _ reconciler.Resource = DiscoveryServiceCertificateTemplate{}

// DiscoveryServiceCertificateTemplate has methods to generate and reconcile a DiscoveryServiceCertificate
type DiscoveryServiceCertificateTemplate struct {
	Template  func() *operatorv1alpha1.DiscoveryServiceCertificate
	IsEnabled bool
}

// Build returns a DiscoveryServiceCertificate resource
func (dsct DiscoveryServiceCertificateTemplate) Build(ctx context.Context, cl client.Client) (client.Object, error) {
	return dsct.Template().DeepCopy(), nil
}

// Enabled indicates if the resource should be present or not
func (dsct DiscoveryServiceCertificateTemplate) Enabled() bool {
	return dsct.IsEnabled
}

// ResourceReconciler implements a generic reconciler for DiscoveryServiceCertificate resources
func (dsct DiscoveryServiceCertificateTemplate) ResourceReconciler(ctx context.Context, cl client.Client, obj client.Object) error {
	logger := log.FromContext(ctx, "kind", "DiscoveryServiceCertificate", "resource", obj.GetName())

	needsUpdate := false
	desired := obj.(*operatorv1alpha1.DiscoveryServiceCertificate)

	instance := &operatorv1alpha1.DiscoveryServiceCertificate{}
	err := cl.Get(ctx, types.NamespacedName{Name: desired.GetName(), Namespace: desired.GetNamespace()}, instance)
	if err != nil {
		if errors.IsNotFound(err) {

			if dsct.Enabled() {
				err = cl.Create(ctx, desired)
				if err != nil {
					return fmt.Errorf("unable to create object: " + err.Error())
				}
				logger.Info("resource created")
				return nil

			} else {
				return nil
			}
		}

		return err
	}

	/* Delete and return if not enabled */
	if !dsct.Enabled() {
		err := cl.Delete(ctx, instance)
		if err != nil {
			return fmt.Errorf("unable to delete object: " + err.Error())
		}
		logger.Info("resource deleted")
		return nil
	}

	/* Reconcile metadata */
	if !equality.Semantic.DeepEqual(instance.GetLabels(), desired.GetLabels()) {
		instance.ObjectMeta.Labels = desired.GetLabels()
		fmt.Println("LABELS NEED UPDATE")
		needsUpdate = true
	}

	/* Reconcile spec */
	discoveryservicecertificate.IsInitialized(desired)
	if !equality.Semantic.DeepEqual(instance.Spec, desired.Spec) {
		instance.Spec = desired.Spec
		needsUpdate = true
	}

	if needsUpdate {
		err := cl.Update(ctx, instance)
		if err != nil {
			return err
		}
		logger.Info("Resource updated")
	}

	return nil
}
