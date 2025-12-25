package server

import (
	"sync"
	"time"
)

// KVStore interface defines a key-value storage system.
type KVStore interface {
	Set(key, value []byte, expiresAt int64) // Sets a key-value pair with optional expiration time (-1 means no expiration).
	Get(key []byte) ([]byte, bool)          // Retrieves the value for a given key.
	Delete(keys [][]byte) int64             // Deletes a key-value pair. Returning the number of keys deleted.
	Close()                                 // Closes the store and releases resources.
}

type Box struct {
	data      []byte
	expiresAt int64
}

// Implement the KVStore interface with a map.
type InMemoryKVStore struct {
	store     map[string]Box
	expirable map[string]struct{}
	mu        sync.RWMutex
	closeCh   chan struct{}
	closed    bool
}

const (
	cleanupInterval   = time.Millisecond * 250
	cleanupCountBound = 25
)

func NewInMemoryKVStore() *InMemoryKVStore {
	store := &InMemoryKVStore{
		store:     make(map[string]Box),
		expirable: make(map[string]struct{}),
		closeCh:   make(chan struct{}),
		closed:    false,
	}

	go store.cleanupExpiredKeys()

	return store
}

func (kv *InMemoryKVStore) Set(key, value []byte, expiresAt int64) {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	if kv.closed {
		return
	}

	box := Box{
		data:      value,
		expiresAt: expiresAt,
	}

	if expiresAt > 0 {
		kv.expirable[string(key)] = struct{}{}
	}
	kv.store[string(key)] = box
}

func (kv *InMemoryKVStore) Get(key []byte) ([]byte, bool) {
	kv.mu.RLock()
	if kv.closed {
		kv.mu.RUnlock()
		return nil, false
	}

	value, exists := kv.store[string(key)]
	kv.mu.RUnlock()
	if !exists {
		return nil, false
	}

	// Check expiration
	if value.expiresAt > 0 && time.Now().UnixNano() > value.expiresAt {
		// Key has expired
		kv.mu.Lock()
		delete(kv.store, string(key))
		delete(kv.expirable, string(key))
		kv.mu.Unlock()
		return nil, false
	}

	return value.data, exists
}

func (kv *InMemoryKVStore) Delete(keys [][]byte) int64 {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	if kv.closed {
		return 0
	}

	var deletedKeys int64 = 0
	for _, key := range keys {
		_, exists := kv.store[string(key)]
		if exists {
			delete(kv.store, string(key))
			delete(kv.expirable, string(key))
			deletedKeys++
		}

		// We set key to nil to help with garbage collection
		key = nil
	}

	return deletedKeys
}

func (kv *InMemoryKVStore) Close() {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	if kv.closed {
		return
	}

	kv.closed = true
	close(kv.closeCh)
}

func (kv *InMemoryKVStore) cleanupExpiredKeys() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			checked := 0
			kv.mu.Lock()

			// Iterate over expirable keys and remove expired ones
			for key := range kv.expirable {
				// If the key exists, check expiration and delete if expired
				if value, exists := kv.store[key]; exists {
					if value.expiresAt > 0 && time.Now().UnixNano() > value.expiresAt {
						delete(kv.store, key)
						delete(kv.expirable, key)
					}
				} else {
					// Key no longer exists, remove from expirable map
					delete(kv.expirable, key)
				}

				checked++
				// Only check a limited number of keys per interval
				if checked >= cleanupCountBound {
					kv.mu.Unlock()
					break
				}
			}
			kv.mu.Unlock()
		case <-kv.closeCh:
			// Store closed, exit the goroutine
			return
		}
	}
}
