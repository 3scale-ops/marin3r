package stats

import (
	"reflect"
	"testing"

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
		want       map[string]kv.Item
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
			want: map[string]kv.Item{
				"node:endpoint:aaaa:pod-xxxx:nonce:7": {Object: "", Expiration: int64(defaultExpiration)}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Stats{store: kv.NewFrom(defaultExpiration, cleanupInterval, tt.cacheItems)}
			s.WriteResponseNonce(tt.args.nodeID, tt.args.rType, tt.args.version, tt.args.podID, tt.args.nonce)
			if got := s.store.Items(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Stats.WriteResponseNonce() = %v, want %v", got, tt.want)
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
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Stats{store: kv.NewFrom(defaultExpiration, cleanupInterval, tt.cacheItems)}
			if err := s.ReportNACK(tt.args.nodeID, tt.args.rType, tt.args.podID, tt.args.nonce); (err != nil) != tt.wantErr {
				t.Errorf("Stats.ReportNACK() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got := s.store.Items(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Stats.ReportNACK() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStats_ReportACK(t *testing.T) {
	type args struct {
		nodeID  string
		podID   string
		version string
		rType   string
	}
	tests := []struct {
		name       string
		cacheItems map[string]kv.Item
		args       args
		want       map[string]kv.Item
	}{
		{
			name: "Increments a NACK counter",
			cacheItems: map[string]kv.Item{
				"node:endpoint:aaaa:pod-xxxx:ack_counter": {Object: int64(5), Expiration: int64(defaultExpiration)},
			},
			args: args{
				nodeID: "node",
				rType:  "endpoint",
				podID:  "pod-xxxx",
			},
			want: map[string]kv.Item{
				"node:endpoint:aaaa:pod-xxxx:ack_counter": {Object: int64(6), Expiration: int64(defaultExpiration)},
			},
		},
		{
			name:       "Creates a NACK counter",
			cacheItems: map[string]kv.Item{},
			args: args{
				nodeID: "node",
				rType:  "endpoint",
				podID:  "pod-xxxx",
			},
			want: map[string]kv.Item{
				"node:endpoint:aaaa:pod-xxxx:ack_counter": {Object: int64(1), Expiration: int64(defaultExpiration)},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Stats{store: kv.NewFrom(defaultExpiration, cleanupInterval, tt.cacheItems)}
			s.ReportACK(tt.args.nodeID, tt.args.podID, tt.args.version, tt.args.rType)
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
