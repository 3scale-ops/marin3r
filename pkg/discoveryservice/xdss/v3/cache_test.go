package discoveryservice

import (
	"context"
	"testing"

	xdss "github.com/3scale-ops/marin3r/pkg/discoveryservice/xdss"
	"github.com/3scale-ops/marin3r/pkg/envoy"
	testutil "github.com/3scale-ops/marin3r/pkg/util/test"
	envoy_config_endpoint_v3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
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
		wantSnap xdss.Snapshot
	}{
		{
			name:   "Write the snapshot in the cache",
			fields: fields{v3: cache_v3.NewSnapshotCache(true, cache_v3.IDHash{}, nil)},
			args: args{
				nodeID: "node",
				snap: NewSnapshot().SetResources(envoy.Endpoint, []envoy.Resource{
					&envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint"}}),
			},
			wantErr: false,
			wantSnap: NewSnapshot().SetResources(envoy.Endpoint, []envoy.Resource{
				&envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint"},
			}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Cache{
				v3: tt.fields.v3,
			}
			if err := c.SetSnapshot(context.TODO(), tt.args.nodeID, tt.args.snap); (err != nil) != tt.wantErr {
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
	type args struct {
		nodeID string
	}
	tests := []struct {
		name    string
		cache   xdss.Cache
		args    args
		want    xdss.Snapshot
		wantErr bool
	}{
		{
			name: "Get the snapshot from the cache",
			cache: func() Cache {
				c := NewCache()
				c.SetSnapshot(context.TODO(), "node", NewSnapshot().SetResources(envoy.Endpoint, []envoy.Resource{
					&envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint"},
				}))
				return c
			}(),
			args: args{nodeID: "node"},
			want: NewSnapshot().SetResources(envoy.Endpoint, []envoy.Resource{
				&envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint"},
			}),
			wantErr: false,
		},
		{
			name: "Snapshot does not exist for given nodeID, error returned",
			cache: func() Cache {
				c := NewCache()
				c.SetSnapshot(context.TODO(), "node", NewSnapshot().SetResources(envoy.Endpoint, []envoy.Resource{
					&envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint"},
				}))
				return c
			}(),
			args:    args{nodeID: "other-node"},
			want:    Snapshot{v3: &cache_v3.Snapshot{}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.cache
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
	type args struct {
		nodeID string
	}
	tests := []struct {
		name  string
		cache xdss.Cache
		args  args
	}{
		{
			name: "Snapshot deleted for the given nodeID",
			cache: func() Cache {
				c := NewCache()
				c.SetSnapshot(context.TODO(), "node", NewSnapshot().SetResources(envoy.Endpoint, []envoy.Resource{
					&envoy_config_endpoint_v3.ClusterLoadAssignment{ClusterName: "endpoint"},
				}))
				return c
			}(),
			args: args{nodeID: "node"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.cache
			c.ClearSnapshot(tt.args.nodeID)
			if _, err := c.GetSnapshot("node"); err == nil {
				t.Errorf("Cache.ClearSnapshot() = not found error expected")
			}
		})
	}
}
