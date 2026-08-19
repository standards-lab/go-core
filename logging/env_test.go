package logging_test

import (
	"testing"

	"github.com/standards-lab/go-core/logging"
)

func TestNewEnv_ComposesNamesFromPrefix(t *testing.T) {
	env := logging.NewEnv("herald")

	for _, tc := range []struct {
		got  string
		want string
	}{
		{env.Level, "HERALD_LOG_LEVEL"},
		{env.Format, "HERALD_LOG_FORMAT"},
	} {
		if tc.got != tc.want {
			t.Errorf("got %q, want %q", tc.got, tc.want)
		}
	}
}

func TestNewEnv_EmptyPrefixReturnsZeroEnv(t *testing.T) {
	if env := logging.NewEnv(""); env != (logging.Env{}) {
		t.Errorf("NewEnv(\"\") = %+v, want the zero Env (overrides disabled)", env)
	}
}
