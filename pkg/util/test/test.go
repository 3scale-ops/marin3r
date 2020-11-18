package test

import (
	xdss "github.com/3scale/marin3r/pkg/discoveryservice/xdss"

	envoy_resources "github.com/3scale/marin3r/pkg/envoy/resources"
	envoy_resources_v2 "github.com/3scale/marin3r/pkg/envoy/resources/v2"
	envoy_resources_v3 "github.com/3scale/marin3r/pkg/envoy/resources/v3"
)

func SnapshotsAreEqual(x xdss.Snapshot, y xdss.Snapshot) bool {

	rTypesV2 := envoy_resources_v2.Mappings()
	for rType := range rTypesV2 {
		if !envoy_resources.ResourcesEqual(x.GetResources(rType), y.GetResources(rType)) {
			return false
		}
		if x.GetVersion(rType) != y.GetVersion(rType) {
			return false
		}
	}

	rTypesV3 := envoy_resources_v3.Mappings()
	for rType := range rTypesV3 {
		if !envoy_resources.ResourcesEqual(x.GetResources(rType), y.GetResources(rType)) {
			return false
		}
		if x.GetVersion(rType) != y.GetVersion(rType) {
			return false
		}
	}
	return true
}
