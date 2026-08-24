package process_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/standards-lab/go-core/process"
)

func TestFail_ReportsAndReturnsFailure(t *testing.T) {
	var buf strings.Builder

	code := process.Fail(&buf, "config load failed", errors.New("no such file"))

	if code != process.ExitFailure {
		t.Errorf("Fail() = %d, want ExitFailure (%d)", code, process.ExitFailure)
	}
	if got := buf.String(); got != "config load failed: no such file\n" {
		t.Errorf("Fail() wrote %q, want %q", got, "config load failed: no such file\n")
	}
}

func TestUsage_ReportsAndReturnsUsage(t *testing.T) {
	var buf strings.Builder

	code := process.Usage(&buf, "db: unknown command")

	if code != process.ExitUsage {
		t.Errorf("Usage() = %d, want ExitUsage (%d)", code, process.ExitUsage)
	}
	if got := buf.String(); got != "db: unknown command\n" {
		t.Errorf("Usage() wrote %q, want the text with a terminated line", got)
	}
}

func TestExitCodes_HoldTheConvention(t *testing.T) {
	if process.ExitOK != 0 || process.ExitFailure != 1 || process.ExitUsage != 2 {
		t.Errorf("exit codes = %d/%d/%d, want 0/1/2",
			process.ExitOK, process.ExitFailure, process.ExitUsage)
	}
}

func TestSignalContext_CancelsOnStop(t *testing.T) {
	ctx, stop := process.SignalContext()

	if err := ctx.Err(); err != nil {
		t.Fatalf("ctx.Err() = %v before any signal, want nil", err)
	}

	stop()

	select {
	case <-ctx.Done():
	default:
		t.Error("ctx not cancelled after stop()")
	}
}
