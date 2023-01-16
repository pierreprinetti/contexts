package contexts

import (
	"context"
	"sync"
	"time"
)

type mergeContext struct {
	child  context.Context
	parent context.Context
	ch     chan struct{}

	sync.Mutex
	err error
}

func idempotentlyClose(ch chan struct{}) {
	select {
	case <-ch:
	default:
		close(ch)
	}
}

// Merge returns a copy of ctx that:
// * returns ctx values if the child's are nil
// * is done when either of the two is done
//
// Merge panics if ctx is nil.
func Merge(ctx, child context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		panic("ctx must not be nil")
	}
	if child == nil {
		return context.WithCancel(ctx)
	}

	ch := make(chan struct{})
	cancelCh := make(chan struct{})

	go func() {
		select {
		case <-child.Done():
		case <-ctx.Done():
		case <-cancelCh:
		}
		close(ch)
	}()

	return &mergeContext{
		child:  child,
		parent: ctx,
		ch:     ch,
	}, func() { idempotentlyClose(cancelCh) }
}

// Value returns the child's value for the key, or the parent's if nil.
func (ctx *mergeContext) Value(key interface{}) interface{} {
	if v := ctx.child.Value(key); v != nil {
		return v
	}
	return ctx.parent.Value(key)
}

// Done returns a channel that is closed when either the child's or the
// parent's is.
func (ctx *mergeContext) Done() <-chan struct{} {
	return ctx.ch
}

// Err returns child's Err(), or the parent's if nil. After Err returns a
// non-nil error, successive calls to Err return the same error.
func (ctx *mergeContext) Err() error {
	ctx.Lock()
	defer ctx.Unlock()

	if ctx.err == nil {
		if err := ctx.child.Err(); err != nil {
			ctx.err = err
		} else {
			ctx.err = ctx.parent.Err()
		}
	}
	return ctx.err
}

// Deadline returns the closest deadline, if any.
func (ctx *mergeContext) Deadline() (deadline time.Time, ok bool) {
	if d1, ok := ctx.child.Deadline(); ok {
		if d2, ok := ctx.parent.Deadline(); ok {
			if d1.Before(d2) {
				return d1, ok
			} else {
				return d2, ok
			}
		}
		return d1, ok
	}
	return ctx.parent.Deadline()
}
