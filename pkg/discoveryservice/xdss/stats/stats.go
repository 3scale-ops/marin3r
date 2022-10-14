package stats

import (
	"fmt"
	"math"
	"time"

	kv "github.com/patrickmn/go-cache"
)

// The format of the keys in the cache is
//   <node-id>:<version>:<resource-type>:<pod-id>:<key>
//
// Note that though currently revision and version have the same value for all
// types (with the exeption of secrets), this might change in the future and have
// each resource type follow its own versioning
type Stats struct {
	store *kv.Cache
}

func New() *Stats {
	return &Stats{
		store: kv.New(defaultExpiration, cleanupInterval),
	}
}

func NewWithItems(items map[string]kv.Item) *Stats {
	return &Stats{
		store: kv.NewFrom(defaultExpiration, cleanupInterval, items),
	}
}

func (s *Stats) WriteResponseNonce(nodeID, rType, version, podID, nonce string) {
	s.SetStringWithExpiration(nodeID, rType, version, podID, fmt.Sprintf("nonce:%s", nonce), "", 10*time.Second)
}

func (s *Stats) ReportNACK(nodeID, rType, podID, nonce string) (int64, error) {
	keys := s.FilterKeys(nodeID, rType, podID, fmt.Sprintf("nonce:%s", nonce))
	if len(keys) != 1 {
		return 0, fmt.Errorf("error reporting failure: unexpected number of nonces in the cache")
	}

	// The value of version is container in the key of the corresponding nonce stored
	// in the cache
	version := NewKeyFromString(func() string {
		for k := range keys {
			return k
		}
		return ""
	}()).Version

	s.IncrementCounter(nodeID, rType, version, podID, "nack_counter", 1)
	return s.GetCounter(nodeID, rType, version, podID, "nack_counter")
}

func (s *Stats) ReportACK(nodeID, rType, version, podID string) {
	s.IncrementCounter(nodeID, rType, version, podID, "ack_counter", 1)
	// aggregated counter, with lower cardinality, to expose as prometheus metric
	s.IncrementCounter(nodeID, rType, "*", podID, "ack_counter", 1)
}

func (s *Stats) ReportRequest(nodeID, rType, podID string, streamID int64) {
	s.IncrementCounter(nodeID, rType, "*", podID, fmt.Sprintf("request_counter:stream_%d", streamID), 1)
	// aggregated counter ,with lower cardinality, to expose as prometheus metric
	s.IncrementCounter(nodeID, rType, "*", podID, "*", 1)
}

func (s *Stats) ReportStreamClosed(streamID int64) {
	s.DeleteKeysByFilter(fmt.Sprintf("request_counter:stream_%d", streamID))
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

func (s *Stats) GetSubscribedPods(nodeID, rType string) []string {

	m := map[string]int8{}
	items := s.FilterKeys(nodeID, rType, "request_counter")

	for k := range items {
		podID := NewKeyFromString(k).PodID
		if _, ok := m[podID]; !ok {
			m[podID] = 0
		}
	}

	pods := make([]string, len(m))
	i := 0
	for k := range m {
		pods[i] = k
		i++
	}

	return pods
}

func (s *Stats) GetPercentageFailing(nodeID, rType, version string) float64 {

	failing := 0
	pods := s.GetSubscribedPods(nodeID, rType)
	for _, pod := range pods {
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
