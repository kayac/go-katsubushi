package katsubushi

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestLogHandlerText(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(newLogHandler(&buf, slog.LevelInfo, "text"))
	logger.Info("hello", "worker_id", uint64(5))

	line := buf.String()
	if !strings.Contains(line, "hello") {
		t.Errorf("message not found in log: %s", line)
	}
	if !strings.Contains(line, "worker_id=5") {
		t.Errorf("attribute not found in log: %s", line)
	}
	if !strings.Contains(line, "INFO") {
		t.Errorf("level not found in log: %s", line)
	}
}

func TestLogHandlerJSON(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(newLogHandler(&buf, slog.LevelInfo, "json"))
	logger.Info("hello", "worker_id", uint64(5))

	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("failed to parse log as JSON: %s: %s", err, buf.String())
	}
	if m["msg"] != "hello" {
		t.Errorf("unexpected msg: %v", m["msg"])
	}
	if m["level"] != "INFO" {
		t.Errorf("unexpected level: %v", m["level"])
	}
	if m["worker_id"] != float64(5) {
		t.Errorf("unexpected worker_id: %v", m["worker_id"])
	}
	if m["source"] == nil {
		t.Errorf("source not found in log: %s", buf.String())
	}
}

func TestLogHandlerLevel(t *testing.T) {
	for _, format := range []string{"text", "json"} {
		var buf bytes.Buffer
		logger := slog.New(newLogHandler(&buf, slog.LevelInfo, format))
		logger.Debug("hidden")
		if buf.Len() != 0 {
			t.Errorf("debug log must be suppressed at info level (%s): %s", format, buf.String())
		}
	}
}

func TestSetLogFormatInvalid(t *testing.T) {
	if err := SetLogFormat("xml"); err == nil {
		t.Error("SetLogFormat must return an error for an invalid format")
	}
}
