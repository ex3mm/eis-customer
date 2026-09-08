package cache

import (
	"sync"
	"time"
)

type entry struct {
	value     []byte
	expiresAt time.Time
}

type Cache struct {
	enabled bool
	ttl     time.Duration
	mu      sync.Mutex
	items   map[string]entry
}

func New(enabled bool, ttl time.Duration) *Cache {
	return &Cache{enabled: enabled, ttl: ttl, items: make(map[string]entry)}
}

func (c *Cache) Get(key string) ([]byte, bool) {
	if !c.enabled {
		return nil, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	item, ok := c.items[key]
	if !ok || time.Now().After(item.expiresAt) {
		delete(c.items, key)
		return nil, false
	}
	return append([]byte(nil), item.value...), true
}

func (c *Cache) Set(key string, value []byte) {
	if !c.enabled {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = entry{value: append([]byte(nil), value...), expiresAt: time.Now().Add(c.ttl)}
}
