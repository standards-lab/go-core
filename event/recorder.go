package event

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"time"
	"uuid"
)

// Sink writes stamped events into a transaction of type Tx, the one the
// command that raised them runs in. An outbox is the sink a composition
// root builds a recorder over; it writes each event as a row the relay
// publishes after the transaction commits.
type Sink[Tx any] interface {
	Write(ctx context.Context, tx Tx, es ...Event) error
}

// Recorder emits the events a command raises into the command's
// transaction, stamped with the id, source, and time the domain never sets,
// and with the service-wide extension attributes its [Options] set.
// The composition root builds one per service and injects it into each
// domain.
type Recorder[Tx any] struct {
	sink   Sink[Tx]
	source string
	now    func() time.Time
	id     func() string
	ext    map[string]string
}

// Options configures a [Recorder]. The zero value stamps the time from
// time.Now, mints version 7 UUIDs, and stamps no extension attributes.
type Options struct {
	// Clock returns the time a recorder stamps, which the recorder converts
	// to UTC; nil uses time.Now.
	Clock func() time.Time
	// IDs mints an event id; nil mints a version 7 UUID, which orders by
	// time.
	IDs func() string
	// Extensions holds the extension attributes the recorder stamps on every
	// event it emits, keyed by name. When a raise sets the same name, the
	// event carries the raise's value. [NewRecorder] copies the map, so a
	// later change to it leaves the recorder as built. Nil or empty stamps
	// none.
	Extensions map[string]string
}

// NewRecorder builds the recorder that writes through sink and stamps every
// event with source, the service's own URI-reference. Source is one value for
// the whole service, never a replica's, because a consumer deduplicates on
// the source and id together. Each field set in opts replaces that
// default; the zero Options keeps them all. A nil sink, an empty source, or
// an extension name in opts that [Event.Validate] rejects is a wiring defect
// and panics.
func NewRecorder[Tx any](sink Sink[Tx], source string, opts Options) *Recorder[Tx] {
	if sink == nil {
		panic("event: recorder: nil sink")
	}
	if source == "" {
		panic("event: recorder: empty source")
	}
	for _, name := range slices.Sorted(maps.Keys(opts.Extensions)) {
		if err := validName(name); err != nil {
			panic(fmt.Sprintf("event: recorder: Options.Extensions: %v", err))
		}
	}
	r := &Recorder[Tx]{sink: sink, source: source, now: opts.Clock, id: opts.IDs}
	if len(opts.Extensions) > 0 {
		r.ext = maps.Clone(opts.Extensions)
	}
	if r.now == nil {
		r.now = time.Now
	}
	if r.id == nil {
		r.id = func() string { return uuid.NewV7().String() }
	}
	return r
}

// Emit wraps fn, a command's body, into the function a transaction runs, such
// as sqlate's Transact. The body runs on the transaction with a fresh
// [Queue]. When it returns an error, Emit returns it and writes nothing.
// When it succeeds, Emit stamps each queued event with the clock's time in
// UTC and the recorder's extension attributes, the raise's value winning
// where a name repeats, and writes them, in the order raised, through the
// sink into the same transaction, before the caller commits. A failed raise or write fails the
// command, so the transaction rolls back with its state and its events
// together. Emit neither begins nor ends the transaction; the code that
// began it ends it. ctx is the context of the write.
func (r *Recorder[Tx]) Emit[R any](ctx context.Context, fn func(Tx, *Queue) (R, error)) func(Tx) (R, error) {
	return func(tx Tx) (R, error) {
		var zero R
		q := &Queue{}
		out, err := fn(tx, q)
		if err != nil {
			return zero, err
		}
		if err := q.Err(); err != nil {
			return zero, fmt.Errorf("event: raise: %w", err)
		}
		if len(q.events) == 0 {
			return out, nil
		}
		now := r.now().UTC()
		es := make([]Event, len(q.events))
		for i, e := range q.events {
			e.ID, e.Source, e.Time = r.id(), r.source, now
			if r.ext != nil {
				ext := maps.Clone(r.ext)
				maps.Copy(ext, e.Extensions)
				e.Extensions = ext
			}
			if err := e.Validate(); err != nil {
				return zero, err
			}
			es[i] = e
		}
		if err := r.sink.Write(ctx, tx, es...); err != nil {
			return zero, fmt.Errorf("event: emit: %w", err)
		}
		return out, nil
	}
}
