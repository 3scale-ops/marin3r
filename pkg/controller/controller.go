package controller

import (
	"github.com/3scale/marin3r/pkg/controller/configmap"
	"github.com/3scale/marin3r/pkg/controller/nodeconfigcache"
	"github.com/3scale/marin3r/pkg/controller/nodeconfigrevision"
	"github.com/3scale/marin3r/pkg/controller/secret"

	"github.com/3scale/marin3r/pkg/controller/discoveryservice"
	"github.com/3scale/marin3r/pkg/controller/servicediscoverycertificate"
	"github.com/3scale/marin3r/pkg/controller/sidecarconfig"

	xds_cache "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddToManager adds all Controllers to the Manager
func AddToManager(m manager.Manager, c *xds_cache.SnapshotCache) error {
	nodeconfigcache.Add(m, c)
	nodeconfigrevision.Add(m, c)
	secret.Add(m)
	configmap.Add(m)

	return nil
}

// AddToOperatorManager adds the Operator Controllers to the OperatorManager
func AddToOperatorManager(m manager.Manager) error {
	discoveryservice.Add(m)
	sidecarconfig.Add(m)
	servicediscoverycertificate.Add(m)

	return nil
}
