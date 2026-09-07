package eventlog

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

// DefaultMaxSize defines the default maximum log size (10 MB).
const DefaultMaxSize int64 = 10 * 1024 * 1024

// Event represents a structured, audit-ready operational event.
type Event struct {
	Timestamp time.Time `json:"timestamp"`
	Command   string    `json:"command"`
	Status    string    `json:"status"`
	Code      string    `json:"code"`
	Reclaimed bool      `json:"reclaimed,omitempty"`
}

// Logger writes JSON-line records to a local file with size capping and 0600 permissions.
type Logger struct {
	mu      sync.Mutex
	Path    string
	MaxSize int64
}

// NewLogger creates a new Logger with the specified path.
func NewLogger(path string, maxSize int64) *Logger {
	if maxSize <= 0 {
		maxSize = DefaultMaxSize
	}
	return &Logger{
		Path:    path,
		MaxSize: maxSize,
	}
}

// Log writes an event as a single JSON line.
func (l *Logger) Log(e Event) error {
	if l == nil || l.Path == "" {
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now().UTC()
	}

	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	// Check file size for capping
	if stat, err := os.Stat(l.Path); err == nil {
		if stat.Size()+int64(len(data)) > l.MaxSize {
			l.truncateOldest()
		}
	}

	f, err := os.OpenFile(l.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(data)
	return err
}

func (l *Logger) truncateOldest() {
	content, err := os.ReadFile(l.Path)
	if err != nil || len(content) == 0 {
		return
	}

	// Keep the second half of the file content
	halfLen := len(content) / 2
	// Find next newline to keep valid JSON lines
	idx := halfLen
	for idx < len(content) && content[idx] != '\n' {
		idx++
	}
	if idx < len(content) {
		idx++ // skip newline
	}

	truncated := content[idx:]
	_ = os.WriteFile(l.Path, truncated, 0600)
}
