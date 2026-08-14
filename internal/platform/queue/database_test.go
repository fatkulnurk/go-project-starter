package queue

import (
	"testing"
	"time"
)

func TestBackoff(t *testing.T) {
	cases := []struct {
		attempts int
		want     time.Duration
	}{
		{1, 5 * time.Second},
		{2, 10 * time.Second},
		{3, 20 * time.Second},
		{4, 40 * time.Second},
		{5, 80 * time.Second},
		{10, backoffMax},
		{100, backoffMax},
		{1 << 20, backoffMax},
	}
	for _, tc := range cases {
		got := backoff(tc.attempts)
		if got != tc.want {
			t.Errorf("backoff(%d) = %v, want %v", tc.attempts, got, tc.want)
		}
		if got <= 0 {
			t.Fatalf("backoff(%d) returned non-positive %v", tc.attempts, got)
		}
	}
}
