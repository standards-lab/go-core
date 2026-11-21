package reactor_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/standards-lab/go-core/reactor"
)

func acquire(ctx context.Context, take func(context.Context) (func(), error)) <-chan func() {
	got := make(chan func(), 1)
	go func() {
		release, err := take(ctx)
		if err != nil {
			release = nil
		}
		got <- release
	}()
	return got
}

func waiting(t *testing.T, got <-chan func(), what string) {
	t.Helper()
	settle()
	select {
	case <-got:
		t.Fatalf("%s acquired the gate, want it waiting", what)
	default:
	}
}

func hold(t *testing.T, take func(context.Context) (func(), error)) func() {
	t.Helper()
	release, err := take(context.Background())
	if err != nil {
		t.Fatalf("acquire an open gate: %v", err)
	}
	return release
}

func TestGate_SharedHoldersCoexist(t *testing.T) {
	var g reactor.Gate
	a := hold(t, g.Shared)
	b := hold(t, g.Shared)
	a()
	b()
}

func TestGate_ExclusiveWaitsForShared(t *testing.T) {
	var g reactor.Gate
	a := hold(t, g.Shared)
	b := hold(t, g.Shared)
	ex := acquire(context.Background(), g.Exclusive)
	waiting(t, ex, "exclusive under two shared holders")
	a()
	waiting(t, ex, "exclusive under one shared holder")
	b()
	recvOrFail(t, ex, "exclusive once the shared holders released")()
}

func TestGate_SharedWaitsForExclusive(t *testing.T) {
	var g reactor.Gate
	release := hold(t, g.Exclusive)
	sh := acquire(context.Background(), g.Shared)
	waiting(t, sh, "shared under an exclusive holder")
	release()
	recvOrFail(t, sh, "shared once the exclusive holder released")()

	release = hold(t, g.Exclusive)
	ex := acquire(context.Background(), g.Exclusive)
	waiting(t, ex, "a second exclusive under the first")
	release()
	recvOrFail(t, ex, "the second exclusive once the first released")()
}

func TestGate_WaitingExclusiveBarsShared(t *testing.T) {
	var g reactor.Gate
	a := hold(t, g.Shared)
	ex := acquire(context.Background(), g.Exclusive)
	waiting(t, ex, "exclusive under a shared holder")
	sh := acquire(context.Background(), g.Shared)
	waiting(t, sh, "a new shared holder behind a waiting exclusive")
	a()
	release := recvOrFail(t, ex, "exclusive once the shared holder released")
	waiting(t, sh, "shared under the exclusive holder")
	release()
	recvOrFail(t, sh, "shared once the exclusive holder released")()
}

func TestGate_CancelledExclusiveAdmitsShared(t *testing.T) {
	var g reactor.Gate
	a := hold(t, g.Shared)
	defer a()
	ctx, cancel := context.WithCancel(context.Background())
	exErr := make(chan error, 1)
	go func() {
		_, err := g.Exclusive(ctx)
		exErr <- err
	}()
	settle()
	sh := acquire(context.Background(), g.Shared)
	waiting(t, sh, "shared behind a waiting exclusive")
	cancel()
	if err := recvOrFail(t, exErr, "the cancelled exclusive"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Exclusive = %v, want the context's cancellation", err)
	}
	recvOrFail(t, sh, "shared once the exclusive withdrew")()
}

func TestGate_CancelledSharedHoldsNothing(t *testing.T) {
	var g reactor.Gate
	release := hold(t, g.Exclusive)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := g.Shared(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Shared = %v, want the deadline", err)
	}
	release()
	hold(t, g.Exclusive)()
}

func TestGate_EndedContextAcquiresNothing(t *testing.T) {
	var g reactor.Gate
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := g.Shared(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("Shared = %v, want the cancellation", err)
	}
	if _, err := g.Exclusive(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("Exclusive = %v, want the cancellation", err)
	}
	hold(t, g.Exclusive)()
}

func TestGate_ReleaseIsIdempotent(t *testing.T) {
	var g reactor.Gate
	a := hold(t, g.Shared)
	b := hold(t, g.Shared)
	a()
	a()
	ex := acquire(context.Background(), g.Exclusive)
	waiting(t, ex, "exclusive while a shared holder remains")
	b()
	recvOrFail(t, ex, "exclusive once the last shared holder released")()
}

func TestGate_ManySharedHolders(t *testing.T) {
	var g reactor.Gate
	releases := make([]func(), 100)
	for i := range releases {
		releases[i] = hold(t, g.Shared)
	}
	ex := acquire(context.Background(), g.Exclusive)
	for _, release := range releases[:len(releases)-1] {
		release()
	}
	waiting(t, ex, "exclusive while one of many shared holders remains")
	releases[len(releases)-1]()
	recvOrFail(t, ex, "exclusive once every shared holder released")()
}

func TestGate_ExclusiveReleaseIsIdempotent(t *testing.T) {
	var g reactor.Gate
	first := hold(t, g.Exclusive)
	first()
	second := hold(t, g.Exclusive)
	first()
	sh := acquire(context.Background(), g.Shared)
	waiting(t, sh, "shared after a stale exclusive release")
	second()
	recvOrFail(t, sh, "shared once the holding exclusive released")()
}
