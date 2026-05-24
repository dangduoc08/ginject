package ctx

import (
	"context"
	"fmt"
	"runtime/debug"
)

// ErrHandler is called when a GoSafe goroutine panics.
type ErrHandler func(err error, stack []byte)

// GoSafe launches fn in a new goroutine that inherits ctx.
//
// Guarantees:
//   - Returns without spawning if ctx is already cancelled (zero goroutine leak).
//   - The goroutine receives ctx so fn can select on ctx.Done().
//   - Panics inside fn are recovered; errFn is called with the error and
//     stack trace. If errFn is nil, panics are silently discarded.
func GoSafe(ctx context.Context, fn func(context.Context), errFn ErrHandler) {
	if ctx.Err() != nil {
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil && errFn != nil {
				errFn(
					fmt.Errorf("GoSafe panic: %v", r),
					debug.Stack(),
				)
			}
		}()
		fn(ctx)
	}()
}
