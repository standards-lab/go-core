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

// Fail writes "msg: err" to w and returns ExitFailure.
func Fail(w io.Writer, msg string, err error) int {
	_, _ = fmt.Fprintf(w, "%s: %v\n", msg, err)
	return ExitFailure
}

// Usage writes text as a line to w and returns ExitUsage; help goes through it too.
func Usage(w io.Writer, text string) int {
	_, _ = fmt.Fprintln(w, text)
	return ExitUsage
}

// SignalContext returns a context cancelled on SIGINT or SIGTERM, and the
// stop function that releases the signal registration. Call stop once the
// context is done, so a second signal terminates the process at once.
func SignalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
}
