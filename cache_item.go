package lrucache

import "time"

type CacheItem struct {
	Key       string
	Value     []byte
	ExpiresAt time.Time
}
