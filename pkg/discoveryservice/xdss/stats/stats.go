package stats

import (
	kv "github.com/patrickmn/go-cache"
)

// The format of the keys in the cache is
//   <node-id>:<version>:<resource-type>:<pod-id>:<key>
//
// Note that though currently revision and version have the same value for all
// types (with the execption of secrets), this might change in the future and have
// each resource type follow its own versioning
type Stats struct {
	store *kv.Cache
}

func New() *Stats {
	return &Stats{
		store: kv.New(defaultExpiration, cleanupInterval),
	}
}
