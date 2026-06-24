// Package cache provides a tiny invalidation-based in-memory cache for
// read-heavy endpoints whose results only change on a known write event
// (queue mutations, admin CRUD).
//
// Why this exists: the database is hosted on Neon, which bills per
// compute-hour and auto-suspends ("scales to zero") only after a stretch with
// no queries. Display TVs and kiosks poll the read endpoints on an interval,
// and every poll that reaches Postgres keeps the compute awake 24/7 even when
// nothing has changed. By serving those polls from memory and only touching
// the DB after an actual write, quiet periods produce zero queries and let the
// compute suspend.
//
// Entries deliberately have NO time-based TTL: a TTL would expire between
// polls and wake the DB anyway, defeating the purpose. Freshness instead comes
// from explicit Invalidate() calls on every write. This assumes a single
// backend instance (the only writer); it is not safe for horizontal scaling
// without a shared invalidation signal.
package cache

import "sync"

// Keyed is a concurrency-safe string-keyed cache with explicit invalidation.
// Stored values are treated as immutable snapshots — callers must not mutate a
// value returned by Get, since it is shared with other readers.
type Keyed[V any] struct {
	mu      sync.RWMutex
	entries map[string]V
}

// NewKeyed returns an empty cache ready for use.
func NewKeyed[V any]() *Keyed[V] {
	return &Keyed[V]{entries: make(map[string]V)}
}

// Get returns the cached value for key and whether it was present.
func (c *Keyed[V]) Get(key string) (V, bool) {
	c.mu.RLock()
	v, ok := c.entries[key]
	c.mu.RUnlock()
	return v, ok
}

// Set stores value under key.
func (c *Keyed[V]) Set(key string, value V) {
	c.mu.Lock()
	c.entries[key] = value
	c.mu.Unlock()
}

// Invalidate drops every entry. Called after any write that could change a
// cached read so the next Get repopulates from the source of truth.
func (c *Keyed[V]) Invalidate() {
	c.mu.Lock()
	c.entries = make(map[string]V)
	c.mu.Unlock()
}
