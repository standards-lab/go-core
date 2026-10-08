package lifecycle_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/standards-lab/go-core/graph"
	"github.com/standards-lab/go-core/lifecycle"
)

// source is a Monitored value that reports the failures sent on it.
type source chan error

func (s source) Err() <-chan error { return s }

var _ lifecycle.Monitored = source(nil)

// lazy is a Monitored Starter that creates its channel in Start, as a value
// whose failures begin only once it runs does.
type lazy struct {
	errs chan error
}

func (l *lazy) Start(context.Context) error {
	l.errs = make(chan error, 1)
	return nil
}

func (l *lazy) Err() <-chan error { return l.errs }

// A monitored failure, from a Monitored value or a channel registered with
// Monitor, cancels Exec's function and joins Exec's return wrapped "run:".
func TestExec_MonitorFailureEndsFn(t *testing.T) {
	for _, tc := range []struct {
		name     string
		inferred bool
	}{
		{"Monitored", true},
		{"Monitor", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			errs := make(source, 1)
			g := graph.New()
			var roots []graph.Ref
			if tc.inferred {
				roots = append(roots, value(g, "source", errs))
			}
			lc := coordinator(t, failsafe, g, roots...)
			if !tc.inferred {
				lc.Monitor(errs)
			}

			sentinel := errors.New("consumer exploded")
			fnErr := errors.New("fn gave up")
			sawCancel := false
			err := lc.Exec(context.Background(), func(ctx context.Context) error {
				errs <- sentinel
				select {
				case <-ctx.Done():
					sawCancel = true
				case <-time.After(failsafe):
				}
				return fnErr
			})

			if !sawCancel {
				t.Error("fn's context was not cancelled by the monitored failure")
			}
			if !errors.Is(err, sentinel) || !strings.Contains(err.Error(), "run: consumer exploded") {
				t.Errorf("Exec = %v, want the monitored failure wrapped \"run:\"", err)
			}
			if !errors.Is(err, fnErr) {
				t.Errorf("Exec = %v, want fn's error joined", err)
			}
		})
	}
}

// Without a monitored failure, Exec returns fn's error alone.
func TestExec_NoMonitorFailureReturnsFnError(t *testing.T) {
	g := graph.New()
	src := value(g, "source", make(source))
	fnErr := errors.New("fn failed")
	err := coordinator(t, failsafe, g, src).Exec(context.Background(), func(context.Context) error {
		return fnErr
	})
	if err == nil || err.Error() != fnErr.Error() {
		t.Fatalf("Exec = %v, want only %v", err, fnErr)
	}
}

func TestRun_MonitoredValueFailureEndsRun(t *testing.T) {
	var r recorder
	errs := make(source, 1)
	g := graph.New()
	a := node(g, &fake{name: "a", r: &r})
	src := value(g, "source", errs)
	lc := coordinator(t, failsafe, g, a, src)

	ready := make(chan struct{})
	lc.OnReady(func() { close(ready) })
	done := run(t.Context(), lc)
	recvOrFail(t, ready, "coordinator to become ready")

	sentinel := errors.New("subscription lost")
	errs <- sentinel

	err := recvOrFail(t, done, "Run to return")
	if !errors.Is(err, sentinel) || !strings.Contains(err.Error(), "run: subscription lost") {
		t.Fatalf("Run = %v, want the monitored failure wrapped \"run:\"", err)
	}
	if got := r.list(); !phased(got, []string{"start a"}, []string{"stop a"}) {
		t.Errorf("events = %q, want a started then shut down", got)
	}
}

// A Monitored value's nil error and closed channel are the quiet end of a
// source, and a nil channel never yields: none of them ends the run.
func TestRun_MonitoredValueIgnoresNilCloseAndNilChannel(t *testing.T) {
	errs := make(source)
	g := graph.New()
	src := value(g, "source", errs)
	silent := value(g, "silent", source(nil))
	lc := coordinator(t, failsafe, g, src, silent)

	ready := make(chan struct{})
	lc.OnReady(func() { close(ready) })
	ctx, cancel := context.WithCancel(context.Background())
	done := run(ctx, lc)
	recvOrFail(t, ready, "coordinator to become ready")

	errs <- nil
	close(errs)

	time.Sleep(20 * time.Millisecond)
	if !lc.Ready() {
		t.Fatal("coordinator left running after a nil error, a close, or a nil channel")
	}

	cancel()
	if err := recvOrFail(t, done, "Run to return"); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

// Err is read once startup has completed, so a value may create its
// channel in Start.
func TestRun_MonitoredValueIsReadAfterStartup(t *testing.T) {
	l := &lazy{}
	g := graph.New()
	n := value(g, "lazy", l)
	lc := coordinator(t, failsafe, g, n)

	ready := make(chan struct{})
	lc.OnReady(func() { close(ready) })
	done := run(t.Context(), lc)
	recvOrFail(t, ready, "coordinator to become ready")

	sentinel := errors.New("lazy failed")
	l.errs <- sentinel

	if err := recvOrFail(t, done, "Run to return"); !errors.Is(err, sentinel) {
		t.Fatalf("Run = %v, want the failure on the channel Start created", err)
	}
}
