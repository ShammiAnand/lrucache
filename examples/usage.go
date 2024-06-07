package examples

import (
	"fmt"
	"time"

	"github.com/shammianand/lrucache"
)

type MyStruct struct {
	Name string
	Age  int
}

func main() {
	evicted := make(map[string]interface{})
	cache, _ := lrucache.NewLRUWithTTL(100, lrucache.Options{
		LogLevel: "warn",
		EvictCallback: func(key string, value interface{}) {
			evicted[key] = value
		},
	})

	cache.Set("key1", MyStruct{"Alice", 30}, 1*time.Hour)

	val, _ := cache.Get("key1")
	fmt.Println(val) // Output: {Alice 30}

	cache.Delete("key1")

	fmt.Println(evicted["key1"]) // Output: {Alice 30}
}
