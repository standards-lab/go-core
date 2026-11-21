package event_test

import (
	"os"
	"testing"
	"time"
	_ "time/tzdata"
)

// See architecture/standards/go-elemental/principles/utc-times.md
func TestMain(m *testing.M) {
	london, err := time.LoadLocation("Europe/London")
	if err != nil {
		panic(err)
	}
	time.Local = london
	os.Exit(m.Run())
}
