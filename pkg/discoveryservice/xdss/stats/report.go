package stats

import (
	"fmt"
	"time"
)

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
}

func (s *Stats) ReportRequest(nodeID, rType, podID string, streamID int64) {
	s.IncrementCounter(nodeID, rType, "*", podID, fmt.Sprintf("request_counter:stream_%d", streamID), 1)
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
