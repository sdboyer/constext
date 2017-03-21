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

// Cons takes two Contexts and combines them into a pair, conjoining their
// behavior:
//
//  - If either parent context is canceled, the constext is canceled. The err is
//  set to whatever the err of the parent that was canceled.
//  - If either parent has a deadline, the constext uses that same deadline. If
//  both have a deadline, it uses the sooner/lesser one.
//  - Values from both parents are unioned together. When a key is present in
//  both parent trees, the left (first) context supercedes the right (second).
//
// All the normal context.With*() funcs should incorporate constexts correctly.
//
// If the two parent contexts both return a nil channel from Done() (which can
// occur if both parents are Background, or were created only through
// context.WithValue()), then the returned cancelFunc() is a no-op; calling it
// will NOT result in the termination of any sub-contexts later created.
func Cons(l, r context.Context) (context.Context, context.CancelFunc) {
	cc := &constext{
		car: l,
		cdr: r,
	}

	if cc.car.Done() == nil && cc.cdr.Done() == nil {
		// Both parents are un-cancelable, so it's more technically correct to
		// return a no-op func here.
		return cc, func() {}
	}

	// Only make a done chan if at least some parents are cancelable.
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
