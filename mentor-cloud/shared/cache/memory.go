package cache

import (
	"strings"
	"sync"
	"time"
)

type entry struct {
	value     []byte
	expiresAt time.Time
}

type MemoryCache struct {
	mu   sync.RWMutex
	data map[string]entry
	once sync.Once
}

func NewMemoryCache() *MemoryCache {
	c := &MemoryCache{
		data: make(map[string]entry),
	}
	c.once.Do(func() {
		go c.evict()
	})
	return c
}

func (c *MemoryCache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	e, ok := c.data[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(e.expiresAt) {
		return nil, false
	}
	return e.value, true
}

func (c *MemoryCache) Set(key string, value []byte, ttl time.Duration) {
	c.mu.Lock()
	c.data[key] = entry{value: value, expiresAt: time.Now().Add(ttl)}
	c.mu.Unlock()
}

func (c *MemoryCache) Delete(key string) {
	c.mu.Lock()
	delete(c.data, key)
	c.mu.Unlock()
}

func (c *MemoryCache) InvalidatePrefix(prefix string) {
	c.mu.Lock()
	for k := range c.data {
		if strings.HasPrefix(k, prefix) {
			delete(c.data, k)
		}
	}
	c.mu.Unlock()
}

func (c *MemoryCache) evict() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		c.mu.Lock()
		for k, e := range c.data {
			if now.After(e.expiresAt) {
				delete(c.data, k)
			}
		}
		c.mu.Unlock()
	}
}
