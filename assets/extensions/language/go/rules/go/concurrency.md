# Go concurrency

- go/C-9 (must) — Rule: Give started goroutines an explicit owner and completion
  or shutdown path; observe completion before releasing resources they use.
  Exception: Process-lifetime work may end with its owning process.
- go/C-10 (must) — Rule: Before closing a channel, establish that no sender can
  send again and no other closer can close it. Channels need not be closed.
- go/C-11 (should) — Rule: Keep critical sections limited to the protected
  invariant; move I/O and callbacks outside locks when that invariant permits.
- go/C-35 (must) — Rule: Synchronize shared state when concurrent accesses include a write.
- go/C-36 (should) — Rule: Bound input-driven concurrency and queued work according
  to the operation's resource budget.
- go/C-37 (must) — Rule: Do not copy used locks, wait groups, or other synchronization
  structs that prohibit copying, including through enclosing structs.
