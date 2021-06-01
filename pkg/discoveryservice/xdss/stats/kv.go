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
	Key          string
}

func NewKey(nodeID, rType, version, podID, key string) *Key {
	return &Key{
		NodeID:       nodeID,
		ResourceType: rType,
		Version:      version,
		PodID:        podID,
		Key:          key,
	}
}

func NewKeyFromString(key string) *Key {
	values := strings.Split(key, ":")
	return &Key{
		NodeID:       values[0],
		ResourceType: values[1],
		Version:      values[2],
		PodID:        values[3],
		Key:          values[4],
	}
}

func (k *Key) String() string {
	return strings.Join([]string{k.NodeID, k.ResourceType, k.Version, k.PodID, k.Key}, ":")
}

func (s *Stats) GetString(nodeID, rtype, version, podID, key string) (string, error) {
	k := NewKey(nodeID, rtype, version, podID, key).String()
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

func (s *Stats) SetString(nodeID, rType, version, podID, key, value string) {
	s.store.SetDefault(NewKey(nodeID, rType, version, podID, key).String(), value)
}

func (s *Stats) SetStringWithExpiration(nodeID, rType, version, podID, key, value string, expiration time.Duration) {
	s.store.Set(NewKey(nodeID, rType, version, podID, key).String(), value, expiration)
}

func (s *Stats) GetCounter(nodeID, rtype, version, podID, key string) (int64, error) {
	k := NewKey(nodeID, rtype, version, podID, key).String()
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

func (s *Stats) IncrementCounter(nodeID, rType, version, podID, key string, increment int64) {
	if _, err := s.store.IncrementInt64(NewKey(nodeID, rType, version, podID, key).String(), increment); err != nil {
		// The key does not exist yet in the kv store
		s.store.SetDefault(NewKey(nodeID, rType, version, podID, key).String(), increment)
	}
}

func (s *Stats) DecrementCounter(nodeID, rType, version, podID, key string, decrement int64) {
	if _, err := s.store.DecrementInt64(NewKey(nodeID, rType, version, podID, key).String(), decrement); err != nil {
		// The key does not exist yet in the kv store
		s.store.SetDefault(NewKey(nodeID, rType, version, podID, key).String(), 0)
	}
}

func (s *Stats) FilterKeys(filters ...string) map[string]kv.Item {
	all := s.store.Items()
	selected := map[string]kv.Item{}
	for key, value := range all {
		isSelected := true
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
