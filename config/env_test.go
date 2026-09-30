package config_test

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/standards-lab/go-core/config"
)

func TestEnvName(t *testing.T) {
	cases := []struct {
		name   string
		prefix string
		parts  []string
		want   string
	}{
		{"prefix and parts", "app", []string{"db", "host"}, "APP_DB_HOST"},
		{"prefix only", "app", nil, "APP"},
		{"empty part dropped", "app", []string{"", "host"}, "APP_HOST"},
		{"whitespace trimmed", "app", []string{"  db  ", "host"}, "APP_DB_HOST"},
		{"empty prefix composes no name", "", []string{"db"}, ""},
		{"separator-only prefix composes no name", " - ", []string{"db"}, ""},
		{"all empty", "", nil, ""},
		{"space becomes separator", "app", []string{"db host"}, "APP_DB_HOST"},
		{"hyphen becomes separator", "app", []string{"read-timeout"}, "APP_READ_TIMEOUT"},
		{"prefix separators trimmed", "-app-", []string{"x"}, "APP_X"},
		{"separator run collapses", "app", []string{"a--b"}, "APP_A_B"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := config.EnvName(tc.prefix, tc.parts...); got != tc.want {
				t.Errorf("EnvName(%q, %v) = %q, want %q", tc.prefix, tc.parts, got, tc.want)
			}
		})
	}
}

func TestSetDurationFromEnv(t *testing.T) {
	const name = "TEST_SET_DURATION"

	t.Run("set value overrides", func(t *testing.T) {
		t.Setenv(name, "30s")
		var dest *config.Duration
		if err := config.SetDurationFromEnv(&dest, name); err != nil {
			t.Fatalf("SetDurationFromEnv: %v", err)
		}
		if dest == nil || dest.Duration() != 30*time.Second {
			t.Errorf("dest = %v, want 30s", dest)
		}
	})

	t.Run("unset name is a no-op", func(t *testing.T) {
		prior := config.Duration(time.Minute)
		dest := &prior
		if err := config.SetDurationFromEnv(&dest, name); err != nil {
			t.Fatalf("SetDurationFromEnv: %v", err)
		}
		if dest != &prior {
			t.Error("dest reassigned on unset variable, want untouched")
		}
	})

	t.Run("empty name is a no-op", func(t *testing.T) {
		var dest *config.Duration
		if err := config.SetDurationFromEnv(&dest, ""); err != nil {
			t.Fatalf("SetDurationFromEnv: %v", err)
		}
		if dest != nil {
			t.Errorf("dest = %v with no variable named, want nil", dest)
		}
	})

	t.Run("malformed value fails and names the variable", func(t *testing.T) {
		t.Setenv(name, "not-a-duration")
		var dest *config.Duration
		err := config.SetDurationFromEnv(&dest, name)
		if err == nil {
			t.Fatal("SetDurationFromEnv accepted a malformed duration")
		}
		if got := err.Error(); !strings.Contains(got, name) {
			t.Errorf("error = %q, want it to name %s", got, name)
		}
		if dest != nil {
			t.Errorf("dest = %v after a failed parse, want nil", dest)
		}
	})
}

func TestSetFromEnv(t *testing.T) {
	const name = "TEST_SET_FROM_ENV"

	t.Run("set value parses", func(t *testing.T) {
		t.Setenv(name, "42")
		var dest *int
		if err := config.SetFromEnv(&dest, name, strconv.Atoi); err != nil {
			t.Fatalf("SetFromEnv: %v", err)
		}
		if dest == nil || *dest != 42 {
			t.Errorf("dest = %v, want 42", dest)
		}
	})

	t.Run("unset or empty name is a no-op", func(t *testing.T) {
		prior := 7
		dest := &prior
		for _, n := range []string{name, ""} {
			if err := config.SetFromEnv(&dest, n, strconv.Atoi); err != nil {
				t.Fatalf("SetFromEnv(%q): %v", n, err)
			}
		}
		if dest != &prior {
			t.Error("dest reassigned with nothing to apply, want untouched")
		}
	})

	t.Run("parse failure names the variable", func(t *testing.T) {
		t.Setenv(name, "many")
		var dest *int
		err := config.SetFromEnv(&dest, name, strconv.Atoi)
		if err == nil || !strings.Contains(err.Error(), name) {
			t.Fatalf("error = %v, want a parse failure naming %s", err, name)
		}
		if dest != nil {
			t.Errorf("dest = %v after a failed parse, want nil", dest)
		}
	})
}
