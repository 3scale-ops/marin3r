package discoveryservice

import (
	"testing"

	xdss "github.com/3scale-ops/marin3r/pkg/discoveryservice/xdss"
	testutil "github.com/3scale-ops/marin3r/pkg/util/test"
	envoy_config_endpoint_v3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	cache_types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cache_v3 "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
)

func TestCache_SetSnapshot(t *testing.T) {
	type fields struct {
		v3 cache_v3.SnapshotCache
	}
	type args struct {
		nodeID string
		snap   xdss.Snapshot
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantErr  bool
		wantSnap Snapshot
	}{
		{
			name:   "Write the snapshot in the cache",
			fields: fields{v3: cache_v3.NewSnapshotCache(true, cache_v3.IDHash{}, nil)},
			args: args{nodeID: "node", snap: Snapshot{v3: &cache_v3.Snapshot{
				Resources: [6]cache_v3.Resources{
					{Version: "xxxx", Items: map[string]cache_types.Resource{
						"endpoint": &envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint"}}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
				}}}},
			wantErr: false,
			wantSnap: Snapshot{v3: &cache_v3.Snapshot{
				Resources: [6]cache_v3.Resources{
					{Version: "xxxx", Items: map[string]cache_types.Resource{
						"endpoint": &envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint"}}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
				}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Cache{
				v3: tt.fields.v3,
			}
			if err := c.SetSnapshot(tt.args.nodeID, tt.args.snap); (err != nil) != tt.wantErr {
				t.Errorf("Cache.SetSnapshot() error = %v, wantErr %v", err, tt.wantErr)
			}
			gotSnap, _ := c.GetSnapshot("node")
			if !tt.wantErr && !testutil.SnapshotsAreEqual(gotSnap, tt.wantSnap) {
				t.Errorf("Cache.SetSnapshot() got = %v, wantSnap %v", gotSnap, tt.wantSnap)
			}
		})
	}
}

func TestCache_GetSnapshot(t *testing.T) {
	type fields struct {
		v3 cache_v3.SnapshotCache
	}
	type args struct {
		nodeID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    xdss.Snapshot
		wantErr bool
	}{
		{
			name: "Get the snapshot from the cache",
			fields: fields{
				v3: func() cache_v3.SnapshotCache {
					c := cache_v3.NewSnapshotCache(true, cache_v3.IDHash{}, nil)
					c.SetSnapshot("node", cache_v3.Snapshot{
						Resources: [6]cache_v3.Resources{
							{Version: "xxxx", Items: map[string]cache_types.Resource{
								"endpoint": &envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint"}}},
							{Version: "xxxx", Items: map[string]cache_types.Resource{}},
							{Version: "xxxx", Items: map[string]cache_types.Resource{}},
							{Version: "xxxx", Items: map[string]cache_types.Resource{}},
							{Version: "xxxx", Items: map[string]cache_types.Resource{}},
							{Version: "xxxx", Items: map[string]cache_types.Resource{}},
						}})
					return c
				}(),
			},
			args: args{nodeID: "node"},
			want: Snapshot{v3: &cache_v3.Snapshot{
				Resources: [6]cache_v3.Resources{
					{Version: "xxxx", Items: map[string]cache_types.Resource{
						"endpoint": &envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint"}}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
				}}},
			wantErr: false,
		},
		{
			name: "Snapshot does not exist for given nodeID, error returned",
			fields: fields{
				v3: func() cache_v3.SnapshotCache {
					c := cache_v3.NewSnapshotCache(true, cache_v3.IDHash{}, nil)
					c.SetSnapshot("node", cache_v3.Snapshot{
						Resources: [6]cache_v3.Resources{
							{Version: "xxxx", Items: map[string]cache_types.Resource{
								"endpoint": &envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint"}}},
							{Version: "xxxx", Items: map[string]cache_types.Resource{}},
							{Version: "xxxx", Items: map[string]cache_types.Resource{}},
							{Version: "xxxx", Items: map[string]cache_types.Resource{}},
							{Version: "xxxx", Items: map[string]cache_types.Resource{}},
							{Version: "xxxx", Items: map[string]cache_types.Resource{}},
						}})
					return c
				}(),
			},
			args:    args{nodeID: "other-node"},
			want:    Snapshot{v3: &cache_v3.Snapshot{}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Cache{
				v3: tt.fields.v3,
			}
			got, err := c.GetSnapshot(tt.args.nodeID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Cache.GetSnapshot() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !testutil.SnapshotsAreEqual(got, tt.want) {
				t.Errorf("Cache.GetSnapshot() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCache_ClearSnapshot(t *testing.T) {
	type fields struct {
		v3 cache_v3.SnapshotCache
	}
	type args struct {
		nodeID string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "Snapshot deleted for the given nodeID",
			fields: fields{
				v3: func() cache_v3.SnapshotCache {
					c := cache_v3.NewSnapshotCache(true, cache_v3.IDHash{}, nil)
					c.SetSnapshot("node", cache_v3.Snapshot{
						Resources: [6]cache_v3.Resources{
							{Version: "xxxx", Items: map[string]cache_types.Resource{
								"endpoint": &envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint"}}},
							{Version: "xxxx", Items: map[string]cache_types.Resource{}},
							{Version: "xxxx", Items: map[string]cache_types.Resource{}},
							{Version: "xxxx", Items: map[string]cache_types.Resource{}},
							{Version: "xxxx", Items: map[string]cache_types.Resource{}},
							{Version: "xxxx", Items: map[string]cache_types.Resource{}},
						}})
					return c
				}(),
			},
			args: args{nodeID: "node"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Cache{
				v3: tt.fields.v3,
			}
			c.ClearSnapshot(tt.args.nodeID)
			if _, err := c.GetSnapshot("node"); err == nil {
				t.Errorf("Cache.ClearSnapshot() = not found error expected")
			}
		})
	}
}

func TestCache_NewSnapshot(t *testing.T) {
	type fields struct {
		v3 cache_v3.SnapshotCache
	}
	type args struct {
		resourcesVersion string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   xdss.Snapshot
	}{
		{
			name:   "Returns a new Snapshot object",
			fields: fields{v3: cache_v3.NewSnapshotCache(true, cache_v3.IDHash{}, nil)},
			args:   args{resourcesVersion: "xxxx"},
			want: Snapshot{v3: &cache_v3.Snapshot{
				Resources: [6]cache_v3.Resources{
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
					{Version: "xxxx", Items: map[string]cache_types.Resource{}},
				}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Cache{
				v3: tt.fields.v3,
			}
			if got := c.NewSnapshot(tt.args.resourcesVersion); !testutil.SnapshotsAreEqual(got, tt.want) {
				t.Errorf("Cache.NewSnapshot() = %v, want %v", got, tt.want)
			}
		})
	}
}
