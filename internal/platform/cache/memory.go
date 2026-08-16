package cache

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/application/cache"
)

// Memory is an in-process cache useful for development and tests.
// Do not use across multiple API replicas.
type Memory struct {
	mu    sync.RWMutex
	items map[string]memItem
}

type memItem struct {
	value     []byte
	expiresAt time.Time
}

// NewMemory builds an empty memory cache.
// Items are held in memory only and are lost when the process exits.
func NewMemory() *Memory {
	return &Memory{items: make(map[string]memItem)}
}

// Get implements cache.Cache. The context is ignored. It returns
// cache.ErrNotFound when the key is missing or expired.
func (m *Memory) Get(_ context.Context, key string) ([]byte, error) {
	m.mu.RLock()
	item, ok := m.items[key]
	m.mu.RUnlock()
	if !ok || (!item.expiresAt.IsZero() && time.Now().After(item.expiresAt)) {
		return nil, cache.ErrNotFound
	}
	return item.value, nil
}

// Set implements cache.Cache. The context is ignored. ttl=0 stores the value
// without an expiry; the value slice is kept by reference.
func (m *Memory) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	item := memItem{value: value}
	if ttl > 0 {
		item.expiresAt = time.Now().Add(ttl)
	}
	m.mu.Lock()
	m.items[key] = item
	m.mu.Unlock()
	return nil
}

// Delete implements cache.Cache. The context is ignored. Deleting a missing
// key is a no-op and never returns an error.
func (m *Memory) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	delete(m.items, key)
	m.mu.Unlock()
	return nil
}

// GetDelete implements cache.Cache. The read and delete happen under one lock,
// so a concurrent GetDelete cannot double-redeem a single-use token.
func (m *Memory) GetDelete(_ context.Context, key string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	item, ok := m.items[key]
	if !ok || (!item.expiresAt.IsZero() && time.Now().After(item.expiresAt)) {
		delete(m.items, key)
		return nil, cache.ErrNotFound
	}
	delete(m.items, key)
	return item.value, nil
}

// Increment implements cache.Cache. The context is ignored. A missing or
// non-numeric value counts as zero before adding delta; the new value is
// returned and the counter is stored as text.
func (m *Memory) Increment(_ context.Context, key string, delta int64) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	item := m.items[key]
	if !item.expiresAt.IsZero() && time.Now().After(item.expiresAt) {
		delete(m.items, key)
		item = memItem{}
	}
	n, err := strconv.ParseInt(string(item.value), 10, 64)
	if err != nil {
		n = 0
	}
	n += delta
	item.value = []byte(strconv.FormatInt(n, 10))
	m.items[key] = item
	return n, nil
}

// Expire implements cache.Cache. The context is ignored. A non-positive ttl
// removes the key; otherwise the key's expiry is extended to now+ttl.
func (m *Memory) Expire(_ context.Context, key string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ttl <= 0 {
		delete(m.items, key)
		return nil
	}
	item, ok := m.items[key]
	if !ok {
		return nil
	}
	item.expiresAt = time.Now().Add(ttl)
	m.items[key] = item
	return nil
}

// Ping implements cache.Cache. An in-memory cache is always reachable, so it
// returns nil without touching the context.
func (m *Memory) Ping(context.Context) error { return nil }

// Close implements cache.Cache. There are no pooled resources to release, so
// it returns nil.
func (m *Memory) Close() error { return nil }
