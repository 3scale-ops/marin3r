package stats

import (
	"reflect"
	"sort"
	"testing"

	kv "github.com/patrickmn/go-cache"
)

func TestStats_GetSubscribedPods(t *testing.T) {
	type args struct {
		nodeID string
		rType  string
	}
	tests := []struct {
		name       string
		cacheItems map[string]kv.Item
		args       args
		want       []string
	}{
		{
			name: "",
			cacheItems: map[string]kv.Item{
				"node:endpoint:*:pod-xxxx:request_counter:stream_1": {Object: int64(1), Expiration: int64(defaultExpiration)},
				"node:cluster:*:pod-xxxx:request_counter:stream_2":  {Object: int64(1), Expiration: int64(defaultExpiration)},
				"node:cluster:*:pod-yyyy:request_counter":           {Object: int64(1), Expiration: int64(defaultExpiration)},
			},
			args: args{
				nodeID: "node",
				rType:  "cluster",
			},
			want: []string{"pod-xxxx", "pod-yyyy"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Stats{store: kv.NewFrom(defaultExpiration, cleanupInterval, tt.cacheItems)}
			got := s.GetSubscribedPods(tt.args.nodeID, tt.args.rType)
			sort.Strings(got)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Stats.GetSubscribedPods() = %v, want %v", sort.StringSlice(got), sort.StringSlice(tt.want))
			}
		})
	}
}

func TestStats_GetPercentageFailing(t *testing.T) {
	type args struct {
		nodeID  string
		rType   string
		version string
	}
	tests := []struct {
		name       string
		cacheItems map[string]kv.Item
		args       args
		want       float64
	}{
		{
			name: "Returns 50%",
			cacheItems: map[string]kv.Item{
				"node:endpoint:*:pod-aaaa:request_counter:stream_1": {Object: int64(2), Expiration: int64(defaultExpiration)},
				"node:endpoint:*:pod-bbbb:request_counter:stream_2": {Object: int64(5), Expiration: int64(defaultExpiration)},
				"node:endpoint:*:pod-cccc:request_counter:stream_3": {Object: int64(1), Expiration: int64(defaultExpiration)},
				"node:endpoint:*:pod-dddd:request_counter:stream_4": {Object: int64(1), Expiration: int64(defaultExpiration)},
				"node:endpoint:xxxx:pod-aaaa:nack_counter":          {Object: int64(1), Expiration: int64(defaultExpiration)},
				"node:endpoint:xxxx:pod-bbbb:nack_counter":          {Object: int64(10), Expiration: int64(defaultExpiration)},
			},
			args: args{
				nodeID:  "node",
				rType:   "endpoint",
				version: "xxxx",
			},
			want: 0.5,
		},
		{
			name: "Returns 100%",
			cacheItems: map[string]kv.Item{
				"node:endpoint:*:pod-aaaa:request_counter:stream_1": {Object: int64(2), Expiration: int64(defaultExpiration)},
				"node:endpoint:*:pod-bbbb:request_counter:stream_2": {Object: int64(5), Expiration: int64(defaultExpiration)},
				"node:endpoint:*:pod-cccc:request_counter:stream_3": {Object: int64(1), Expiration: int64(defaultExpiration)},
				"node:endpoint:*:pod-dddd:request_counter:stream_4": {Object: int64(1), Expiration: int64(defaultExpiration)},
				"node:endpoint:xxxx:pod-aaaa:nack_counter":          {Object: int64(1), Expiration: int64(defaultExpiration)},
				"node:endpoint:xxxx:pod-bbbb:nack_counter":          {Object: int64(10), Expiration: int64(defaultExpiration)},
				"node:endpoint:xxxx:pod-cccc:nack_counter":          {Object: int64(10), Expiration: int64(defaultExpiration)},
				"node:endpoint:xxxx:pod-dddd:nack_counter":          {Object: int64(10), Expiration: int64(defaultExpiration)},
			},
			args: args{
				nodeID:  "node",
				rType:   "endpoint",
				version: "xxxx",
			},
			want: 1,
		},
		{
			name: "Returns 0%",
			cacheItems: map[string]kv.Item{
				"node:endpoint:*:pod-aaaa:request_counter:stream_1": {Object: int64(2), Expiration: int64(defaultExpiration)},
				"node:endpoint:*:pod-bbbb:request_counter:stream_2": {Object: int64(5), Expiration: int64(defaultExpiration)},
			},
			args: args{
				nodeID:  "node",
				rType:   "endpoint",
				version: "xxxx",
			},
			want: 0,
		},
		{
			name:       "Returns 0%",
			cacheItems: map[string]kv.Item{},
			args: args{
				nodeID:  "node",
				rType:   "endpoint",
				version: "xxxx",
			},
			want: 0,
		},
		{
			name: "Returns 0% if NaN",
			cacheItems: map[string]kv.Item{
				"node:endpoint:xxxx:pod-aaaa:nack_counter": {Object: int64(1), Expiration: int64(defaultExpiration)},
			},
			args: args{
				nodeID:  "node",
				rType:   "endpoint",
				version: "xxxx",
			},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Stats{store: kv.NewFrom(defaultExpiration, cleanupInterval, tt.cacheItems)}
			if got := s.GetPercentageFailing(tt.args.nodeID, tt.args.rType, tt.args.version); got != tt.want {
				t.Errorf("Stats.GetPercentageFailing() = %v, want %v", got, tt.want)
			}
		})
	}
}
