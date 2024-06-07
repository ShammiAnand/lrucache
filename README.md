# LRU Cache

A thread-safe LRU cache implementation in Go.

## Description

This package provides a thread-safe LRU (Least Recently Used) cache implementation in Go. The cache is implemented using a combination of a hash table and a min-heap, which allows for efficient insertion, retrieval, and eviction of items based on their usage frequency and expiration time.

The cache supports customizable options, such as log level and eviction callback, which allow for fine-grained control over the cache behavior. The cache also supports serialization and deserialization of values, which allows for storing and retrieving arbitrary data types.

## Usage

### Installation

To use this package in your project, add the following import statement to your Go code:

```go
import "github.com/shammianand/lrucache"
```
