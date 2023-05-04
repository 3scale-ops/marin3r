package discover

import (
	"fmt"
	"net"

	"context"

	"github.com/3scale-ops/marin3r/pkg/envoy"
	envoy_resources "github.com/3scale-ops/marin3r/pkg/envoy/resources"
	"github.com/go-logr/logr"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Endpoints(ctx context.Context, cl client.Client, namespace string,
	clusterName, portName string, labelSelector *metav1.LabelSelector,
	generator envoy_resources.Generator, log logr.Logger) (envoy.Resource, error) {

	esl := &discoveryv1.EndpointSliceList{}

	selector, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		return nil, err
	}

	if err := cl.List(ctx, esl, client.InNamespace(namespace), client.MatchingLabelsSelector{Selector: selector}); err != nil {
		return nil, err
	}

	if len(esl.Items) == 0 {
		return nil, fmt.Errorf("no endpoints returned for label selector '%s'", selector)
	}

	hosts, err := endpointSlices_to_UpstreamHosts(esl, portName, log)
	if err != nil {
		return nil, err
	}
	endpoints := generator.NewClusterLoadAssignment(clusterName, hosts...)

	return endpoints, nil
}

func endpointSlices_to_UpstreamHosts(esl *discoveryv1.EndpointSliceList, portName string, log logr.Logger) ([]envoy.UpstreamHost, error) {
	hosts := []envoy.UpstreamHost{}
	var port *int32

	if esl.Items[0].AddressType == discoveryv1.AddressTypeFQDN {
		return nil, fmt.Errorf("unsupported address type FQDN")
	}

	// find the port
	for _, p := range esl.Items[0].Ports {
		if p.Name != nil && *p.Name == portName {
			port = p.Port
			break
		}
	}
	if port == nil {
		return nil, fmt.Errorf("no port by the name of '%s' found", portName)
	}

	// generate the list of hosts
	for _, endpointSlice := range esl.Items {

		for _, item := range endpointSlice.Endpoints {
			// only using address in position zero, see https://github.com/kubernetes/kubernetes/issues/106267
			if ip := net.ParseIP(item.Addresses[0]); ip == nil {
				log.Error(fmt.Errorf("'%s' doesn't look like an IP address", item.Addresses[0]), "error parsing endpoint")
				continue
			} else {

				hosts = append(hosts, envoy.UpstreamHost{
					IP:     ip,
					Port:   uint32(*port),
					Health: health(item.Conditions),
				})
			}

		}
	}

	return hosts, nil
}

func health(ec discoveryv1.EndpointConditions) envoy.EndpointHealthStatus {
	var health envoy.EndpointHealthStatus = envoy.HealthStatus_UNKNOWN

	// 1. determine if terminating
	if ec.Terminating != nil && *ec.Terminating {
		health = envoy.HealthStatus_DRAINING

	} else {

		// 2. Use 'serving' to determine health
		if ec.Serving != nil {

			if *ec.Serving {
				health = envoy.HealthStatus_HEALTHY
			} else {
				health = envoy.HealthStatus_UNHEALTHY
			}

		} else {

			// 3. fall back to 'ready' if 'serving' not set
			if ec.Ready != nil {

				if *ec.Ready {
					health = envoy.HealthStatus_HEALTHY
				} else {
					health = envoy.HealthStatus_UNHEALTHY
				}

			} else {
				// neither 'ready' nor 'serving' fields availabel, unable
				// to determine health
				health = envoy.HealthStatus_UNKNOWN
			}

		}

	}

	return health
}
