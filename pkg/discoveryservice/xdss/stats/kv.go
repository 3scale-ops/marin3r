package stats

import (
	"fmt"
	"strings"
	"time"

	kv "github.com/patrickmn/go-cache"
)

const (
	defaultExpiration = 0
	cleanupInterval   = 300 * time.Second
)

type Key struct {
	NodeID       string
	ResourceType string
	Version      string
	PodID        string
	StatName     string
}

func NewKey(nodeID, rType, version, podID, statName string) *Key {
	return &Key{
		NodeID:       nodeID,
		ResourceType: rType,
		Version:      version,
		PodID:        podID,
		StatName:     statName,
	}
}

func NewKeyFromString(key string) *Key {
	values := strings.Split(key, ":")
	return &Key{
		NodeID:       values[0],
		ResourceType: values[1],
		Version:      values[2],
		PodID:        values[3],
		StatName:     strings.Join(values[4:], ":"),
	}
}

func (k *Key) String() string {
	return strings.Join([]string{k.NodeID, k.ResourceType, k.Version, k.PodID, k.StatName}, ":")
}

func (s *Stats) GetString(nodeID, rtype, version, podID, statName string) (string, error) {
	k := NewKey(nodeID, rtype, version, podID, statName).String()
	if v, ok := s.store.Get(k); ok {
		if value, ok := v.(string); !ok {
			return "", fmt.Errorf("value of key '%s' is not a string", k)
		} else {
			return value, nil
		}

	} else {
		return "", fmt.Errorf("key %s not found", k)
	}
}

func (s *Stats) SetString(nodeID, rType, version, podID, statName, value string) {
	s.store.SetDefault(NewKey(nodeID, rType, version, podID, statName).String(), value)
}

func (s *Stats) SetStringWithExpiration(nodeID, rType, version, podID, statName, value string, expiration time.Duration) {
	s.store.Set(NewKey(nodeID, rType, version, podID, statName).String(), value, expiration)
}

func (s *Stats) SetInt64(nodeID, rType, version, podID, statName string, value int64) {
	s.store.SetDefault(NewKey(nodeID, rType, version, podID, statName).String(), value)
}

func (s *Stats) GetCounter(nodeID, rtype, version, podID, statName string) (int64, error) {
	k := NewKey(nodeID, rtype, version, podID, statName).String()
	if v, ok := s.store.Get(k); ok {
		if value, ok := v.(int64); !ok {
			return 0, fmt.Errorf("value of key '%s' is not an int", k)
		} else {
			return value, nil
		}

	} else {
		return 0, fmt.Errorf("key %s not found", k)
	}
}

// IncrementCounter increments the counter if it already exists or creates it if it doesn't. IncrementCount
// removes any expiration that the cache item might had previously.
func (s *Stats) IncrementCounter(nodeID, rType, version, podID, statName string, increment int64) {
	// GetCounter returns 0 when an error happens so we don't need to check for errors
	counter, _ := s.GetCounter(nodeID, rType, version, podID, statName)
	s.store.SetDefault(NewKey(nodeID, rType, version, podID, statName).String(), counter+increment)
}

// ExpireCounter adds expiration to a counter.
func (s *Stats) ExpireCounter(nodeID, rType, version, podID, statName string, expiration time.Duration) {
	counter, _ := s.GetCounter(nodeID, rType, version, podID, statName)
	s.store.Set(NewKey(nodeID, rType, version, podID, statName).String(), counter, expiration)
}

func (s *Stats) FilterKeys(filters ...string) map[string]kv.Item {
	all := s.store.Items()
	selected := map[string]kv.Item{}
	var isSelected bool
	for key, value := range all {
		isSelected = true
		for _, filter := range filters {
			if !strings.Contains(key, filter) {
				isSelected = false
			}
		}
		if isSelected {
			selected[key] = value
		}
	}
	return selected
}

func (s *Stats) DeleteKeysByFilter(filters ...string) {
	keys := s.FilterKeys(filters...)
	for k := range keys {
		s.store.Delete(k)
	}
}

func (s *Stats) DumpAll() map[string]kv.Item {
	return s.store.Items()
}
