package stats

import (
	"github.com/prometheus/client_golang/prometheus"
)

// ensure Stats implements the prometheus Collector interface
var _ prometheus.Collector = Stats{}

// Descriptors used to create the metrics
var (
	requestCountDesc = prometheus.NewDesc(
		"marin3r_xdss_discovery_requests_total",
		"Number of discovery requests",
		[]string{"node_id", "resource_type", "pod_name"}, nil,
	)
	ackCountDesc = prometheus.NewDesc(
		"marin3r_xdss_discovery_ack_total",
		"Number of discovery ACK responses",
		[]string{"node_id", "resource_type", "pod_name"}, nil,
	)
	nackCountDesc = prometheus.NewDesc(
		"marin3r_xdss_discovery_nack_total",
		"Number of discovery NACK responses",
		[]string{"node_id", "resource_type", "pod_name"}, nil,
	)
	infoDesc = prometheus.NewDesc(
		"marin3r_xdss_discovery_info",
		"Information about the version a certain resource type is at",
		[]string{"node_id", "resource_type", "pod_name", "version"}, nil,
	)
)

// Describe is implemented with DescribeByCollect. That's possible because the
// Collect method will always return the same 4 metrics with the same 4
// descriptors.
func (xmc Stats) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(xmc, ch)
}

// Collect dumps all the keys in the stats cache. Then it
// creates constant metrics for each modeID/resourceType/pod on the fly based on the
// dumped returned data.
func (s Stats) Collect(ch chan<- prometheus.Metric) {

	items := s.DumpAll()
	for k, v := range items {
		key := NewKeyFromString(k)

		switch metric := key.StatName + "/" + key.Version; metric {

		case "request_counter/*":
			ch <- prometheus.MustNewConstMetric(
				requestCountDesc,
				prometheus.CounterValue,
				float64(v.Object.(int64)),
				key.NodeID, key.ResourceType, key.PodID,
			)

		case "ack_counter/*":
			ch <- prometheus.MustNewConstMetric(
				ackCountDesc,
				prometheus.CounterValue,
				float64(v.Object.(int64)),
				key.NodeID, key.ResourceType, key.PodID,
			)

		case "nack_counter/*":
			ch <- prometheus.MustNewConstMetric(
				nackCountDesc,
				prometheus.CounterValue,
				float64(v.Object.(int64)),
				key.NodeID, key.ResourceType, key.PodID,
			)

		}

	}

	// expose info metrics
	type currentVersion struct {
		version string
		ts      int64
	}
	info := map[string]currentVersion{}
	for k, v := range items {
		key := NewKeyFromString(k)
		if key.StatName == "info" {
			version := key.Version
			key.Version = "*"
			if cv, ok := info[key.String()]; !ok || v.Object.(int64) > cv.ts {
				info[key.String()] = currentVersion{version, v.Object.(int64)}
			}
		}
	}

	for k, v := range info {
		key := NewKeyFromString(k)
		ch <- prometheus.MustNewConstMetric(
			infoDesc,
			prometheus.UntypedValue,
			float64(0),
			key.NodeID, key.ResourceType, key.PodID, v.version,
		)
	}

}
