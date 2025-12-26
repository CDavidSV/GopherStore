package server

import (
	"fmt"
	"sync"
	"time"

	"github.com/CDavidSV/GopherStore/internal/util"
)

// KVStore interface defines a key-value storage system.
type KVStore interface {
	Set(key, value []byte, expiresAt int64)                          // Sets a key-value pair with optional expiration time (-1 means no expiration).
	Push(key []byte, values [][]byte, pushAtFront bool) (int, error) // Pushes values to a list stored at key. If pushAtFront is true, values are added to the front.
	Pop(key []byte, popAtFront bool) ([]byte, error)                 // Pops a value from a list stored at key. Returns nil if the list is empty or key does not exist.
	GetValue(key []byte) ([]byte, error)                             // Retrieves the value for a given key.
	GetList(key []byte) ([][]byte, error)                            // Retrieves the list for a given key.
	Delete(keys [][]byte) int64                                      // Deletes a key-value pair. Returning the number of keys deleted.
	Exists(keys [][]byte) int64                                      // Returns the number of keys currently stored.
	Expire(key []byte, expiresAt int64) bool                         // Sets expiration for a key. Returns true if the key exists and expiration is set.
	Close()                                                          // Closes the store and releases resources.
}

type Entry struct {
	value     []byte
	list      [][]byte
	isList    bool
	expiresAt int64
}

func NewValueEntry(value []byte, expiresAt int64) *Entry {
	return &Entry{
		value:     value,
		isList:    false,
		expiresAt: expiresAt,
	}
}

func NewListEntry(list [][]byte, expiresAt int64) *Entry {
	return &Entry{
		list:      list,
		isList:    true,
		expiresAt: expiresAt,
	}
}

// Checks if the current entry is expired.
func (e *Entry) isExpired() bool {
	return e.expiresAt > 0 && time.Now().UnixNano() > e.expiresAt
}

// Implement the KVStore interface with a map.
type InMemoryKVStore struct {
	store     map[string]*Entry
	expirable map[string]struct{}
	mu        sync.RWMutex
	closeCh   chan struct{}
	closed    bool
}

const (
	cleanupInterval   = time.Millisecond * 250
	cleanupCountBound = 25
)

// Removes a key from both the store and expirable maps.
// Must be called with the lock already held.
func (kv *InMemoryKVStore) deleteKey(key string) {
	delete(kv.store, key)
	delete(kv.expirable, key)
}

func NewInMemoryKVStore() *InMemoryKVStore {
	store := &InMemoryKVStore{
		store:     make(map[string]*Entry),
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

	entry := NewValueEntry(value, expiresAt)

	if expiresAt > 0 {
		kv.expirable[string(key)] = struct{}{}
	}
	kv.store[string(key)] = entry
}

func (kv *InMemoryKVStore) get(key []byte) (*Entry, bool) {
	kv.mu.RLock()
	if kv.closed {
		kv.mu.RUnlock()
		return nil, false
	}

	entry, exists := kv.store[string(key)]
	kv.mu.RUnlock()
	if !exists {
		return nil, false
	}

	// Check expiration
	if entry.isExpired() {
		// Key has expired
		kv.mu.Lock()
		kv.deleteKey(string(key))
		kv.mu.Unlock()
		return nil, false
	}

	return entry, true
}

func (kv *InMemoryKVStore) GetValue(key []byte) ([]byte, error) {
	entry, exists := kv.get(key)
	if !exists {
		return nil, nil
	}

	if entry.isList {
		return nil, fmt.Errorf("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	return entry.value, nil
}

func (kv *InMemoryKVStore) GetList(key []byte) ([][]byte, error) {
	kv.mu.RLock()
	defer kv.mu.RUnlock()

	if kv.closed {
		return nil, fmt.Errorf("store is closed")
	}

	entry, exists := kv.get(key)
	if !exists {
		return nil, nil
	}

	if !entry.isList {
		return nil, fmt.Errorf("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	return entry.list, nil
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
			kv.deleteKey(string(key))
			deletedKeys++
		}

		// We set key to nil to help with garbage collection
		key = nil
	}

	return deletedKeys
}

func (kv *InMemoryKVStore) Exists(keys [][]byte) int64 {
	kv.mu.RLock()
	defer kv.mu.RUnlock()

	if kv.closed {
		return 0
	}

	var existingKeys int64 = 0
	for _, key := range keys {
		entry, exists := kv.store[string(key)]
		if exists {
			// Check expiration
			if entry.isExpired() {
				// Key has expired, skip counting
				continue
			}
			existingKeys++
		}
	}

	return existingKeys
}

func (kv *InMemoryKVStore) Expire(key []byte, expiresAt int64) bool {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	if kv.closed {
		return false
	}

	entry, exists := kv.store[string(key)]
	if !exists {
		return false
	}

	// Check if expired already
	if entry.isExpired() {
		// Key has expired
		kv.deleteKey(string(key))
		return false
	}

	// Update expiration time
	entry.expiresAt = expiresAt
	kv.store[string(key)] = entry

	return true
}

func (kv *InMemoryKVStore) Push(key []byte, values [][]byte, pushAtFront bool) (int, error) {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	if kv.closed {
		return 0, fmt.Errorf("store is closed")
	}

	entry, exists := kv.store[string(key)]
	if exists && !entry.isList {
		return 0, fmt.Errorf("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	// Check if expired already
	if exists && entry.isExpired() {
		// Key has expired
		kv.deleteKey(string(key))
		exists = false
	}

	// Craete a copy of the values
	elements := make([][]byte, len(values))
	copy(elements, values)

	// Depending on pushAtFront, we add elements to the front or back
	if exists {
		if pushAtFront {
			util.ReverseSlice(elements)
			entry.list = append(elements, entry.list...)
		} else {
			entry.list = append(entry.list, elements...)
		}
	} else {
		if pushAtFront {
			util.ReverseSlice(elements)
		}

		entry = NewListEntry(elements, -1)
		kv.store[string(key)] = entry
	}

	return len(entry.list), nil
}

func (kv *InMemoryKVStore) Pop(key []byte, popAtFront bool) ([]byte, error) {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	if kv.closed {
		return nil, fmt.Errorf("store is closed")
	}

	entry, exists := kv.store[string(key)]
	if exists && !entry.isList {
		return nil, fmt.Errorf("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	// Check if expired already
	if exists && entry.isExpired() {
		// Key has expired
		kv.deleteKey(string(key))
		return nil, nil
	}

	if !exists || len(entry.list) == 0 {
		return nil, nil
	}

	var value []byte

	if popAtFront {
		value = entry.list[0]
		entry.list = entry.list[1:]
	} else {
		value = entry.list[len(entry.list)-1]
		entry.list = entry.list[:len(entry.list)-1]
	}
	// We do not delete the key even if empty

	return value, nil
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
				if entry, exists := kv.store[key]; exists {
					if entry.isExpired() {
						kv.deleteKey(key)
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
