package server

import "sync"

// KVStore interface defines a key-value storage system.
type KVStore interface {
	Set(key, value []byte)
	Get(key []byte) ([]byte, bool)
	Delete(key []byte) bool
}

// Implement the KVStore interface with a map.
type InMemoryKVStore struct {
	store map[string][]byte
	mu    sync.RWMutex
}

func NewInMemoryKVStore() *InMemoryKVStore {
	return &InMemoryKVStore{
		store: make(map[string][]byte),
	}
}

func (kv *InMemoryKVStore) Set(key, value []byte) {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	kv.store[string(key)] = value
}

func (kv *InMemoryKVStore) Get(key []byte) ([]byte, bool) {
	kv.mu.RLock()
	defer kv.mu.RUnlock()

	value, exists := kv.store[string(key)]
	return value, exists
}

func (kv *InMemoryKVStore) Delete(key []byte) bool {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	_, exists := kv.store[string(key)]
	if exists {
		delete(kv.store, string(key))
	}
	return exists
}
