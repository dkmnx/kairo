package execution

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// StartSession sets up a context that cancels on SIGINT/SIGTERM.
// Returns the new context with its cancel func and a stop function
// that restores the previous signal handling state.
func StartSession(parent context.Context) (ctx context.Context, cancel context.CancelFunc, stop func()) {
	ctx, cancel = context.WithCancel(parent)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

	done := make(chan struct{})
	stop = func() {
		signal.Stop(ch)
		close(done)
	}

	go func() {
		select {
		case <-ch:
			signal.Stop(ch)
			cancel()
		case <-done:
			signal.Stop(ch)
		}
	}()

	return ctx, cancel, stop
}
