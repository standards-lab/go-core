package process_test

import (
	"errors"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/standards-lab/go-core/process"
)

// The exit codes are the convention an orchestrator reads, so the tests
// assert the numbers a reporter returns, not the constants' names.

func TestFail_ReportsAndReturnsFailure(t *testing.T) {
	var buf strings.Builder

	code := process.Fail(&buf, "config load failed", errors.New("no such file"))

	if code != 1 {
		t.Errorf("Fail() = %d, want 1", code)
	}
	if got := buf.String(); got != "config load failed: no such file\n" {
		t.Errorf("Fail() wrote %q, want %q", got, "config load failed: no such file\n")
	}
}

func TestUsage_ReportsAndReturnsUsage(t *testing.T) {
	var buf strings.Builder

	code := process.Usage(&buf, "db: unknown command")

	if code != 2 {
		t.Errorf("Usage() = %d, want 2", code)
	}
	if got := buf.String(); got != "db: unknown command\n" {
		t.Errorf("Usage() wrote %q, want the text with a terminated line", got)
	}
}

// The context ends on each signal a terminal or an orchestrator sends, and
// not before; the registration catches the signal, so the test process
// survives it.
func TestSignalContext_CancelsOnSignal(t *testing.T) {
	for _, sig := range []syscall.Signal{syscall.SIGINT, syscall.SIGTERM} {
		t.Run(sig.String(), func(t *testing.T) {
			ctx, stop := process.SignalContext()
			defer stop()

			if err := ctx.Err(); err != nil {
				t.Fatalf("ctx.Err() = %v before any signal, want nil", err)
			}

			self, err := os.FindProcess(os.Getpid())
			if err != nil {
				t.Fatal(err)
			}
			if err := self.Signal(sig); err != nil {
				t.Fatalf("signal self: %v", err)
			}

			select {
			case <-ctx.Done():
			case <-time.After(2 * time.Second):
				t.Fatalf("ctx not cancelled after %v", sig)
			}
		})
	}
}
