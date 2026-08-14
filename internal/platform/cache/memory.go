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
func NewMemory() *Memory {
	return &Memory{items: make(map[string]memItem)}
}

// Get implements cache.Cache.
func (m *Memory) Get(_ context.Context, key string) ([]byte, error) {
	m.mu.RLock()
	item, ok := m.items[key]
	m.mu.RUnlock()
	if !ok || (!item.expiresAt.IsZero() && time.Now().After(item.expiresAt)) {
		return nil, cache.ErrNotFound
	}
	return item.value, nil
}

// Set implements cache.Cache.
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

// Delete implements cache.Cache.
func (m *Memory) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	delete(m.items, key)
	m.mu.Unlock()
	return nil
}

// Increment implements cache.Cache.
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

// Expire implements cache.Cache.
func (m *Memory) Expire(_ context.Context, key string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	item, ok := m.items[key]
	if !ok {
		return nil
	}
	item.expiresAt = time.Now().Add(ttl)
	m.items[key] = item
	return nil
}

// Ping implements cache.Cache.
func (m *Memory) Ping(context.Context) error { return nil }

// Close implements cache.Cache.
func (m *Memory) Close() error { return nil }
