package adapters

import (
	"sync"
	"time"
)

type bucket struct {
	mu       sync.Mutex
	tokens   float64
	cap      float64
	rate     float64
	lastFill time.Time
}

func (b *bucket) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	b.tokens += now.Sub(b.lastFill).Seconds() * b.rate
	b.lastFill = now
	if b.tokens > b.cap {
		b.tokens = b.cap
	}
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	cap     float64
	rate    float64
}

func NewRateLimiter(cap, ratePerSec float64) *RateLimiter {
	rl := &RateLimiter{
		buckets: make(map[string]*bucket),
		cap:     cap,
		rate:    ratePerSec,
	}
	go rl.cleanup()
	return rl
}

func (r *RateLimiter) Allow(key string) bool {
	r.mu.Lock()
	b, ok := r.buckets[key]
	if !ok {
		b = &bucket{tokens: r.cap, cap: r.cap, rate: r.rate, lastFill: time.Now()}
		r.buckets[key] = b
	}
	r.mu.Unlock()
	return b.allow()
}

func (r *RateLimiter) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-30 * time.Minute)
		r.mu.Lock()
		for k, b := range r.buckets {
			b.mu.Lock()
			idle := b.lastFill.Before(cutoff)
			b.mu.Unlock()
			if idle {
				delete(r.buckets, k)
			}
		}
		r.mu.Unlock()
	}
}
