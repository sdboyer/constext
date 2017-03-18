// Package coalext provides facilities for coalescing multiple contexts together
// so that they behave as one.

package coalext

import (
	"context"
	"sync"
	"time"
)

type unionCtx struct {
	head, tail context.Context
	done       chan struct{} // chan closed on cancelFunc() call, or parent done
	mu         sync.Mutex    // protects timer and err
	timer      *time.Timer   // if either parent has a deadline
	err        error         // err set on cancel or timeout
}

func Union(c1, c2 context.Context) (context.Context, context.CancelFunc) {
	uc := &unionCtx{
		head: c1,
		tail: c2,
	}

	if uc.head.Done() == nil && uc.head.Done() == nil {
		return uc, func() { uc.cancel(context.Canceled) }
	}

	uc.done = make(chan struct{})

	if uc.head.Err() != nil {
		uc.err = uc.head.Err()
		return uc, func() { uc.cancel(context.Canceled) }
	}
	if uc.tail.Err() != nil {
		uc.err = uc.tail.Err()
		return uc, func() { uc.cancel(context.Canceled) }
	}

	// If there's a deadline set, make sure we respect it.
	if dl, ok := uc.Deadline(); ok {
		d := dl.Sub(time.Now())
		if d <= 0 {
			uc.cancel(context.DeadlineExceeded)
			return uc, func() { uc.cancel(context.Canceled) }
		}
		uc.timer = time.AfterFunc(d, func() { uc.cancel(context.DeadlineExceeded) })
	}

	go func() {
		select {
		case <-uc.head.Done():
			uc.cancel(uc.head.Err())
		case <-uc.tail.Done():
			uc.cancel(uc.tail.Err())
		}
	}()

	return uc, func() { uc.cancel(context.Canceled) }
}

func (uc *unionCtx) cancel(err error) {
	if err == nil {
		panic("coalext: internal error: missing cancel error")
	}

	uc.mu.Lock()
	if uc.err == nil {
		uc.err = err
		close(uc.done)

		if uc.timer != nil {
			uc.timer.Stop()
			uc.timer = nil
		}
	}

	uc.mu.Unlock()
}

func (uc *unionCtx) Deadline() (time.Time, bool) {
	if deadline, ok := uc.head.Deadline(); ok {
		return deadline, ok
	}
	return uc.tail.Deadline()
}

func (uc *unionCtx) Done() <-chan struct{} {
	return uc.done
}

func (uc *unionCtx) Err() error {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	return uc.err
}

func (uc *unionCtx) Value(key interface{}) interface{} {
	v := uc.head.Value(key)
	if v != nil {
		return v
	}
	return uc.tail.Value(key)
}
