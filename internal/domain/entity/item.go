// Package domain defines the core domain types for the key-value store.
package entity

// Item represents a stored value with an optional expiration time.
type Item struct {
	Value     string
	ExpiresAt *int64
}

// IsExpired reports whether the item has expired relative to the given
// Unix timestamp. Items without an expiration never expire.
func (i *Item) IsExpired(now int64) bool {
	if i.ExpiresAt == nil {
		return false
	}

	return now >= *i.ExpiresAt
}
