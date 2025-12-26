package server

import (
	"sync"
	"testing"
	"time"
)

func TestSetAndGet(t *testing.T) {
	store := NewInMemoryKVStore()
	defer store.Close()

	key := []byte("testkey")
	value := []byte("testvalue")

	// Set without expiration
	store.Set(key, value, -1)

	// Get the value back
	result, exists := store.Get(key)
	if !exists {
		t.Fatal("Expected key to exist")
	}

	if string(result) != string(value) {
		t.Errorf("Expected %s, got %s", value, result)
	}
}

func TestGetNonExistent(t *testing.T) {
	store := NewInMemoryKVStore()
	defer store.Close()

	key := []byte("nonexistent")

	result, exists := store.Get(key)
	if exists {
		t.Fatal("Expected key to not exist")
	}

	if result != nil {
		t.Errorf("Expected nil value, got %v", result)
	}
}

func TestDelete(t *testing.T) {
	store := NewInMemoryKVStore()
	defer store.Close()

	key1 := []byte("key1")
	key2 := []byte("key2")
	key3 := []byte("key3")
	value := []byte("value")

	// Set multiple keys
	store.Set(key1, value, -1)
	store.Set(key2, value, -1)

	// Delete existing and non-existing keys
	deletedCount := store.Delete([][]byte{key1, key2, key3})

	if deletedCount != 2 {
		t.Errorf("Expected 2 keys deleted, got %d", deletedCount)
	}

	// Verify keys are deleted
	_, exists := store.Get(key1)
	if exists {
		t.Error("Expected key1 to be deleted")
	}

	_, exists = store.Get(key2)
	if exists {
		t.Error("Expected key2 to be deleted")
	}
}

func TestExpiration(t *testing.T) {
	store := NewInMemoryKVStore()
	defer store.Close()

	key := []byte("expiring_key")
	value := []byte("expiring_value")

	// Set key to expire in 100ms
	expiresAt := time.Now().Add(100 * time.Millisecond).UnixNano()
	store.Set(key, value, expiresAt)

	// Should exist immediately
	result, exists := store.Get(key)
	if !exists {
		t.Fatal("Expected key to exist")
	}
	if string(result) != string(value) {
		t.Errorf("Expected %s, got %s", value, result)
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should not exist after expiration
	_, exists = store.Get(key)
	if exists {
		t.Error("Expected key to be expired")
	}
}

func TestExpirationCleanup(t *testing.T) {
	store := NewInMemoryKVStore()
	defer store.Close()

	// Set multiple keys with expiration
	for i := 0; i < 10; i++ {
		key := []byte{byte(i)}
		value := []byte("value")
		expiresAt := time.Now().Add(50 * time.Millisecond).UnixNano()
		store.Set(key, value, expiresAt)
	}

	// Wait for cleanup to run (cleanup interval is 250ms)
	time.Sleep(400 * time.Millisecond)

	// Verify keys are cleaned up
	store.mu.RLock()
	storeLen := len(store.store)
	expirableLen := len(store.expirable)
	store.mu.RUnlock()

	if storeLen != 0 {
		t.Errorf("Expected store to be empty, but has %d entries", storeLen)
	}

	if expirableLen != 0 {
		t.Errorf("Expected expirable map to be empty, but has %d entries", expirableLen)
	}
}

func TestUpdateExistingKey(t *testing.T) {
	store := NewInMemoryKVStore()
	defer store.Close()

	key := []byte("key")
	value1 := []byte("value1")
	value2 := []byte("value2")

	store.Set(key, value1, -1)
	result, _ := store.Get(key)
	if string(result) != string(value1) {
		t.Errorf("Expected %s, got %s", value1, result)
	}

	// Update the key
	store.Set(key, value2, -1)
	result, _ = store.Get(key)
	if string(result) != string(value2) {
		t.Errorf("Expected %s, got %s", value2, result)
	}
}

func TestClose(t *testing.T) {
	store := NewInMemoryKVStore()

	key := []byte("key")
	value := []byte("value")
	store.Set(key, value, -1)

	store.Close()

	// Operations after close should be no-op
	store.Set([]byte("newkey"), []byte("newvalue"), -1)

	result, exists := store.Get(key)
	if exists || result != nil {
		t.Error("Expected Get to return false after Close")
	}

	deletedCount := store.Delete([][]byte{key})
	if deletedCount != 0 {
		t.Errorf("Expected 0 deletions after Close, got %d", deletedCount)
	}

	// Calling Close again should be safe
	store.Close()
}

func TestConcurrentAccess(t *testing.T) {
	store := NewInMemoryKVStore()
	defer store.Close()

	var wg sync.WaitGroup
	numGoroutines := 100
	numOperations := 100

	// Concurrent writes
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := []byte{byte(id), byte(j)}
				value := []byte{byte(id * j)}
				store.Set(key, value, -1)
			}
		}(i)
	}

	// Concurrent reads
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := []byte{byte(id), byte(j)}
				store.Get(key)
			}
		}(i)
	}

	// Concurrent deletes
	wg.Add(numGoroutines / 2)
	for i := 0; i < numGoroutines/2; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations/2; j++ {
				key := []byte{byte(id), byte(j)}
				store.Delete([][]byte{key})
			}
		}(i)
	}

	wg.Wait()
	// Test passes if no race conditions or panics occur
}

