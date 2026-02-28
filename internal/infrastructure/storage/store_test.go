package storage

import (
	"context"
	"sort"
	"testing"
	"time"
)

// expireKey directly sets a key's expiration to secondsAgo seconds in the past.
// This avoids waiting for real time to pass in tests.
func expireKey(s *Store, key string, secondsAgo int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	past := time.Now().Unix() - secondsAgo
	s.data[key].ExpiresAt = &past
}

// ---------------------------------------------------------------------------
// Set / Get
// ---------------------------------------------------------------------------

func TestSetGet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setup     func(ctx context.Context, s *Store)
		key       string
		wantVal   string
		wantFound bool
	}{
		{
			name: "set then get returns the value",
			setup: func(ctx context.Context, s *Store) {
				s.Set(ctx, "greeting", "hello")
			},
			key:       "greeting",
			wantVal:   "hello",
			wantFound: true,
		},
		{
			name:      "get non-existent key returns empty and false",
			setup:     func(ctx context.Context, s *Store) {},
			key:       "missing",
			wantVal:   "",
			wantFound: false,
		},
		{
			name: "set overwrites existing value",
			setup: func(ctx context.Context, s *Store) {
				s.Set(ctx, "key", "first")
				s.Set(ctx, "key", "second")
			},
			key:       "key",
			wantVal:   "second",
			wantFound: true,
		},
		{
			name: "set with cancelled context is a no-op",
			setup: func(_ context.Context, s *Store) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				s.Set(ctx, "key", "value")
			},
			key:       "key",
			wantVal:   "",
			wantFound: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := NewStore()
			ctx := context.Background()

			tc.setup(ctx, s)

			got, found := s.Get(ctx, tc.key)
			if got != tc.wantVal || found != tc.wantFound {
				t.Errorf("Get(%q) = (%q, %v), want (%q, %v)",
					tc.key, got, found, tc.wantVal, tc.wantFound)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Del
// ---------------------------------------------------------------------------

func TestDel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setup     func(ctx context.Context, s *Store)
		key       string
		cancelCtx bool
		want      int
	}{
		{
			name: "del existing key returns 1",
			setup: func(ctx context.Context, s *Store) {
				s.Set(ctx, "key", "value")
			},
			key:  "key",
			want: 1,
		},
		{
			name:  "del non-existent key returns 0",
			setup: func(ctx context.Context, s *Store) {},
			key:   "missing",
			want:  0,
		},
		{
			name: "del with cancelled context returns 0",
			setup: func(ctx context.Context, s *Store) {
				s.Set(ctx, "key", "value")
			},
			key:       "key",
			cancelCtx: true,
			want:      0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := NewStore()
			ctx := context.Background()

			tc.setup(ctx, s)

			delCtx := ctx
			if tc.cancelCtx {
				var cancel context.CancelFunc
				delCtx, cancel = context.WithCancel(ctx)
				cancel()
			}

			got := s.Del(delCtx, tc.key)
			if got != tc.want {
				t.Errorf("Del(%q) = %d, want %d", tc.key, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Expire / TTL
// ---------------------------------------------------------------------------

func TestExpire(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(ctx context.Context, s *Store)
		key   string
		want  bool
	}{
		{
			name: "expire on existing key returns true",
			setup: func(ctx context.Context, s *Store) {
				s.Set(ctx, "key", "value")
			},
			key:  "key",
			want: true,
		},
		{
			name:  "expire on non-existent key returns false",
			setup: func(ctx context.Context, s *Store) {},
			key:   "missing",
			want:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := NewStore()
			ctx := context.Background()

			tc.setup(ctx, s)

			got := s.Expire(ctx, tc.key, 60)
			if got != tc.want {
				t.Errorf("Expire(%q, 60) = %v, want %v", tc.key, got, tc.want)
			}
		})
	}
}

func TestTTL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(ctx context.Context, s *Store)
		key   string
		check func(t *testing.T, ttl int64)
	}{
		{
			name: "returns remaining seconds for key with expiration",
			setup: func(ctx context.Context, s *Store) {
				s.Set(ctx, "key", "value")
				s.Expire(ctx, "key", 300)
			},
			key: "key",
			check: func(t *testing.T, ttl int64) {
				t.Helper()
				// Should be close to 300 (allow a few seconds of slack).
				if ttl < 295 || ttl > 300 {
					t.Errorf("TTL = %d, want ~300", ttl)
				}
			},
		},
		{
			name: "returns -1 for key without expiration",
			setup: func(ctx context.Context, s *Store) {
				s.Set(ctx, "key", "value")
			},
			key: "key",
			check: func(t *testing.T, ttl int64) {
				t.Helper()
				if ttl != -1 {
					t.Errorf("TTL = %d, want -1", ttl)
				}
			},
		},
		{
			name:  "returns -2 for non-existent key",
			setup: func(ctx context.Context, s *Store) {},
			key:   "missing",
			check: func(t *testing.T, ttl int64) {
				t.Helper()
				if ttl != -2 {
					t.Errorf("TTL = %d, want -2", ttl)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := NewStore()
			ctx := context.Background()

			tc.setup(ctx, s)

			ttl := s.TTL(ctx, tc.key)
			tc.check(t, ttl)
		})
	}
}

func TestGetExpiredKey(t *testing.T) {
	t.Parallel()

	s := NewStore()
	ctx := context.Background()

	s.Set(ctx, "key", "value")

	expireKey(s, "key", 10)

	val, found := s.Get(ctx, "key")
	if val != "" || found != false {
		t.Errorf("Get(expired key) = (%q, %v), want (\"\", false)", val, found)
	}
}

// ---------------------------------------------------------------------------
// Persist
// ---------------------------------------------------------------------------

func TestPersist(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setup     func(ctx context.Context, s *Store)
		key       string
		cancelCtx bool
		want      bool
	}{
		{
			name: "removes TTL and returns true",
			setup: func(ctx context.Context, s *Store) {
				s.Set(ctx, "key", "value")
				s.Expire(ctx, "key", 60)
			},
			key:  "key",
			want: true,
		},
		{
			name: "key without TTL returns false",
			setup: func(ctx context.Context, s *Store) {
				s.Set(ctx, "key", "value")
			},
			key:  "key",
			want: false,
		},
		{
			name:  "non-existent key returns false",
			setup: func(ctx context.Context, s *Store) {},
			key:   "missing",
			want:  false,
		},
		{
			name: "cancelled context returns false",
			setup: func(ctx context.Context, s *Store) {
				s.Set(ctx, "key", "value")
				s.Expire(ctx, "key", 60)
			},
			key:       "key",
			cancelCtx: true,
			want:      false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := NewStore()
			ctx := context.Background()

			tc.setup(ctx, s)

			persistCtx := ctx
			if tc.cancelCtx {
				var cancel context.CancelFunc
				persistCtx, cancel = context.WithCancel(ctx)
				cancel()
			}

			got := s.Persist(persistCtx, tc.key)
			if got != tc.want {
				t.Errorf("Persist(%q) = %v, want %v", tc.key, got, tc.want)
			}
		})
	}
}

