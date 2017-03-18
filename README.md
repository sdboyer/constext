# coalext

Coalext allows you to coalesce multiple `Context`s together, conjoining
them for all `Context` behaviors:

1. If either parent context is canceled, the coalesced context is canceled.
2. If either parent has a deadline, the coalesced context inherits that same
   deadline. If both have a deadline, it inherits the sooner one.
3. Values from both parents are unioned together. When a key is present in both
   parent trees, the first context supercedes the second.
