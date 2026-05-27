package execution

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// StartSession sets up a context that cancels on SIGINT/SIGTERM.
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
