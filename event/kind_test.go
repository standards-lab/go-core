package event_test

import (
	"encoding/json"
	"math"
	"strings"
	"testing"

	"github.com/standards-lab/go-core/event"
)

type approved struct {
	By string `json:"by"`
}

var (
	kindApproved = event.Define[approved]("lab.grant.approved")
	kindUnfit    = event.Define[float64]("lab.grant.unfit")
)

func TestCheckType(t *testing.T) {
	for _, typ := range []string{"a", "lab.grant.approved", "com.example-1.v2_x"} {
		if err := event.CheckType(typ); err != nil {
			t.Errorf("CheckType(%q) = %v, want nil", typ, err)
		}
	}
	for _, typ := range []string{"", ".", "lab.", ".lab", "lab..x", "lab.*", "lab.>", "a*b", "lab x", "lab\tx", "lab x", "lab\x00x", "lab\x7fx"} {
		if err := event.CheckType(typ); err == nil {
			t.Errorf("CheckType(%q) = nil, want an error", typ)
		}
	}
}

func TestDefinePanicsOnARejectedType(t *testing.T) {
	for _, typ := range []string{"", "lab..x", "a.*", "lab.>", "lab x", "lab\nx"} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("Define(%q) did not panic", typ)
				}
			}()
			event.Define[approved](typ)
		}()
	}
	if got := kindApproved.Type(); got != "lab.grant.approved" {
		t.Errorf("Type() = %q", got)
	}
}

func TestRaiseQueuesAnUnstampedJSONEvent(t *testing.T) {
	var q event.Queue
	kindApproved.Raise(&q, "grant-1", approved{By: "ada"})
	kindApproved.Raise(&q, "", approved{By: "bea"})
	if err := q.Err(); err != nil {
		t.Fatalf("Err() = %v", err)
	}
	es := q.Events()
	if len(es) != 2 {
		t.Fatalf("queued %d events, want 2", len(es))
	}
	e := es[0]
	if e.Type != "lab.grant.approved" || e.Subject != "grant-1" || e.DataContentType != "application/json" {
		t.Errorf("event = %+v", e)
	}
	if e.ID != "" || e.Source != "" || !e.Time.IsZero() {
		t.Errorf("event is stamped: id %q, source %q, time %v", e.ID, e.Source, e.Time)
	}
	var got approved
	if err := json.Unmarshal(e.Data, &got); err != nil || got.By != "ada" {
		t.Errorf("data = %s (%v), want the JSON of the raised entity", e.Data, err)
	}
	if es[1].Subject != "" || !strings.Contains(string(es[1].Data), "bea") {
		t.Errorf("second event = %+v, want the second raise with no subject", es[1])
	}
}

func TestRaiseHoldsAnEncodeFailure(t *testing.T) {
	var q event.Queue
	kindApproved.Raise(&q, "grant-1", approved{})
	kindUnfit.Raise(&q, "grant-1", math.NaN())
	err := q.Err()
	if err == nil || !strings.Contains(err.Error(), "lab.grant.unfit") {
		t.Fatalf("Err() = %v, want the failed raise's", err)
	}
	if n := len(q.Events()); n != 1 {
		t.Errorf("queued %d events, want the one that encoded", n)
	}
}

func TestQueueEventsReturnsACopy(t *testing.T) {
	var q event.Queue
	kindApproved.Raise(&q, "grant-1", approved{})
	q.Events()[0].Subject = "changed"
	if got := q.Events()[0].Subject; got != "grant-1" {
		t.Errorf("Subject = %q after changing the returned slice, want grant-1", got)
	}
}
