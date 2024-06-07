package lrucache

import "errors"

var (
	ErrCacheNotInitialized = errors.New("cache not initialized")
	ErrItemExpired         = errors.New("item expired")
	ErrItemNotFound        = errors.New("item not found")
)
