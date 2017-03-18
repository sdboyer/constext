package constext

import (
	"context"
	"runtime"
	"testing"
	"time"
)

func TestConsCancel(t *testing.T) {
	c1, cancel1 := context.WithCancel(context.Background())
	c2, cancel2 := context.WithCancel(context.Background())

	uc, _ := Cons(c1, c2)
	if _, has := uc.Deadline(); has {
		t.Fatal("coalesce ctx should not have a deadline if parents do not")
	}

	cancel1()
	select {
	case <-uc.Done():
	case <-time.After(1 * time.Second):
		buf := make([]byte, 10<<10)
		n := runtime.Stack(buf, true)
		t.Fatalf("timed out waiting for parent to quit; stacks:\n%s", buf[:n])
	}

	uc, _ = Cons(c1, c2)
	if uc.Err() == nil {
		t.Fatal("pre-canceled (c1) coalesced context did not begin canceled")
	}

	uc, _ = Cons(c2, c1)
	if uc.Err() == nil {
		t.Fatal("pre-canceled (c2) coalesced context did not begin canceled")
	}

	c3, _ := context.WithCancel(context.Background())
	uc, _ = Cons(c3, c2)
	cancel2()
	select {
	case <-uc.Done():
	case <-time.After(1 * time.Second):
		buf := make([]byte, 10<<10)
		n := runtime.Stack(buf, true)
		t.Fatalf("timed out waiting for second parent to quit; stacks:\n%s", buf[:n])
	}
}

func TestCancelPassdown(t *testing.T) {
	c1, cancel1 := context.WithCancel(context.Background())
	c2, _ := context.WithCancel(context.Background())
	uc, _ := Cons(c1, c2)
	c3, _ := context.WithCancel(uc)

	cancel1()
	select {
	case <-c3.Done():
	case <-time.After(1 * time.Second):
		buf := make([]byte, 10<<10)
		n := runtime.Stack(buf, true)
		t.Fatalf("timed out waiting for parent to quit; stacks:\n%s", buf[:n])
	}

	c1, cancel1 = context.WithCancel(context.Background())
	uc, _ = Cons(c1, c2)
	c3 = context.WithValue(uc, "foo", "bar")

	cancel1()
	select {
	case <-c3.Done():
	case <-time.After(1 * time.Second):
		buf := make([]byte, 10<<10)
		n := runtime.Stack(buf, true)
		t.Fatalf("timed out waiting for parent to quit; stacks:\n%s", buf[:n])
	}
}

func TestValueUnion(t *testing.T) {
	c1 := context.WithValue(context.Background(), "foo", "bar")
	c2 := context.WithValue(context.Background(), "foo", "baz")
	uc, _ := Cons(c1, c2)

	v := uc.Value("foo")
	if v != "bar" {
		t.Fatalf("wanted value of \"foo\" from first union member, \"bar\", got %q", v)
	}

	c3 := context.WithValue(context.Background(), "bar", "quux")
	uc2, _ := Cons(c1, c3)
	v = uc2.Value("bar")
	if v != "quux" {
		t.Fatalf("wanted value from c2, \"quux\", got %q", v)
	}

	uc, _ = Cons(uc, c3)
	v = uc.Value("bar")
	if v != "quux" {
		t.Fatalf("wanted value from nested c2, \"quux\", got %q", v)
	}
}
