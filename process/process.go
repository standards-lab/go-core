package process

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
)

// Exit codes shared by a program's binaries: 0 ok, 1 runtime failure,
// 2 usage error.
const (
	ExitOK      = 0
	ExitFailure = 1
	ExitUsage   = 2
)

// Fail reports a runtime failure as "msg: err" and returns ExitFailure. It
// owns the write-error discard for failures that happen before a logger
// exists.
func Fail(w io.Writer, msg string, err error) int {
	_, _ = fmt.Fprintf(w, "%s: %v\n", msg, err)
	return ExitFailure
}

// Usage reports a usage error, terminating the line, and returns ExitUsage.
// Requested help routes through it too — one path for usage output, the Go
// toolchain's own convention.
func Usage(w io.Writer, text string) int {
	_, _ = fmt.Fprintln(w, text)
	return ExitUsage
}

// SignalContext returns a context cancelled on SIGINT or SIGTERM, and the
// stop function that releases the signal registration.
func SignalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
}
