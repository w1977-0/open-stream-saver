package retry

import (
	"context"
	"errors"
	"testing"
)

func TestDoRetriesTransientError(t *testing.T) {
	calls := 0
	err := Do(context.Background(), 3, func() (error, bool) {
		calls++
		if calls < 3 {
			return errors.New("temporary"), true
		}
		return nil, false
	})
	if err != nil {
		t.Fatalf("Do returned error: %v", err)
	}
	if calls != 3 {
		t.Fatalf("calls = %d, want 3", calls)
	}
}

func TestDoStopsForPermanentError(t *testing.T) {
	calls := 0
	permanent := errors.New("permanent")
	err := Do(context.Background(), 4, func() (error, bool) {
		calls++
		return permanent, false
	})
	if !errors.Is(err, permanent) {
		t.Fatalf("Do error = %v, want permanent", err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}
