package lockedresources

import (
	"github.com/redhat-cop/operator-utils/pkg/util/lockedresourcecontroller"
	"github.com/redhat-cop/operator-utils/pkg/util/lockedresourcecontroller/lockedresource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// Reconciler computes a list of resources that it needs to keep in place
type Reconciler struct {
	lockedresourcecontroller.EnforcingReconciler
}

// NewFromManager constructs a new Reconciler from the given manager
func NewFromManager(mgr manager.Manager, recorder record.EventRecorder, clusterWatchers bool) Reconciler {
	return Reconciler{
		EnforcingReconciler: lockedresourcecontroller.NewFromManager(mgr, mgr.GetEventRecorderFor("DiscoveryService"), clusterWatchers),
	}
}

// GeneratorFunction is a function that returns a client.Object
type GeneratorFunction func() client.Object

// LockedResource is a struct that instructs the reconciler how to
// generate and reconcile a resource
type LockedResource struct {
	GeneratorFn  GeneratorFunction
	ExcludePaths []string
}

// NewLockedResources returns the list of lockedresource.LockedResource that the reconciler needs to enforce
func (r *Reconciler) NewLockedResources(list []LockedResource, owner client.Object) ([]lockedresource.LockedResource, error) {
	resources := []lockedresource.LockedResource{}
	var err error

	for _, res := range list {
		resources, err = add(resources, res.GeneratorFn, res.ExcludePaths, owner, r.GetScheme())
		if err != nil {
			return nil, err
		}
	}
	return resources, nil
}

func add(resources []lockedresource.LockedResource, fn GeneratorFunction, excludedPaths []string,
	owner client.Object, scheme *runtime.Scheme) ([]lockedresource.LockedResource, error) {

	u, err := newUnstructured(fn, owner, scheme)
	if err != nil {
		return nil, err
	}

	res := lockedresource.LockedResource{
		Unstructured:  u,
		ExcludedPaths: excludedPaths,
	}

	return append(resources, res), nil
}

func newUnstructured(fn GeneratorFunction, owner client.Object, scheme *runtime.Scheme) (unstructured.Unstructured, error) {
	o := fn()
	if err := controllerutil.SetControllerReference(owner, o, scheme); err != nil {
		return unstructured.Unstructured{}, err
	}
	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(o)
	if err != nil {
		return unstructured.Unstructured{}, err
	}
	return unstructured.Unstructured{Object: u}, nil
}
