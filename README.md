# constext [![Doc Status](https://godoc.org/github.com/sdboyer/constext?status.png)](https://godoc.org/github.com/sdboyer/constext)

constext allows you to [`cons`](https://en.wikipedia.org/wiki/Cons) `Context`s
together as a pair, conjoining them for the purpose of all `Context` behaviors:

1. If either parent context is canceled, the constext is canceled. The
   err is set to whatever the err of the parent that was canceled.
2. If either parent has a deadline, the constext uses that same
   deadline. If both have a deadline, it uses the sooner/lesser one.
3. Values from both parents are unioned together. When a key is present in both
   parent trees, the left (first) context supercedes the right (second).

Paired contexts can be recombined using the standard `context.With*()`
functions.

## Usage

Use is simple, and patterned after the `context` package. The `constext.Cons()`
function takes two `context.Context` arguments and returns a single, unified
one, along with a `context.CancelFunc`.

```go
cctx, cancelFunc := constext.Cons(context.Background(), context.Background())
```

True to the spirit of `cons`, recursive trees can be formed through
nesting:

```go
bg := context.Background()
cctx := constext.Cons(bg, constext.Cons(bg, constext.Cons(bg, bg)))
```

This probably isn't a good idea, but it's possible.

## Rationale

While the unary model of context works well for the original vision - an object
operating within an [HTTP] request's scope - there are times when we need a
little more.

For example: in [dep](https://github.com/golang/dep), the subsystem that
manages interaction with source repositories is called a
[`SourceManager`](https://godoc.org/github.com/sdboyer/gps#SourceManager). It
is a long-lived object; generally, only one is created over the course of any
single `dep` invocation. The `SourceManager` has a number of methods on it that
may initiate network and/or disk interaction. As such, these methods need to
take a `context.Context`, so that the caller can cancel them if needed.

However, this is not sufficient. The `SourceManager` itself may need to be
terminated (e.g., if the process received a signal). In such a case, in-flight
method calls also need to be canceled, to avoid leaving disk in inconsistent
state.

As a result, each in-flight request serves two parents - the initator of the
request, and the `SourceManager` itself. We can abstract away this complexity
by having a `Context` for each, and `Cons`ing them together on a per-call
basis.