func TestExpirationEdgeCases(t *testing.T) {
	store := NewInMemoryKVStore()
	defer store.Close()

	t.Run("Already expired key", func(t *testing.T) {
		key := []byte("past_key")
		value := []byte("past_value")
		// Set expiration in the past
		expiresAt := time.Now().Add(-1 * time.Second).UnixNano()
		store.Set(key, value, expiresAt)

		// Should immediately return not found
		_, exists := store.Get(key)
		if exists {
			t.Error("Expected already-expired key to not exist")
		}
	})

	t.Run("No expiration (value 0)", func(t *testing.T) {
		key := []byte("no_expire_0")
		value := []byte("value")
		store.Set(key, value, 0)

		time.Sleep(100 * time.Millisecond)
		result, exists := store.Get(key)
		if !exists {
			t.Error("Expected key with expiration 0 to exist")
		}
		if string(result) != string(value) {
			t.Errorf("Expected %s, got %s", value, result)
		}
	})

	t.Run("No expiration (value -1)", func(t *testing.T) {
		key := []byte("no_expire_neg")
		value := []byte("value")
		store.Set(key, value, -1)

		time.Sleep(100 * time.Millisecond)
		result, exists := store.Get(key)
		if !exists {
			t.Error("Expected key with expiration -1 to exist")
		}
		if string(result) != string(value) {
			t.Errorf("Expected %s, got %s", value, result)
		}
	})
}

func TestEmptyKeys(t *testing.T) {
	store := NewInMemoryKVStore()
	defer store.Close()

	emptyKey := []byte{}
	value := []byte("value")

	store.Set(emptyKey, value, -1)
	result, exists := store.Get(emptyKey)
	if !exists {
		t.Error("Expected empty key to be stored")
	}
	if string(result) != string(value) {
		t.Errorf("Expected %s, got %s", value, result)
	}

	deletedCount := store.Delete([][]byte{emptyKey})
	if deletedCount != 1 {
		t.Errorf("Expected 1 deletion, got %d", deletedCount)
	}
}

func TestDeleteMultiple(t *testing.T) {
	store := NewInMemoryKVStore()
	defer store.Close()

	// Set up multiple keys
	keys := [][]byte{
		[]byte("key1"),
		[]byte("key2"),
		[]byte("key3"),
		[]byte("key4"),
		[]byte("key5"),
	}
	value := []byte("value")

	for _, key := range keys {
		store.Set(key, value, -1)
	}

	// Delete some existing and some non-existing keys
	toDelete := [][]byte{
		[]byte("key1"),
		[]byte("key3"),
		[]byte("key5"),
		[]byte("nonexistent1"),
		[]byte("nonexistent2"),
	}

	deletedCount := store.Delete(toDelete)
	if deletedCount != 3 {
		t.Errorf("Expected 3 deletions, got %d", deletedCount)
	}

	// Verify deleted keys don't exist
	_, exists := store.Get([]byte("key1"))
	if exists {
		t.Error("Expected key1 to be deleted")
	}

	// Verify non-deleted keys still exist
	_, exists = store.Get([]byte("key2"))
	if !exists {
		t.Error("Expected key2 to still exist")
	}
}

func BenchmarkSet(b *testing.B) {
	store := NewInMemoryKVStore()
	defer store.Close()

	key := []byte("benchmark_key")
	value := []byte("benchmark_value")

	for b.Loop() {
		store.Set(key, value, -1)
	}
}

func BenchmarkGet(b *testing.B) {
	store := NewInMemoryKVStore()
	defer store.Close()

	key := []byte("benchmark_key")
	value := []byte("benchmark_value")
	store.Set(key, value, -1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Get(key)
	}
}

func BenchmarkDelete(b *testing.B) {
	store := NewInMemoryKVStore()
	defer store.Close()

	value := []byte("value")
	key := []byte("benchmark_key")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Set(key, value, -1)
		store.Delete([][]byte{key})
	}
}

func BenchmarkConcurrentReadWrite(b *testing.B) {
	store := NewInMemoryKVStore()
	defer store.Close()

	key := []byte("concurrent_key")
	value := []byte("concurrent_value")

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%2 == 0 {
				store.Set(key, value, -1)
			} else {
				store.Get(key)
			}
			i++
		}
	})
}