func TestPersistRemovesTTL(t *testing.T) {
	t.Parallel()

	s := NewStore()
	ctx := context.Background()

	s.Set(ctx, "key", "value")
	s.Expire(ctx, "key", 300)

	// Sanity: TTL should be positive.
	if ttl := s.TTL(ctx, "key"); ttl <= 0 {
		t.Fatalf("precondition: TTL = %d, want > 0", ttl)
	}

	s.Persist(ctx, "key")

	if ttl := s.TTL(ctx, "key"); ttl != -1 {
		t.Errorf("after Persist: TTL = %d, want -1", ttl)
	}
}

// ---------------------------------------------------------------------------
// Keys
// ---------------------------------------------------------------------------

func TestKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(ctx context.Context, s *Store)
		pattern string
		want    []string
	}{
		{
			name: "wildcard returns all non-expired keys",
			setup: func(ctx context.Context, s *Store) {
				s.Set(ctx, "a", "1")
				s.Set(ctx, "b", "2")
				s.Set(ctx, "c", "3")
			},
			pattern: "*",
			want:    []string{"a", "b", "c"},
		},
		{
			name: "pattern filters correctly",
			setup: func(ctx context.Context, s *Store) {
				s.Set(ctx, "user:1", "alice")
				s.Set(ctx, "user:2", "bob")
				s.Set(ctx, "session:1", "s1")
			},
			pattern: "user:*",
			want:    []string{"user:1", "user:2"},
		},
		{
			name: "excludes expired keys",
			setup: func(ctx context.Context, s *Store) {
				s.Set(ctx, "alive", "yes")
				s.Set(ctx, "dead", "no")
				expireKey(s, "dead", 10)
			},
			pattern: "*",
			want:    []string{"alive"},
		},
		{
			name:    "empty store returns empty slice",
			setup:   func(ctx context.Context, s *Store) {},
			pattern: "*",
			want:    []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := NewStore()
			ctx := context.Background()

			tc.setup(ctx, s)

			got := s.Keys(ctx, tc.pattern)
			if got == nil {
				got = []string{}
			}

			sort.Strings(got)
			sort.Strings(tc.want)

			if len(got) != len(tc.want) {
				t.Fatalf("Keys(%q) returned %d keys %v, want %d keys %v",
					tc.pattern, len(got), got, len(tc.want), tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("Keys(%q)[%d] = %q, want %q",
						tc.pattern, i, got[i], tc.want[i])
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Exists
// ---------------------------------------------------------------------------

func TestExists(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(ctx context.Context, s *Store)
		key   string
		want  bool
	}{
		{
			name: "returns true for existing key",
			setup: func(ctx context.Context, s *Store) {
				s.Set(ctx, "key", "value")
			},
			key:  "key",
			want: true,
		},
		{
			name:  "returns false for non-existent key",
			setup: func(ctx context.Context, s *Store) {},
			key:   "missing",
			want:  false,
		},
		{
			name: "returns false for expired key",
			setup: func(ctx context.Context, s *Store) {
				s.Set(ctx, "key", "value")
				expireKey(s, "key", 10)
			},
			key:  "key",
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := NewStore()
			ctx := context.Background()

			tc.setup(ctx, s)

			got := s.Exists(ctx, tc.key)
			if got != tc.want {
				t.Errorf("Exists(%q) = %v, want %v", tc.key, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Size
// ---------------------------------------------------------------------------

func TestSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(ctx context.Context, s *Store)
		want  int
	}{
		{
			name:  "empty store",
			setup: func(ctx context.Context, s *Store) {},
			want:  0,
		},
		{
			name: "counts only non-expired keys",
			setup: func(ctx context.Context, s *Store) {
				s.Set(ctx, "a", "1")
				s.Set(ctx, "b", "2")
				s.Set(ctx, "expired", "3")
				expireKey(s, "expired", 10)
			},
			want: 2,
		},
		{
			name: "all keys alive",
			setup: func(ctx context.Context, s *Store) {
				s.Set(ctx, "x", "1")
				s.Set(ctx, "y", "2")
				s.Set(ctx, "z", "3")
			},
			want: 3,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := NewStore()
			ctx := context.Background()

			tc.setup(ctx, s)

			got := s.Size(ctx)
			if got != tc.want {
				t.Errorf("Size() = %d, want %d", got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// StartCleanup / StopCleanup
// ---------------------------------------------------------------------------

func TestCleanupRemovesExpiredKeys(t *testing.T) {
	t.Parallel()

	s := NewStore()
	ctx := context.Background()

	s.Set(ctx, "keep", "yes")
	s.Set(ctx, "remove", "no")

	expireKey(s, "remove", 1)

	// Start cleanup with a short interval (50ms).
	s.StartCleanup(50)
	defer s.StopCleanup()

	// Wait enough time for at least one cleanup cycle.
	time.Sleep(200 * time.Millisecond)

	// The expired key should have been physically deleted from the map.
	s.mu.RLock()
	_, stillExists := s.data["remove"]
	s.mu.RUnlock()

	if stillExists {
		t.Error("expired key 'remove' still in map after cleanup")
	}

	// The live key should remain.
	val, found := s.Get(ctx, "keep")
	if !found || val != "yes" {
		t.Errorf("Get(keep) = (%q, %v), want (\"yes\", true)", val, found)
	}
}

func TestCleanupWithShortTTL(t *testing.T) {
	t.Parallel()

	s := NewStore()
	ctx := context.Background()

	s.Set(ctx, "temp", "data")
	// Set TTL to 1 second.
	s.Expire(ctx, "temp", 1)

	// Start cleanup with 50ms interval.
	s.StartCleanup(50)
	defer s.StopCleanup()

	// Wait for the key to expire and cleanup to run.
	time.Sleep(1500 * time.Millisecond)

	s.mu.RLock()
	_, stillExists := s.data["temp"]
	s.mu.RUnlock()

	if stillExists {
		t.Error("key 'temp' should have been cleaned up after TTL expiration")
	}
}
