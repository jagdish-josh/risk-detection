package audit

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"
)

const (
	DefaultBufferSize = 1000
)

// Logger is an async, append-only audit logger
type Logger struct {
	file   *os.File
	writer *bufio.Writer

	ch     chan AuditLog
	wg     sync.WaitGroup
	closed bool
	mu     sync.Mutex
}

// NewLogger initializes the audit logger
func NewLogger(filePath string) (*Logger, error) {
	if filePath == "" {
		return nil, errors.New("audit log file path required")
	}

	file, err := os.OpenFile(
		filePath,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0600,
	)
	if err != nil {
		return nil, err
	}

	l := &Logger{
		file:   file,
		writer: bufio.NewWriterSize(file, 64*1024),
		ch:     make(chan AuditLog, DefaultBufferSize),
	}

	l.wg.Add(1)
	go l.run()

	return l, nil
}

// Log sends an audit event to the logger (non-blocking)
func (l *Logger) Log(entry AuditLog) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return errors.New("audit logger is closed")
	}

	// enforce UTC timestamp
	if entry.EventTime.IsZero() {
		entry.EventTime = time.Now().UTC()
	}

	select {
	case l.ch <- entry:
		return nil
	default:
		// channel full → drop or handle
		return errors.New("audit log buffer full")
	}
}

// run is the single writer goroutine
func (l *Logger) run() {
	defer l.wg.Done()

	for entry := range l.ch {
		data, err := json.Marshal(entry)
		if err != nil {
			continue // never crash app because of audit
		}

		_, _ = l.writer.Write(data)
		_, _ = l.writer.WriteString("\n")
		_ = l.writer.Flush()
	}
}

// Close gracefully shuts down the logger
func (l *Logger) Close() error {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil
	}
	l.closed = true
	close(l.ch)
	l.mu.Unlock()

	l.wg.Wait()

	_ = l.writer.Flush()
	return l.file.Close()
}
