package main

import (
	"flag"
	"testing"
)

func TestEnvToFlag(t *testing.T) {
	testCases := []struct {
		name     string
		env      map[string]string
		expected string
	}{
		{
			name:     "no env",
			env:      map[string]string{},
			expected: "text",
		},
		{
			name:     "lower case",
			env:      map[string]string{"log_format": "json"},
			expected: "json",
		},
		{
			name:     "upper case precedes lower case",
			env:      map[string]string{"LOG_FORMAT": "json", "log_format": "text"},
			expected: "json",
		},
		{
			name: "KATSUBUSHI_ prefix precedes others",
			env: map[string]string{
				"KATSUBUSHI_LOG_FORMAT": "json",
				"LOG_FORMAT":            "text",
				"log_format":            "text",
			},
			expected: "json",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			var v string
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			fs.StringVar(&v, "log-format", "text", "")
			fs.VisitAll(envToFlag)
			if v != tc.expected {
				t.Errorf("expected %q got %q", tc.expected, v)
			}
		})
	}
}

func TestApplyEnvToFlagInvalidValue(t *testing.T) {
	t.Setenv("KATSUBUSHI_PORT", "abc")
	var port int
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.IntVar(&port, "port", 11212, "")
	var gotErr error
	fs.VisitAll(func(f *flag.Flag) {
		if err := applyEnvToFlag(f); err != nil {
			gotErr = err
		}
	})
	if gotErr == nil {
		t.Error("applyEnvToFlag must return an error for an invalid value")
	}
}

func TestApplyEnvToFlagIgnoresInvalidLegacyValue(t *testing.T) {
	// Platforms often set non-prefixed variables like PORT for their own
	// purposes. An unparseable value is likely such a collision, so it
	// must be ignored with a warning, not abort the startup.
	t.Setenv("PORT", "not-a-port")
	var port int
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.Var(newPortValue(&port, 11212), "port", "")
	fs.VisitAll(func(f *flag.Flag) {
		if err := applyEnvToFlag(f); err != nil {
			t.Errorf("applyEnvToFlag must not return an error for an invalid legacy value: %v", err)
		}
	})
	if port != 11212 {
		t.Errorf("port must keep the default value, got %d", port)
	}
}

func TestPortValue(t *testing.T) {
	testCases := []struct {
		value    string
		expected int
		isError  bool
	}{
		{value: "11212", expected: 11212},
		{value: "0", expected: 0},
		// Kubernetes and Docker service links inject URLs into *_PORT
		{value: "tcp://10.0.0.1:11212", expected: 11212},
		{value: "tcp://10.0.0.1", isError: true},
		{value: "tcp://:invalid", isError: true},
		{value: "not-a-port", isError: true},
		{value: "1121a", isError: true},
	}
	for _, tc := range testCases {
		t.Run(tc.value, func(t *testing.T) {
			var port int
			v := newPortValue(&port, 8080)
			err := v.Set(tc.value)
			if tc.isError {
				if err == nil {
					t.Errorf("Set(%q) must return an error", tc.value)
				}
				return
			}
			if err != nil {
				t.Errorf("Set(%q) must not return an error: %v", tc.value, err)
			}
			if port != tc.expected {
				t.Errorf("Set(%q) must set %d, got %d", tc.value, tc.expected, port)
			}
		})
	}
}

func TestPortValueFromEnv(t *testing.T) {
	// A Kubernetes service named "katsubushi" injects
	// KATSUBUSHI_PORT=tcp://... into pods by service links.
	t.Setenv("KATSUBUSHI_PORT", "tcp://10.0.0.1:11212")
	var port int
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.Var(newPortValue(&port, 0), "port", "")
	fs.VisitAll(func(f *flag.Flag) {
		if err := applyEnvToFlag(f); err != nil {
			t.Errorf("applyEnvToFlag must not return an error for a URL value: %v", err)
		}
	})
	if port != 11212 {
		t.Errorf("port must be 11212, got %d", port)
	}
}

func TestApplyEnvToFlagSkipsVersion(t *testing.T) {
	t.Setenv("KATSUBUSHI_VERSION", "true")
	t.Setenv("VERSION", "2.3.0")
	var showVersion bool
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.BoolVar(&showVersion, "version", false, "")
	fs.VisitAll(func(f *flag.Flag) {
		if err := applyEnvToFlag(f); err != nil {
			t.Errorf("applyEnvToFlag must ignore environment variables for -version: %v", err)
		}
	})
	if showVersion {
		t.Error("-version must not be settable via environment variables")
	}
}
