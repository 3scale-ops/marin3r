package stats

import "math"

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
