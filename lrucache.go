package lrucache

import (
	"container/heap"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hashicorp/go-memdb"
)

// EvictCallback is a function that is called when an item is evicted from the cache.
type EvictCallback func(key string, value interface{})

type Options struct {
	LogLevel      string // "debug", "info", "warn", "error"
	EvictCallback EvictCallback
}

type LRU struct {
	db      *memdb.MemDB
	size    int
	opts    Options
	lock    sync.RWMutex
	expHeap *expirationHeap
}

func NewLRUWithTTL(size int, opts Options) (*LRU, error) {
	if size <= 0 {
		return nil, errors.New("cache size must be positive")
	}

	// Define the schema
	schema := &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			"cache": {
				Name: "cache",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:    "id",
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "Key"},
					},
				},
			},
		},
	}

	// Create a new database
	db, err := memdb.NewMemDB(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to create memdb: %v", err)
	}

	lru := &LRU{
		db:   db,
		size: size,
		opts: opts,
		expHeap: &expirationHeap{
			items:     make([]string, 0, size),
			expiresAt: make(map[string]time.Time),
		},
	}

	go lru.expirationManager()
	return lru, nil
}

func (l *LRU) expirationManager() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		l.removeExpiredItems()
	}
}

func (l *LRU) removeExpiredItems() {
	l.lock.Lock()
	defer l.lock.Unlock()

	now := time.Now()
	for l.expHeap.Len() > 0 && l.expHeap.expiresAt[l.expHeap.items[0]].Before(now) {
		key := heap.Pop(l.expHeap).(string)
		l.removeItem(key)
	}
}

func (l *LRU) Set(key string, value interface{}, ttl time.Duration) error {
	if ttl <= 0 {
		return errors.New("ttl must be positive")
	}

	l.lock.Lock()
	defer l.lock.Unlock()

	expiresAt := time.Now().Add(ttl)
	data, err := serialize(value)
	if err != nil {
		return fmt.Errorf("failed to serialize value: %v", err)
	}

	item := &CacheItem{Key: key, Value: data, ExpiresAt: expiresAt}

	txn := l.db.Txn(true)
	if err := txn.Insert("cache", item); err != nil {
		txn.Abort()
		return fmt.Errorf("failed to insert item: %v", err)
	}
	txn.Commit()

	l.expHeap.expiresAt[key] = expiresAt
	heap.Push(l.expHeap, key)

	// Evict if over capacity
	for l.expHeap.Len() > l.size {
		evictKey := heap.Pop(l.expHeap).(string)
		l.removeItem(evictKey)
	}

	l.log("debug", "Set key: %s, TTL: %v", key, ttl)
	return nil
}

func (l *LRU) Get(key string) (interface{}, error) {
	l.lock.RLock()
	defer l.lock.RUnlock()

	txn := l.db.Txn(false)
	raw, err := txn.First("cache", "id", key)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve item: %v", err)
	}
	if raw == nil {
		return nil, ErrItemNotFound
	}

	item := raw.(*CacheItem)
	if time.Now().After(item.ExpiresAt) {
		l.removeItem(key)
		return nil, ErrItemExpired
	}

	value, err := deserialize(item.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize value: %v", err)
	}

	l.log("debug", "Get key: %s", key)
	return value, nil
}

func (l *LRU) Delete(key string) error {
	l.lock.Lock()
	defer l.lock.Unlock()

	txn := l.db.Txn(true)
	if _, err := txn.First("cache", "id", key); err != nil {
		txn.Abort()
		return fmt.Errorf("failed to find item: %v", err)
	} else {
		if err := txn.Delete("cache", &CacheItem{Key: key}); err != nil {
			txn.Abort()
			return fmt.Errorf("failed to delete item: %v", err)
		}
	}
	txn.Commit()

	delete(l.expHeap.expiresAt, key)
	l.log("debug", "Deleted key: %s", key)
	return nil
}

func (l *LRU) Clear() error {
	l.lock.Lock()
	defer l.lock.Unlock()

	txn := l.db.Txn(true)
	raw, err := txn.Get("cache", "id")
	if err != nil {
		txn.Abort()
		return fmt.Errorf("failed to get all items: %v", err)
	}

	for obj := raw.Next(); obj != nil; obj = raw.Next() {
		item := obj.(*CacheItem)
		if err := txn.Delete("cache", item); err != nil {
			txn.Abort()
			return fmt.Errorf("failed to delete item: %v", err)
		}
	}
	txn.Commit()

	l.expHeap.items = l.expHeap.items[:0]
	l.expHeap.expiresAt = make(map[string]time.Time)

	l.log("info", "Cache cleared")
	return nil
}

func (l *LRU) Len() int {
	l.lock.RLock()
	defer l.lock.RUnlock()

	txn := l.db.Txn(false)
	it, err := txn.Get("cache", "id")
	if err != nil {
		l.log("error", "Failed to get cache size: %v", err)
		return 0
	}
	count := 0
	for obj := it.Next(); obj != nil; obj = it.Next() {
		count++
	}
	return count
}

func (l *LRU) removeItem(key string) {
	txn := l.db.Txn(true)
	if err := txn.Delete("cache", &CacheItem{Key: key}); err != nil {
		txn.Abort()
		l.log("error", "Failed to remove item: %v", err)
		return
	}
	txn.Commit()

	delete(l.expHeap.expiresAt, key)

	if l.opts.EvictCallback != nil {
		l.opts.EvictCallback(key, nil)
	}
}

func (l *LRU) log(level, format string, v ...interface{}) {
	switch l.opts.LogLevel {
	case "debug":
		log.Printf("[DEBUG] "+format, v...)
	case "info":
		if level != "debug" {
			log.Printf("[INFO] "+format, v...)
		}
	case "warn":
		if level != "debug" && level != "info" {
			log.Printf("[WARN] "+format, v...)
		}
	case "error":
		if level == "error" {
			log.Printf("[ERROR] "+format, v...)
		}
	}
}
