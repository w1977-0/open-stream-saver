package progress

import (
	"io"
	"testing"
)

// The bar renders to io.Discard, so these tests make no terminal output and
// assert only on the counting rules the download workers depend on.

func TestAddCountsOnlyPositiveIncrements(t *testing.T) {
	tracker := New(100, "test", io.Discard)
	tracker.Add(10)
	if got := tracker.bar.Current(); got != 10 {
		t.Fatalf("after Add(10): current = %d, want 10", got)
	}

	tracker.Add(0)
	tracker.Add(-5)
	if got := tracker.bar.Current(); got != 10 {
		t.Fatalf("non-positive Add must be ignored: current = %d, want 10", got)
	}
	tracker.Complete()
}

func TestCompleteIsSafeToCallMoreThanOnce(t *testing.T) {
	tracker := New(100, "test", io.Discard)
	tracker.Add(4)
	tracker.Complete()
	// Workers finish at different times; the second call must return at once
	// rather than blocking on a pool that has already been waited on.
	tracker.Complete()
}
