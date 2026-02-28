// Package storage provides an in-memory key-value store with support for
// key expiration and background cleanup of expired entries.
package storage

import (
	"context"
	"path"
	"sync"
	"time"

	"github.com/rodrigodotdev/gokv/internal/domain/entity"
)

// Store is a concurrency-safe in-memory key-value store. Each key maps to a
// entity.Item that may carry an optional expiration timestamp.
type Store struct {
	data        map[string]*entity.Item
	mu          sync.RWMutex
	stopCleanup chan struct{}
	stopOnce    sync.Once
}

// NewStore creates a new in-memory key-value store.
// The returned *Store satisfies usecase.KeyValueStore and also
// exposes lifecycle methods (StartCleanup/StopCleanup) directly.
func NewStore() *Store {
	return &Store{
		data:        make(map[string]*entity.Item),
		stopCleanup: make(chan struct{}),
	}
}

// Set stores a key-value pair, overwriting any previous value and removing
// any existing expiration.
func (s *Store) Set(ctx context.Context, key, value string) {
	if ctx.Err() != nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = &entity.Item{
		Value:     value,
		ExpiresAt: nil,
	}
}

// Get returns the value for key and true if the key exists and has not
// expired, or an empty string and false otherwise.
func (s *Store) Get(ctx context.Context, key string) (string, bool) {
	if ctx.Err() != nil {
		return "", false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	item, exists := s.data[key]
	if !exists {
		return "", false
	}

	if item.IsExpired(time.Now().Unix()) {
		return "", false
	}

	return item.Value, true
}

// Del removes the key from the store and returns 1 if the key existed, or 0
// otherwise.
func (s *Store) Del(ctx context.Context, key string) int {
	if ctx.Err() != nil {
		return 0
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.data[key]; exists {
		delete(s.data, key)
		return 1
	}

	return 0
}

// Expire sets a TTL on the given key. It returns true if the key exists and
// the timeout was set, or false if the key does not exist.
func (s *Store) Expire(ctx context.Context, key string, seconds int) bool {
	if ctx.Err() != nil {
		return false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	item, exists := s.data[key]
	if !exists {
		return false
	}

	expiration := time.Now().Unix() + int64(seconds)
	item.ExpiresAt = &expiration

	return true
}

// TTL returns the remaining time-to-live in seconds for a key. It returns -2
// if the key does not exist (matching Redis semantics), or -1 if the key
// exists but has no associated expiration.
func (s *Store) TTL(ctx context.Context, key string) int64 {
	if ctx.Err() != nil {
		return -2
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now().Unix()

	item, exists := s.data[key]
	if !exists {
		return -2
	}

	if item.IsExpired(now) {
		return -2
	}

	if item.ExpiresAt == nil {
		return -1
	}

	ttl := *item.ExpiresAt - now
	if ttl < 0 {
		return -2
	}

	return ttl
}

// Persist removes the expiration from a key. It returns true only if the key
// exists and had a TTL that was removed. If the key does not exist or has no
// expiration set, it returns false (matching Redis semantics).
func (s *Store) Persist(ctx context.Context, key string) bool {
	if ctx.Err() != nil {
		return false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	item, exists := s.data[key]
	if !exists {
		return false
	}

	if item.ExpiresAt == nil {
		return false
	}

	item.ExpiresAt = nil

	return true
}

// Keys returns all non-expired keys matching the given glob pattern.
func (s *Store) Keys(ctx context.Context, pattern string) []string {
	if ctx.Err() != nil {
		return []string{}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now().Unix()
	var matches []string

	for key, item := range s.data {
		if item.IsExpired(now) {
			continue
		}

		if matchPattern(key, pattern) {
			matches = append(matches, key)
		}
	}

	return matches
}

// Exists reports whether a non-expired entry exists for the given key.
func (s *Store) Exists(ctx context.Context, key string) bool {
	if ctx.Err() != nil {
		return false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	item, exists := s.data[key]
	if !exists {
		return false
	}

	return !item.IsExpired(time.Now().Unix())
}

// Size returns the number of non-expired keys in the store.
func (s *Store) Size(ctx context.Context) int {
	if ctx.Err() != nil {
		return 0
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now().Unix()
	count := 0

	for _, item := range s.data {
		if !item.IsExpired(now) {
			count++
		}
	}

	return count
}

// StartCleanup launches a background goroutine that periodically removes
// expired keys at the given interval.
func (s *Store) StartCleanup(interval time.Duration) {

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.cleanupExpiredItems()
			case <-s.stopCleanup:
				return
			}
		}
	}()
}

// StopCleanup signals the cleanup goroutine to stop.
// Safe to call multiple times.
func (s *Store) StopCleanup() {
	s.stopOnce.Do(func() {
		close(s.stopCleanup)
	})
}

func (s *Store) cleanupExpiredItems() {
	// Pass 1: collect expired keys under read lock.
	s.mu.RLock()
	now := time.Now().Unix()
	var expired []string
	for key, item := range s.data {
		if item.IsExpired(now) {
			expired = append(expired, key)
		}
	}
	s.mu.RUnlock()

	if len(expired) == 0 {
		return
	}

	// Pass 2: delete expired keys under write lock.
	s.mu.Lock()
	defer s.mu.Unlock()
	now = time.Now().Unix()
	for _, key := range expired {
		if item, exists := s.data[key]; exists && item.IsExpired(now) {
			delete(s.data, key)
		}
	}
}

func matchPattern(key, pattern string) bool {
	if pattern == "*" {
		return true
	}

	matched, err := path.Match(pattern, key)
	if err != nil {
		return key == pattern
	}

	return matched
}
