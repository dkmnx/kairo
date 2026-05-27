package execution

import (
	"context"
	"os"
	"runtime"
	"syscall"
	"testing"
	"time"
)

func TestStartSession_CancelOnSignal(t *testing.T) {
	parent := context.Background()
	ctx, cancel, stop := StartSession(parent)
	defer cancel()
	defer stop()

	if ctx == nil {
		t.Fatal("StartSession returned nil context")
	}
	if cancel == nil {
		t.Fatal("StartSession returned nil cancel")
	}
	if stop == nil {
		t.Fatal("StartSession returned nil stop")
	}

	if err := ctx.Err(); err != nil {
		t.Fatalf("new session context already canceled: %v", err)
	}
}

func TestStartSession_StopCleansUp(t *testing.T) {
	parent := context.Background()
	ctx, cancel, stop := StartSession(parent)
	defer cancel()

	stop()

	select {
	case <-time.After(100 * time.Millisecond):
	case <-ctx.Done():
		t.Fatal("context should not be canceled after stop without signal")
	}
}

func TestStartSession_SignalCancelsContext(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sending signals to self is not supported on Windows")
	}
	parent := context.Background()
	ctx, cancel, stop := StartSession(parent)
	defer cancel()
	defer stop()

	proc, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("finding process: %v", err)
	}

	if err := proc.Signal(syscall.SIGINT); err != nil {
		t.Fatalf("sending SIGINT: %v", err)
	}

	select {
	case <-ctx.Done():
		if err := ctx.Err(); err != context.Canceled {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("context was not canceled after SIGINT")
	}
}

func TestStartSession_ParentCancellation(t *testing.T) {
	parent, parentCancel := context.WithCancel(context.Background())
	ctx, cancel, stop := StartSession(parent)
	defer cancel()
	defer stop()

	parentCancel()

	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("child context was not canceled when parent canceled")
	}
}
