package controller

import (
	"github.com/3scale/marin3r/pkg/controller/discoveryservice"
	"github.com/3scale/marin3r/pkg/controller/discoveryservicecertificate"
	"github.com/3scale/marin3r/pkg/controller/envoyconfig"
	"github.com/3scale/marin3r/pkg/controller/envoyconfigrevision"
	"github.com/3scale/marin3r/pkg/controller/secret"

	xds_cache "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddToManager adds all Controllers to the Manager
func AddToManager(m manager.Manager, c *xds_cache.SnapshotCache) error {
	envoyconfig.Add(m, c)
	envoyconfigrevision.Add(m, c)
	secret.Add(m)

	return nil
}

// AddToOperatorManager adds the Operator Controllers to the OperatorManager
func AddToOperatorManager(m manager.Manager) error {
	discoveryservice.Add(m)
	discoveryservicecertificate.Add(m)

	return nil
}
