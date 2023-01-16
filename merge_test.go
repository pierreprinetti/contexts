package contexts_test

import (
	"context"
	"testing"
	"time"

	"github.com/pierreprinetti/contexts"
)

func TestMerge(t *testing.T) {
	isChannelBlocking := func(ctx context.Context) bool {
		time.Sleep(time.Millisecond)
		select {
		case <-ctx.Done():
			return false
		default:
			return true
		}
	}

	t.Run("cancels when ctx cancels", func(t *testing.T) {
		ctx1, cancel1 := context.WithCancel(context.Background())
		merged, cancel := contexts.Merge(ctx1, context.Background())
		defer cancel()

		if !isChannelBlocking(merged) {
			t.Errorf("expected merged context to be active, was cancelled (channel not blocking)")
		}
		if err := merged.Err(); err != nil {
			t.Errorf("expected merged context to be active, was cancelled (ctx.Err is not nil: %v)", err)
		}
		cancel1()

		if isChannelBlocking(merged) {
			t.Errorf("expected merged context to be cancelled, was active (channel blocking)")
		}
		if err := merged.Err(); err == nil {
			t.Errorf("expected merged context to be active, was cancelled (ctx.Err is nil)")
		}
	})

	t.Run("cancels when child cancels", func(t *testing.T) {
		ctx2, cancel2 := context.WithCancel(context.Background())
		merged, cancel := contexts.Merge(context.Background(), ctx2)
		defer cancel()

		if !isChannelBlocking(merged) {
			t.Errorf("expected merged context to be active, was cancelled (channel not blocking)")
		}
		if err := merged.Err(); err != nil {
			t.Errorf("expected merged context to be active, was cancelled (ctx.Err is not nil: %v)", err)
		}
		cancel2()

		if isChannelBlocking(merged) {
			t.Errorf("expected merged context to be cancelled, was active (channel blocking)")
		}
		if err := merged.Err(); err == nil {
			t.Errorf("expected merged context to be active, was cancelled (ctx.Err is nil)")
		}
	})

	t.Run("panics when ctx is nil", func(t *testing.T) {
		defer func() {
			if p := recover(); p == nil {
				t.Errorf("panic expected, not detected")
			}
		}()

		_, cancel := contexts.Merge(nil, context.Background())
		defer cancel()
	})

	t.Run("reports the shortest deadline", func(t *testing.T) {
		t.Run("when it's ctx", func(t *testing.T) {
			ctx1, cancel1 := context.WithTimeout(context.Background(), 1*time.Hour)
			ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Hour)
			merged, cancel := contexts.Merge(ctx1, ctx2)
			defer cancel1()
			defer cancel2()
			defer cancel()

			deadline, ok := merged.Deadline()
			if !ok {
				t.Fatalf("expected deadline, not found")
			}

			if deadline.After(time.Now().Add(time.Hour)) {
				t.Errorf("expected merged ctx to return the shortest deadline, found %v", deadline)
			}
		})
		t.Run("when it's child", func(t *testing.T) {
			ctx1, cancel1 := context.WithTimeout(context.Background(), 2*time.Hour)
			ctx2, cancel2 := context.WithTimeout(context.Background(), 1*time.Hour)
			merged, cancel := contexts.Merge(ctx1, ctx2)
			defer cancel1()
			defer cancel2()
			defer cancel()

			deadline, ok := merged.Deadline()
			if !ok {
				t.Fatalf("expected deadline, not found")
			}

			if deadline.After(time.Now().Add(time.Hour)) {
				t.Errorf("expected merged ctx to return the shortest deadline, found %v", deadline)
			}
		})
	})
}
