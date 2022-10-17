package stats

import (
	"reflect"
	"testing"
	"time"

	kv "github.com/patrickmn/go-cache"
)

func TestNewKey(t *testing.T) {
	type args struct {
		nodeID   string
		version  string
		rType    string
		podID    string
		statName string
	}
	tests := []struct {
		name string
		args args
		want *Key
	}{
		{
			name: "Returns a Key struct",
			args: args{
				nodeID:   "node1",
				rType:    "endpoint",
				version:  "aaaa",
				podID:    "pod-xxxx",
				statName: "stat1",
			},
			want: &Key{
				NodeID:       "node1",
				ResourceType: "endpoint",
				Version:      "aaaa",
				PodID:        "pod-xxxx",
				StatName:     "stat1",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewKey(tt.args.nodeID, tt.args.rType, tt.args.version, tt.args.podID, tt.args.statName); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewKeyFromString(t *testing.T) {
	type args struct {
		key string
	}
	tests := []struct {
		name string
		args args
		want *Key
	}{
		{
			name: "Returns a key struct",
			args: args{
				key: "node:endpoint:aaaa:pod-xxxx:something:something_else",
			},
			want: &Key{
				NodeID:       "node",
				ResourceType: "endpoint",
				Version:      "aaaa",
				PodID:        "pod-xxxx",
				StatName:     "something:something_else",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewKeyFromString(tt.args.key); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewKeyFromString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKey_String(t *testing.T) {
	type fields struct {
		NodeID       string
		ResourceType string
		Version      string
		PodID        string
		Key          string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "Returns the string representation of a Key",
			fields: fields{
				NodeID:       "node1",
				ResourceType: "endpoint",
				Version:      "aaaa",
				PodID:        "pod-xxxx",
				Key:          "stat1",
			},
			want: "node1:endpoint:aaaa:pod-xxxx:stat1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &Key{
				NodeID:       tt.fields.NodeID,
				ResourceType: tt.fields.ResourceType,
				Version:      tt.fields.Version,
				PodID:        tt.fields.PodID,
				StatName:     tt.fields.Key,
			}
			if got := k.String(); got != tt.want {
				t.Errorf("Key.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStats_GetString(t *testing.T) {
	type args struct {
		nodeID   string
		version  string
		rType    string
		podID    string
		statName string
	}
	tests := []struct {
		name       string
		cacheItems map[string]kv.Item
		args       args
		want       string
		wantErr    bool
	}{
		{
			name: "Returns a string value",
			cacheItems: map[string]kv.Item{
				"node1:endpoint:aaaa:pod-xxxx:stat1": {Object: "item1", Expiration: int64(defaultExpiration)},
				"node1:endpoint:aaaa:pod-xxxx:stat2": {Object: "item2", Expiration: int64(defaultExpiration)},
				"node1:endpoint:bbbb:pod-xxxx:stat1": {Object: "item3", Expiration: int64(defaultExpiration)},
			},
			args: args{
				nodeID:   "node1",
				rType:    "endpoint",
				version:  "aaaa",
				podID:    "pod-xxxx",
				statName: "stat1",
			},
			want:    "item1",
			wantErr: false,
		},
		{
			name: "Not a string error",
			cacheItems: map[string]kv.Item{
				"node1:endpoint:aaaa:pod-xxxx:stat1": {Object: "item1", Expiration: int64(defaultExpiration)},
				"node1:endpoint:aaaa:pod-xxxx:stat2": {Object: 5, Expiration: int64(defaultExpiration)},
				"node1:endpoint:bbbb:pod-xxxx:stat1": {Object: "item3", Expiration: int64(defaultExpiration)},
			},
			args: args{
				nodeID:   "node1",
				rType:    "endpoint",
				version:  "aaaa",
				podID:    "pod-xxxx",
				statName: "stat2",
			},
			want:    "",
			wantErr: true,
		},
		{
			name:       "Not a found error",
			cacheItems: map[string]kv.Item{},
			args: args{
				nodeID:   "node1",
				rType:    "endpoint",
				version:  "aaaa",
				podID:    "pod-xxxx",
				statName: "stat2",
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Stats{store: kv.NewFrom(defaultExpiration, cleanupInterval, tt.cacheItems)}
			got, err := s.GetString(tt.args.nodeID, tt.args.rType, tt.args.version, tt.args.podID, tt.args.statName)
			if (err != nil) != tt.wantErr {
				t.Errorf("Stats.GetString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Stats.GetString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStats_SetString(t *testing.T) {
	type args struct {
		nodeID   string
		version  string
		rType    string
		podID    string
		statName string
		value    string
	}
	tests := []struct {
		name       string
		cacheItems map[string]kv.Item
		args       args
		want       kv.Item
	}{
		{
			name:       "Writes a string key",
			cacheItems: map[string]kv.Item{},
			args: args{
				nodeID:   "node",
				rType:    "endpoint",
				version:  "aaaa",
				podID:    "pod-xxxx",
				statName: "stat",
				value:    "value",
			},
			want: kv.Item{
				Object:     "value",
				Expiration: 0,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Stats{store: kv.NewFrom(defaultExpiration, cleanupInterval, tt.cacheItems)}
			s.SetString(tt.args.nodeID, tt.args.rType, tt.args.version, tt.args.podID, tt.args.statName, tt.args.value)
			if got, _ := s.store.Get(NewKey(tt.args.nodeID, tt.args.rType, tt.args.version, tt.args.podID, tt.args.statName).String()); got != tt.want.Object {
				t.Errorf("Stats.SetString() = %v, want %v", got, tt.want.Object)
			}
		})
	}
}

func TestStats_SetStringWithExpiration(t *testing.T) {
	type args struct {
		nodeID     string
		rType      string
		version    string
		podID      string
		statName   string
		value      string
		expiration time.Duration
	}
	tests := []struct {
		name       string
		cacheItems map[string]kv.Item
		args       args
		want       kv.Item
	}{
		{
			name:       "Writes a string key with expiration",
			cacheItems: map[string]kv.Item{},
			args: args{
				nodeID:     "node",
				rType:      "endpoint",
				version:    "aaaa",
				podID:      "pod-xxxx",
				statName:   "stat",
				value:      "value",
				expiration: time.Duration(time.Second),
			},
			want: kv.Item{
				Object:     "value",
				Expiration: int64(time.Second),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Stats{store: kv.NewFrom(defaultExpiration, cleanupInterval, tt.cacheItems)}
			s.SetStringWithExpiration(tt.args.nodeID, tt.args.rType, tt.args.version, tt.args.podID, tt.args.statName, tt.args.value, tt.args.expiration)
			if got, _ := s.store.Get(NewKey(tt.args.nodeID, tt.args.rType, tt.args.version, tt.args.podID, tt.args.statName).String()); got != tt.want.Object {
				t.Errorf("Stats.SetStringWithExpiration() = %v, want %v", got, tt.want.Object)
			}
		})
	}
}

func TestStats_FilterKeys(t *testing.T) {
	type args struct {
		filters []string
	}
	tests := []struct {
		name       string
		cacheItems map[string]kv.Item
		args       args
		want       map[string]kv.Item
	}{
		{
			name: "Selects one key that match the filter",
			cacheItems: map[string]kv.Item{
				"key1":        {Object: "item1", Expiration: int64(defaultExpiration)},
				"key2:filter": {Object: "item2", Expiration: int64(defaultExpiration)},
				"key3":        {Object: "item3", Expiration: int64(defaultExpiration)},
			},
			args: args{
				filters: []string{"filter"},
			},
			want: map[string]kv.Item{
				"key2:filter": {Object: "item2", Expiration: int64(defaultExpiration)},
			},
		},
		{
			name: "Selects several keys that match the filter",
			cacheItems: map[string]kv.Item{
				"key1":        {Object: "item1", Expiration: int64(defaultExpiration)},
				"key2:filter": {Object: "item2", Expiration: int64(defaultExpiration)},
				"key3:filter": {Object: "item3", Expiration: int64(defaultExpiration)},
			},
			args: args{
				filters: []string{"filter"},
			},
			want: map[string]kv.Item{
				"key2:filter": {Object: "item2", Expiration: int64(defaultExpiration)},
				"key3:filter": {Object: "item3", Expiration: int64(defaultExpiration)},
			},
		},
		{
			name: "Selects one key that matches several filters",
			cacheItems: map[string]kv.Item{
				"key1":                 {Object: "item1", Expiration: int64(defaultExpiration)},
				"key2:filter1:filter2": {Object: "item2", Expiration: int64(defaultExpiration)},
				"key3:filter1":         {Object: "item3", Expiration: int64(defaultExpiration)},
			},
			args: args{
				filters: []string{"filter1", "filter2"},
			},
			want: map[string]kv.Item{
				"key2:filter1:filter2": {Object: "item2", Expiration: int64(defaultExpiration)},
			},
		},
		{
			name: "Selects several keys that matches several filters",
			cacheItems: map[string]kv.Item{
				"key1":                 {Object: "item1", Expiration: int64(defaultExpiration)},
				"key2:filter1:filter2": {Object: "item2", Expiration: int64(defaultExpiration)},
				"filter2:key3:filter1": {Object: "item3", Expiration: int64(defaultExpiration)},
			},
			args: args{
				filters: []string{"filter1", "filter2"},
			},
			want: map[string]kv.Item{
				"key2:filter1:filter2": {Object: "item2", Expiration: int64(defaultExpiration)},
				"filter2:key3:filter1": {Object: "item3", Expiration: int64(defaultExpiration)},
			},
		},
		{
			name: "Selects no keys",
			cacheItems: map[string]kv.Item{
				"key1":                 {Object: "item1", Expiration: int64(defaultExpiration)},
				"key2:filter1:filter2": {Object: "item2", Expiration: int64(defaultExpiration)},
				"filter2:key3:filter1": {Object: "item3", Expiration: int64(defaultExpiration)},
			},
			args: args{
				filters: []string{"filter1", "filter3"},
			},
			want: map[string]kv.Item{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Stats{store: kv.NewFrom(defaultExpiration, cleanupInterval, tt.cacheItems)}
			if got := s.FilterKeys(tt.args.filters...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Stats.FilterKeys() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStats_GetCounter(t *testing.T) {
	type args struct {
		nodeID   string
		rtype    string
		version  string
		podID    string
		statName string
	}
	tests := []struct {
		name       string
		cacheItems map[string]kv.Item
		args       args
		want       int64
		wantErr    bool
	}{
		{
			name: "retrieves the counter value",
			cacheItems: map[string]kv.Item{
				"node1:endpoint:aaaa:pod-xxxx:stat1": {Object: int64(2), Expiration: int64(defaultExpiration)},
			},
			args: args{
				nodeID:   "node1",
				rtype:    "endpoint",
				version:  "aaaa",
				podID:    "pod-xxxx",
				statName: "stat1",
			},
			want:    2,
			wantErr: false,
		},
		{
			name:       "returns 0 and error if not found",
			cacheItems: map[string]kv.Item{},
			args: args{
				nodeID:   "node1",
				rtype:    "endpoint",
				version:  "aaaa",
				podID:    "pod-xxxx",
				statName: "stat1",
			},
			want:    0,
			wantErr: true,
		},
		{
			name: "returns error if not a number",
			cacheItems: map[string]kv.Item{
				"node1:endpoint:aaaa:pod-xxxx:stat1": {Object: "kk", Expiration: int64(defaultExpiration)},
			},
			args: args{
				nodeID:   "node1",
				rtype:    "endpoint",
				version:  "aaaa",
				podID:    "pod-xxxx",
				statName: "stat1",
			},
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Stats{store: kv.NewFrom(defaultExpiration, cleanupInterval, tt.cacheItems)}
			got, err := s.GetCounter(tt.args.nodeID, tt.args.rtype, tt.args.version, tt.args.podID, tt.args.statName)
			if (err != nil) != tt.wantErr {
				t.Errorf("Stats.GetCounter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Stats.GetCounter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStats_IncrementCounter(t *testing.T) {
	type args struct {
		nodeID    string
		version   string
		rType     string
		podID     string
		statName  string
		increment int64
	}
	tests := []struct {
		name       string
		cacheItems map[string]kv.Item
		args       args
		want       map[string]kv.Item
	}{
		{
			name: "Increments a value",
			cacheItems: map[string]kv.Item{
				"node:endpoint:aaaa:pod-xxxx:stat": {Object: int64(4), Expiration: int64(defaultExpiration)}},
			args: args{
				nodeID:    "node",
				rType:     "endpoint",
				version:   "aaaa",
				podID:     "pod-xxxx",
				statName:  "stat",
				increment: 1,
			},
			want: map[string]kv.Item{
				"node:endpoint:aaaa:pod-xxxx:stat": {Object: int64(5), Expiration: int64(defaultExpiration)}},
		},
		{
			name:       "Create value if it does not yet exist",
			cacheItems: map[string]kv.Item{},
			args: args{
				nodeID:    "node",
				rType:     "endpoint",
				version:   "aaaa",
				podID:     "pod-xxxx",
				statName:  "stat",
				increment: 1,
			},
			want: map[string]kv.Item{
				"node:endpoint:aaaa:pod-xxxx:stat": {Object: int64(1), Expiration: int64(defaultExpiration)}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Stats{store: kv.NewFrom(defaultExpiration, cleanupInterval, tt.cacheItems)}
			s.IncrementCounter(tt.args.nodeID, tt.args.rType, tt.args.version, tt.args.podID, tt.args.statName, tt.args.increment)
			if got := s.store.Items(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Stats.IncrementCounter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStats_DeleteKeysByFilter(t *testing.T) {
	type args struct {
		filters []string
	}
	tests := []struct {
		name       string
		cacheItems map[string]kv.Item
		args       args
		want       map[string]kv.Item
	}{
		{
			name: "Deletes keys that match all the filters",
			cacheItems: map[string]kv.Item{
				"node1:endpoint:aaaa:pod-xxxx:stat1": {Object: "item1", Expiration: int64(defaultExpiration)},
				"node1:endpoint:aaaa:pod-xxxx:stat2": {Object: "item2", Expiration: int64(defaultExpiration)},
				"node1:cluster:aaaa:pod-xxxx:stat2":  {Object: "item3", Expiration: int64(defaultExpiration)},
				"node1:endpoint:bbbb:pod-xxxx:stat1": {Object: "item4", Expiration: int64(defaultExpiration)},
			},
			args: args{
				filters: []string{"endpoint", "aaaa"},
			},
			want: map[string]kv.Item{
				"node1:cluster:aaaa:pod-xxxx:stat2":  {Object: "item3", Expiration: int64(defaultExpiration)},
				"node1:endpoint:bbbb:pod-xxxx:stat1": {Object: "item4", Expiration: int64(defaultExpiration)},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Stats{store: kv.NewFrom(defaultExpiration, cleanupInterval, tt.cacheItems)}
			s.DeleteKeysByFilter(tt.args.filters...)
			if got := s.store.Items(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Stats.DeleteKeysByFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}
