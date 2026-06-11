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
