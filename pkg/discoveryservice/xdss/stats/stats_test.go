package stats

import (
	"reflect"
	"testing"
	"time"

	kv "github.com/patrickmn/go-cache"
)

func TestStats_WriteResponseNonce(t *testing.T) {
	type args struct {
		nodeID  string
		version string
		podID   string
		rType   string
		nonce   string
	}
	tests := []struct {
		name       string
		cacheItems map[string]kv.Item
		args       args
		wantKey    string
	}{
		{
			name:       "Writes the nonce",
			cacheItems: map[string]kv.Item{},
			args: args{
				nodeID:  "node",
				rType:   "endpoint",
				version: "aaaa",
				podID:   "pod-xxxx",
				nonce:   "7",
			},
			wantKey: "node:endpoint:aaaa:pod-xxxx:nonce:7",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Stats{store: kv.NewFrom(defaultExpiration, cleanupInterval, tt.cacheItems)}
			s.WriteResponseNonce(tt.args.nodeID, tt.args.rType, tt.args.version, tt.args.podID, tt.args.nonce)
			if _, ok := s.store.Get(tt.wantKey); !ok {
				t.Errorf("Stats.WriteResponseNonce() = key not found")
			}
		})
	}
}

func TestStats_ReportNACK(t *testing.T) {
	type args struct {
		nodeID string
		rType  string
		podID  string
		nonce  string
	}
	tests := []struct {
		name       string
		cacheItems map[string]kv.Item
		args       args
		want       map[string]kv.Item
		wantErr    bool
	}{
		{
			name: "Increments a NACK counter",
			cacheItems: map[string]kv.Item{
				"node:endpoint:aaaa:pod-xxxx:nonce:7":      {Object: "", Expiration: int64(defaultExpiration)},
				"node:endpoint:aaaa:pod-xxxx:nack_counter": {Object: int64(5), Expiration: int64(defaultExpiration)},
				"node:endpoint:*:pod-xxxx:nack_counter":    {Object: int64(5), Expiration: int64(defaultExpiration)},
			},
			args: args{
				nodeID: "node",
				rType:  "endpoint",
				podID:  "pod-xxxx",
				nonce:  "7",
			},
			want: map[string]kv.Item{
				"node:endpoint:aaaa:pod-xxxx:nonce:7":      {Object: "", Expiration: int64(defaultExpiration)},
				"node:endpoint:aaaa:pod-xxxx:nack_counter": {Object: int64(6), Expiration: int64(defaultExpiration)},
				"node:endpoint:*:pod-xxxx:nack_counter":    {Object: int64(6), Expiration: int64(defaultExpiration)},
			},
		},
		{
			name: "Creates a new NACK counter",
			cacheItems: map[string]kv.Item{
				"node:endpoint:aaaa:pod-xxxx:nonce:xyz": {Object: "", Expiration: int64(defaultExpiration)},
			},
			args: args{
				nodeID: "node",
				rType:  "endpoint",
				podID:  "pod-xxxx",
				nonce:  "xyz",
			},
			want: map[string]kv.Item{
				"node:endpoint:aaaa:pod-xxxx:nonce:xyz":    {Object: "", Expiration: int64(defaultExpiration)},
				"node:endpoint:aaaa:pod-xxxx:nack_counter": {Object: int64(1), Expiration: int64(defaultExpiration)},
				"node:endpoint:*:pod-xxxx:nack_counter":    {Object: int64(1), Expiration: int64(defaultExpiration)},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Stats{store: kv.NewFrom(defaultExpiration, cleanupInterval, tt.cacheItems)}
			_, err := s.ReportNACK(tt.args.nodeID, tt.args.rType, tt.args.podID, tt.args.nonce)
			if (err != nil) != tt.wantErr {
				t.Errorf("Stats.ReportNACK() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got := s.store.Items(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Stats.ReportACK() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStats_ReportACK(t *testing.T) {
	type args struct {
		nodeID  string
		rType   string
		version string
		podID   string
	}
	tests := []struct {
		name       string
		cacheItems map[string]kv.Item
		args       args
		want       map[string]kv.Item
	}{
		{
			name: "Increments an ACK counter",
			cacheItems: map[string]kv.Item{
				"node:endpoint:aaaa:pod-xxxx:ack_counter": {Object: int64(5), Expiration: int64(defaultExpiration)},
				"node:endpoint:bbbb:pod-xxxx:ack_counter": {Object: int64(3), Expiration: int64(defaultExpiration)},
				"node:endpoint:*:pod-xxxx:ack_counter":    {Object: int64(8), Expiration: int64(defaultExpiration)},
			},
			args: args{
				nodeID:  "node",
				rType:   "endpoint",
				version: "aaaa",
				podID:   "pod-xxxx",
			},
			want: map[string]kv.Item{
				"node:endpoint:aaaa:pod-xxxx:ack_counter": {Object: int64(6), Expiration: int64(defaultExpiration)},
				"node:endpoint:bbbb:pod-xxxx:ack_counter": {Object: int64(3), Expiration: int64(defaultExpiration)},
				"node:endpoint:*:pod-xxxx:ack_counter":    {Object: int64(9), Expiration: int64(defaultExpiration)},
			},
		},
		{
			name:       "Creates an ACK counter",
			cacheItems: map[string]kv.Item{},
			args: args{
				nodeID:  "node",
				rType:   "endpoint",
				version: "aaaa",
				podID:   "pod-xxxx",
			},
			want: map[string]kv.Item{
				"node:endpoint:aaaa:pod-xxxx:ack_counter": {Object: int64(1), Expiration: int64(defaultExpiration)},
				"node:endpoint:*:pod-xxxx:ack_counter":    {Object: int64(1), Expiration: int64(defaultExpiration)},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Stats{store: kv.NewFrom(defaultExpiration, cleanupInterval, tt.cacheItems)}
			s.ReportACK(tt.args.nodeID, tt.args.rType, tt.args.version, tt.args.podID)
			if got := s.store.Items(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Stats.ReportACK() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStats_ReportRequest(t *testing.T) {
	type args struct {
		nodeID   string
		rType    string
		podID    string
		streamID int64
	}
	tests := []struct {
		name       string
		cacheItems map[string]kv.Item
		args       args
		want       map[string]kv.Item
	}{
		{
			name: "Increases counter",
			cacheItems: map[string]kv.Item{
				"node:endpoint:*:pod-xxxx:request_counter:stream_10": {Object: int64(3), Expiration: int64(defaultExpiration)},
				"node:endpoint:*:pod-xxxx:request_counter:stream_9":  {Object: int64(20), Expiration: int64(defaultExpiration)},
				"node:endpoint:*:pod-xxxx:request_counter":           {Object: int64(23), Expiration: int64(defaultExpiration)},
			},
			args: args{
				nodeID:   "node",
				rType:    "endpoint",
				podID:    "pod-xxxx",
				streamID: 10,
			},
			want: map[string]kv.Item{
				"node:endpoint:*:pod-xxxx:request_counter:stream_10": {Object: int64(4), Expiration: int64(defaultExpiration)},
				"node:endpoint:*:pod-xxxx:request_counter:stream_9":  {Object: int64(20), Expiration: int64(defaultExpiration)},
				"node:endpoint:*:pod-xxxx:request_counter":           {Object: int64(24), Expiration: int64(defaultExpiration)},
			},
		},
		{
			name: "Creates new counter",
			cacheItems: map[string]kv.Item{
				"node:endpoint:*:pod-xxxx:request_counter:stream_10": {Object: int64(3), Expiration: int64(defaultExpiration)},
				"node:endpoint:*:pod-xxxx:request_counter":           {Object: int64(3), Expiration: int64(defaultExpiration)},
			},
			args: args{
				nodeID:   "node",
				rType:    "endpoint",
				podID:    "pod-xxxx",
				streamID: 11,
			},
			want: map[string]kv.Item{
				"node:endpoint:*:pod-xxxx:request_counter:stream_10": {Object: int64(3), Expiration: int64(defaultExpiration)},
				"node:endpoint:*:pod-xxxx:request_counter:stream_11": {Object: int64(1), Expiration: int64(defaultExpiration)},
				"node:endpoint:*:pod-xxxx:request_counter":           {Object: int64(4), Expiration: int64(defaultExpiration)},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Stats{store: kv.NewFrom(defaultExpiration, cleanupInterval, tt.cacheItems)}
			s.ReportRequest(tt.args.nodeID, tt.args.rType, tt.args.podID, tt.args.streamID)
			if got := s.store.Items(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Stats.ReportRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStats_ReportStreamClosed(t *testing.T) {
	type args struct {
		streamID int64
	}
	tests := []struct {
		name       string
		cacheItems map[string]kv.Item
		args       args
		want       map[string]kv.Item
	}{
		{
			name: "Deletes all request counters associates with the stream id",
			cacheItems: map[string]kv.Item{
				"node:endpoint:*:pod-xxxx:request_counter:stream_10": {Object: int64(3), Expiration: int64(defaultExpiration)},
				"node:endpoint:*:pod-xxxx:request_counter:stream_11": {Object: int64(1), Expiration: int64(defaultExpiration)},
				"node:cluster:*:pod-xxxx:request_counter:stream_11":  {Object: int64(1), Expiration: int64(defaultExpiration)},
			},
			args: args{
				streamID: 11,
			},
			want: map[string]kv.Item{
				"node:endpoint:*:pod-xxxx:request_counter:stream_10": {Object: int64(3), Expiration: int64(defaultExpiration)},
			},
		},
		{
			name: "Does nothing",
			cacheItems: map[string]kv.Item{
				"node:endpoint:*:pod-xxxx:request_counter:stream_10": {Object: int64(3), Expiration: int64(defaultExpiration)},
			},
			args: args{
				streamID: 11,
			},
			want: map[string]kv.Item{
				"node:endpoint:*:pod-xxxx:request_counter:stream_10": {Object: int64(3), Expiration: int64(defaultExpiration)},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Stats{store: kv.NewFrom(defaultExpiration, cleanupInterval, tt.cacheItems)}
			s.ReportStreamClosed(tt.args.streamID)
			if got := s.store.Items(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Stats.ReportRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetStringValueFromMetadata(t *testing.T) {
	type args struct {
		meta map[string]interface{}
		key  string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "Returns the metadata value as a string",
			args: args{
				meta: map[string]interface{}{
					"meta_key": func() interface{} { s := "name"; return s }(),
				},
				key: "meta_key",
			},
			want:    "name",
			wantErr: false,
		},
		{
			name: "Not a string value",
			args: args{
				meta: map[string]interface{}{
					"meta_key": func() interface{} { s := 3; return s }(),
				},
				key: "meta_key",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Not found",
			args: args{
				meta: map[string]interface{}{
					"meta_key": func() interface{} { s := "name"; return s }(),
				},
				key: "other_key",
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetStringValueFromMetadata(tt.args.meta, tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetStringValueFromMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetStringValueFromMetadata() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStats_GetSubscribedPods(t *testing.T) {
	type args struct {
		nodeID string
		rType  string
	}
	tests := []struct {
		name       string
		cacheItems map[string]kv.Item
		args       args
		want       map[string]int8
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
			want: map[string]int8{"pod-xxxx": 1, "pod-yyyy": 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Stats{store: kv.NewFrom(defaultExpiration, cleanupInterval, tt.cacheItems)}
			got := s.GetSubscribedPods(tt.args.nodeID, tt.args.rType)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Stats.GetSubscribedPods() = %v, want %v", got, tt.want)
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

func TestStats_CleanStats(t *testing.T) {
	type args struct {
		podName string
	}
	tests := []struct {
		name       string
		cacheItems map[string]kv.Item
		args       args
		want       map[string]kv.Item
		// name   string
		// fields fields
		// args   args
	}{
		{
			name: "cleans stat keys for the given pod name",
			cacheItems: map[string]kv.Item{
				"node:endpoint:*:pod-xxxx:request_counter:stream_10": {Object: int64(3), Expiration: int64(defaultExpiration)},
				"node:endpoint:*:pod-xxxx:request_counter:stream_9":  {Object: int64(20), Expiration: int64(defaultExpiration)},
				"node:endpoint:*:pod-xxxx:request_counter":           {Object: int64(23), Expiration: int64(defaultExpiration)},
				"node:endpoint:*:pod-zzzz:request_counter:stream_3":  {Object: int64(2), Expiration: int64(defaultExpiration)},
			},
			args: args{
				podName: "pod-xxxx",
			},
			want: map[string]kv.Item{
				"node:endpoint:*:pod-xxxx:request_counter:stream_10": {Object: int64(3), Expiration: int64(time.Now().Add(5 * time.Minute).UnixNano())},
				"node:endpoint:*:pod-xxxx:request_counter:stream_9":  {Object: int64(20), Expiration: int64(time.Now().Add(5 * time.Minute).UnixNano())},
				"node:endpoint:*:pod-xxxx:request_counter":           {Object: int64(23), Expiration: int64(time.Now().Add(5 * time.Minute).UnixNano())},
				"node:endpoint:*:pod-zzzz:request_counter:stream_3":  {Object: int64(2), Expiration: int64(defaultExpiration)},
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Stats{store: kv.NewFrom(defaultExpiration, cleanupInterval, tt.cacheItems)}
			s.CleanStats(tt.args.podName)
			expired := s.FilterKeys(tt.args.podName)
			for k, v := range expired {
				if v.Expiration == 0 {
					t.Errorf("Stats.CleanStats() = key %+v has no expiration", k)
				}
			}
		})
	}
}