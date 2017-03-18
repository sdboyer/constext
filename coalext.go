// Package constext provides facilities for pairing contexts together so that
// they behave as one.

package constext

import (
	"context"
	"sync"
	"time"
)

type constext struct {
	car, cdr context.Context
	done     chan struct{} // chan closed on cancelFunc() call, or parent done
	mu       sync.Mutex    // protects timer and err
	timer    *time.Timer   // if either parent has a deadline
	err      error         // err set on cancel or timeout
}

func Cons(c1, c2 context.Context) (context.Context, context.CancelFunc) {
	cc := &constext{
		car: c1,
		cdr: c2,
	}

	if cc.car.Done() == nil && cc.car.Done() == nil {
		return cc, func() { cc.cancel(context.Canceled) }
	}

	cc.done = make(chan struct{})

	if cc.car.Err() != nil {
		cc.err = cc.car.Err()
		return cc, func() { cc.cancel(context.Canceled) }
	}
	if cc.cdr.Err() != nil {
		cc.err = cc.cdr.Err()
		return cc, func() { cc.cancel(context.Canceled) }
	}

	// If there's a deadline set, make sure we respect it.
	if dl, ok := cc.Deadline(); ok {
		d := dl.Sub(time.Now())
		if d <= 0 {
			cc.cancel(context.DeadlineExceeded)
			return cc, func() { cc.cancel(context.Canceled) }
		}
		cc.timer = time.AfterFunc(d, func() { cc.cancel(context.DeadlineExceeded) })
	}

	go func() {
		select {
		case <-cc.car.Done():
			cc.cancel(cc.car.Err())
		case <-cc.cdr.Done():
			cc.cancel(cc.cdr.Err())
		}
	}()

	return cc, func() { cc.cancel(context.Canceled) }
}

func (cc *constext) cancel(err error) {
	if err == nil {
		panic("constext: internal error: missing cancel error")
	}

	cc.mu.Lock()
	if cc.err == nil {
		cc.err = err
		close(cc.done)

		if cc.timer != nil {
			cc.timer.Stop()
			cc.timer = nil
		}
	}

	cc.mu.Unlock()
}

func (cc *constext) Deadline() (time.Time, bool) {
	hdeadline, hok := cc.car.Deadline()
	tdeadline, tok := cc.cdr.Deadline()
	if !hok && !tok {
		return time.Time{}, false
	}

	if hok && !tok {
		return hdeadline, true
	}
	if !hok && tok {
		return tdeadline, true
	}

	if hdeadline.Before(tdeadline) {
		return hdeadline, true
	}
	return tdeadline, true
}

func (cc *constext) Done() <-chan struct{} {
	return cc.done
}

func (cc *constext) Err() error {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	return cc.err
}

func (cc *constext) Value(key interface{}) interface{} {
	v := cc.car.Value(key)
	if v != nil {
		return v
	}
	return cc.cdr.Value(key)
}
