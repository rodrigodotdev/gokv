package entity

import "testing"

func TestItem_IsExpired(t *testing.T) {
	t.Parallel()

	ptr := func(v int64) *int64 { return &v }

	tests := []struct {
		name string
		item Item
		now  int64
		want bool
	}{
		{
			name: "nil ExpiresAt is never expired",
			item: Item{Value: "val", ExpiresAt: nil},
			now:  1000,
			want: false,
		},
		{
			name: "ExpiresAt in the future is not expired",
			item: Item{Value: "val", ExpiresAt: ptr(2000)},
			now:  1000,
			want: false,
		},
		{
			name: "ExpiresAt equal to now is expired (boundary)",
			item: Item{Value: "val", ExpiresAt: ptr(1000)},
			now:  1000,
			want: true,
		},
		{
			name: "ExpiresAt in the past is expired",
			item: Item{Value: "val", ExpiresAt: ptr(500)},
			now:  1000,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.item.IsExpired(tt.now)
			if got != tt.want {
				t.Errorf("IsExpired(%d) = %v, want %v", tt.now, got, tt.want)
			}
		})
	}
}
