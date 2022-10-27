package stats

import (
	"fmt"
	"math"
	"time"

	"github.com/3scale-ops/marin3r/pkg/util/clock"
	kv "github.com/patrickmn/go-cache"
)

// The format of the keys in the cache is
//   <node-id>:<version>:<resource-type>:<pod-id>:<stat-name>
type Stats struct {
	store *kv.Cache
	clock clock.Clock
}

func New() *Stats {
	return &Stats{
		store: kv.New(defaultExpiration, cleanupInterval),
		clock: clock.Real{},
	}
}

func NewWithItems(items map[string]kv.Item, now time.Time) *Stats {
	return &Stats{
		store: kv.NewFrom(defaultExpiration, cleanupInterval, items),
		clock: clock.NewTest(now),
	}
}

func (s *Stats) WriteResponseNonce(nodeID, rType, version, podID, nonce string) {
	s.SetStringWithExpiration(nodeID, rType, version, podID, "nonce:"+nonce, "", 10*time.Second)
}

func (s *Stats) ReportNACK(nodeID, rType, podID, nonce string) (int64, error) {
	keys := s.FilterKeys(nodeID, rType, podID, "nonce:"+nonce)
	if len(keys) != 1 {
		return 0, fmt.Errorf("error reporting failure: unexpected number of nonces in the cache")
	}

	// The value of version is contained in the key of the corresponding nonce stored
	// in the cache
	version := NewKeyFromString(func() string {
		for k := range keys {
			return k
		}
		return ""
	}()).Version

	s.IncrementCounter(nodeID, rType, version, podID, "nack_counter", 1)
	// aggregated counter, with lower cardinality, to expose as prometheus metric
	s.IncrementCounter(nodeID, rType, "*", podID, "nack_counter", 1)
	return s.GetCounter(nodeID, rType, version, podID, "nack_counter")
}

func (s *Stats) ReportACK(nodeID, rType, version, podID string) {
	s.IncrementCounter(nodeID, rType, version, podID, "ack_counter", 1)
	// aggregated counter, with lower cardinality, to expose as prometheus metric
	s.IncrementCounter(nodeID, rType, "*", podID, "ack_counter", 1)
	// add stat with timestamp to expose info metric
	s.SetInt64(nodeID, rType, version, podID, "info", s.clock.Now().UnixMilli())
}

func (s *Stats) ReportRequest(nodeID, rType, podID string) {
	s.IncrementCounter(nodeID, rType, "*", podID, "request_counter", 1)
}

func GetStringValueFromMetadata(meta map[string]interface{}, key string) (string, error) {

	v, ok := meta[key]
	if !ok {
		return "", fmt.Errorf("missing 'pod_name' in node's metadata")
	} else if _, ok := v.(string); !ok {
		return "", fmt.Errorf("metadata value is not a string")
	}

	return v.(string), nil
}

func (s *Stats) GetSubscribedPods(nodeID, rType string) map[string]int8 {

	m := map[string]int8{}

	filters := []string{"request_counter"}
	if nodeID != "" {
		filters = append(filters, nodeID)
	}
	if rType != "" {
		filters = append(filters, rType)
	}
	items := s.FilterKeys(filters...)

	for k := range items {
		podID := NewKeyFromString(k).PodID
		if _, ok := m[podID]; !ok {
			m[podID] = 1
		}
	}

	return m
}

func (s *Stats) GetPercentageFailing(nodeID, rType, version string) float64 {

	failing := 0
	pods := s.GetSubscribedPods(nodeID, rType)
	for pod := range pods {
		if v, err := s.GetCounter(nodeID, rType, version, pod, "nack_counter"); err == nil && v > 0 {
			failing++
		}
	}

	val := float64(failing) / float64(len(pods))
	if math.IsNaN(val) {
		return 0
	}
	return val
}
