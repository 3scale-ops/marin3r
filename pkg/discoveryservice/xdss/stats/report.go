package stats

import "fmt"

func (s *Stats) WriteResponseNonce(nodeID, rType, version, podID, nonce string) {
	s.SetString(nodeID, rType, version, podID, fmt.Sprintf("nonce:%s", nonce), "")
}

func (s *Stats) ReportNACK(nodeID, rType, podID, nonce string) error {
	keys := s.FilterKeys(nodeID, rType, podID, fmt.Sprintf("nonce:%s", nonce))
	if len(keys) != 1 {
		return fmt.Errorf("error reporting failure: unexpected number of nonces in the cache")
	}

	// The value fo version is container in the key of the corresponding nonce stored
	// in the cache
	version := NewKeyFromString(func() string {
		for k := range keys {
			return k
		}
		return ""
	}()).Version

	s.IncrementCounter(nodeID, rType, version, podID, "nack_counter", 1)
	return nil
}

func (s *Stats) ReportACK(nodeID, rType, version, podID string) {
	s.IncrementCounter(nodeID, rType, version, podID, "ack_counter", 1)
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
