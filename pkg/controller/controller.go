package controller

import (
	// "github.com/3scale/marin3r/pkg/controller/configmap"
	"github.com/3scale/marin3r/pkg/controller/nodeconfigcache"
	// "github.com/3scale/marin3r/pkg/controller/secret"
	xds_cache "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddToManagerFuncs is a list of functions to add all Controllers to the Manager
var AddToManagerFuncs []func(manager.Manager) error

// AddToManager adds all Controllers to the Manager
func AddToManager(m manager.Manager, c *xds_cache.SnapshotCache) error {
	for _, f := range AddToManagerFuncs {
		if err := f(m); err != nil {
			return err
		}
	}
	nodeconfigcache.Add(m, c)
	return nil
}
