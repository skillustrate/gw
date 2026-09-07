package eventlog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLogger_LogEvent(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "events.log")

	logger := NewLogger(logPath, 1024*1024)

	err := logger.Log(Event{
		Timestamp: time.Now().UTC(),
		Command:   "push",
		Status:    "SUCCESS",
		Code:      "PUSH",
	})
	if err != nil {
		t.Fatalf("failed to log event: %v", err)
	}

	// Verify file content
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	var recorded Event
	if err := json.Unmarshal(data, &recorded); err != nil {
		t.Fatalf("failed to parse logged event JSON: %v", err)
	}

	if recorded.Command != "push" || recorded.Status != "SUCCESS" || recorded.Code != "PUSH" {
		t.Errorf("unexpected recorded event: %+v", recorded)
	}
}

func TestLogger_SizeCapping(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "events.log")

	// Max size 300 bytes
	logger := NewLogger(logPath, 300)

	for i := 0; i < 20; i++ {
		_ = logger.Log(Event{
			Command: "inspect",
			Status:  "SUCCESS",
			Code:    "OK",
		})
	}

	stat, err := os.Stat(logPath)
	if err != nil {
		t.Fatalf("failed to stat log: %v", err)
	}

	if stat.Size() > 400 {
		t.Errorf("log file grew beyond limit: %d bytes", stat.Size())
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 {
		t.Errorf("expected retained log lines")
	}
}

func TestLogger_NilSafe(t *testing.T) {
	var logger *Logger
	err := logger.Log(Event{
		Command: "doctor",
		Status:  "SUCCESS",
		Code:    "DOCTOR_OK",
	})
	if err != nil {
		t.Errorf("expected nil logger to safely return nil, got %v", err)
	}
}
