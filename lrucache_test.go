package lrucache

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestLRUBasicOperations(t *testing.T) {
	cache, err := NewLRUWithTTL(3, Options{LogLevel: "error"})
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	if err := cache.Set("key1", "value1", 1*time.Hour); err != nil {
		t.Errorf("Failed to set key1: %v", err)
	}
	if err := cache.Set("key2", 42, 1*time.Hour); err != nil {
		t.Errorf("Failed to set key2: %v", err)
	}

	v, err := cache.Get("key1")
	if err != nil || v.(string) != "value1" {
		t.Errorf("Get key1 failed. Got %v, %v", v, err)
	}

	v, err = cache.Get("key2")
	if err != nil || v.(int) != 42 {
		t.Errorf("Get key2 failed. Got %v, %v", v, err)
	}

	if err := cache.Delete("key1"); err != nil {
		t.Errorf("Delete key1 failed: %v", err)
	}

	if _, err := cache.Get("key1"); err != ErrItemNotFound {
		t.Errorf("Expected ErrItemNotFound, got %v", err)
	}

	if l := cache.Len(); l != 1 {
		t.Errorf("Expected len 1, got %d", l)
	}

	if err := cache.Clear(); err != nil {
		t.Errorf("Clear failed: %v", err)
	}

	if l := cache.Len(); l != 0 {
		t.Errorf("Expected len 0 after clear, got %d", l)
	}
}

func TestLRUExpiration(t *testing.T) {
	cache, _ := NewLRUWithTTL(10, Options{LogLevel: "error"})

	cache.Set("key1", "value1", 100*time.Millisecond)
	cache.Set("key2", "value2", 200*time.Millisecond)

	time.Sleep(150 * time.Millisecond)

	if _, err := cache.Get("key1"); err != ErrItemExpired {
		t.Errorf("Expected key1 to be expired, got %v", err)
	}
	if _, err := cache.Get("key2"); err != nil {
		t.Errorf("key2 should not be expired yet, got %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	if _, err := cache.Get("key2"); err != ErrItemExpired {
		t.Errorf("Expected key2 to be expired, got %v", err)
	}
}

func TestLRUEviction(t *testing.T) {
	cache, _ := NewLRUWithTTL(3, Options{LogLevel: "error"})

	cache.Set("key1", 1, 1*time.Hour)
	cache.Set("key2", 2, 1*time.Hour)
	cache.Set("key3", 3, 1*time.Hour)
	cache.Set("key4", 4, 1*time.Hour) // should evict key1

	if _, err := cache.Get("key1"); err != ErrItemNotFound {
		t.Errorf("key1 should have been evicted, got %v", err)
	}
}

func TestLRUConcurrency(t *testing.T) {
	cache, _ := NewLRUWithTTL(1000, Options{LogLevel: "error"})
	var wg sync.WaitGroup

	// Concurrent sets
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", i)
			cache.Set(key, i, 1*time.Hour)
		}(i)
	}

	// Wait for all sets to complete
	wg.Wait()

	// Concurrent gets
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", i)
			val, err := cache.Get(key)
			if err != nil || val.(int) != i {
				t.Errorf("Concurrent get failed for %s: %v, %v", key, val, err)
			}
		}(i)
	}

	wg.Wait()
	if cache.Len() != 100 {
		t.Errorf("Expected 100 items, got %d", cache.Len())
	}
}

func TestLRUSerializationEdgeCases(t *testing.T) {
	cache, _ := NewLRUWithTTL(10, Options{LogLevel: "error"})

	// Test various types
	cache.Set("string", "hello", 1*time.Hour)
	cache.Set("int", 42, 1*time.Hour)
	cache.Set("float", 3.14, 1*time.Hour)
	cache.Set("bool", true, 1*time.Hour)
	cache.Set("struct", struct{ Name string }{"Alice"}, 1*time.Hour)

	if v, _ := cache.Get("string"); v.(string) != "hello" {
		t.Errorf("String serialization failed, got %v", v)
	}
	if v, _ := cache.Get("int"); v.(int) != 42 {
		t.Errorf("Int serialization failed, got %v", v)
	}
	if v, _ := cache.Get("float"); v.(float64) != 3.14 {
		t.Errorf("Float serialization failed, got %v", v)
	}
	if v, _ := cache.Get("bool"); v.(bool) != true {
		t.Errorf("Bool serialization failed, got %v", v)
	}
	if v, _ := cache.Get("struct"); v.(map[string]interface{})["Name"].(string) != "Alice" {
		t.Errorf("Struct serialization failed, got %v", v)
	}
}

func TestLRUOptions(t *testing.T) {
	evicted := make(map[string]interface{})
	cache, _ := NewLRUWithTTL(3, Options{
		LogLevel: "warn",
		EvictCallback: func(key string, value interface{}) {
			evicted[key] = value
		},
	})

	cache.Set("key1", 1, 1*time.Hour)
	cache.Set("key2", 2, 1*time.Hour)
	cache.Set("key3", 3, 1*time.Hour)
	cache.Set("key4", 4, 1*time.Hour)

	if _, ok := evicted["key1"]; !ok {
		t.Errorf("EvictCallback not called for key1")
	}
}
