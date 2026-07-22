package cache

import "time"

type Store interface {
	Get(key string) ([]byte, bool)
	Set(key string, value []byte, ttl time.Duration)
	Delete(key string)
	InvalidatePrefix(prefix string)
}
