# constext

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

Use is simple, and patterned after the `context` package:

```go
cctx, _ := constext.Cons(context.Background(), context.Background())
```

True to the spirit of `cons`, trees of constexts can also be constructed through
repeated calls:

```go
cctx, _ := constext.Cons(context.Background(), context.Background())
cctx2, _ := constext.Cons(context.Background(), cctx)
```

...not that that's a good idea.
