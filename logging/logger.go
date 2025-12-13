package logging

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// LogLevel represents the severity of a log entry
type LogLevel string

const (
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp string   `json:"timestamp"`
	Level     LogLevel `json:"level"`
	Message   string   `json:"message"`
	Details   any      `json:"details,omitempty"`
}

// Logger handles collecting and sending logs to the TRMNL API
type Logger struct {
	baseURL    string
	apiKey     string
	entries    []LogEntry
	mu         sync.Mutex
	maxEntries int
	verbose    bool
}

// NewLogger creates a new logger instance
func NewLogger(baseURL, apiKey string, verbose bool) *Logger {
	return &Logger{
		baseURL:    baseURL,
		apiKey:     apiKey,
		entries:    make([]LogEntry, 0, 20),
		maxEntries: 20, // Keep last 20 entries
		verbose:    verbose,
	}
}

// Log adds a log entry
func (l *Logger) Log(level LogLevel, message string, details any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level,
		Message:   message,
		Details:   details,
	}

	// Print to console if verbose
	if l.verbose {
		l.printEntry(entry)
	}

	// Add to buffer
	l.entries = append(l.entries, entry)

	// Keep only last N entries
	if len(l.entries) > l.maxEntries {
		l.entries = l.entries[len(l.entries)-l.maxEntries:]
	}
}

// Info logs an info message
func (l *Logger) Info(message string, details any) {
	l.Log(LogLevelInfo, message, details)
}

// Warn logs a warning message
func (l *Logger) Warn(message string, details any) {
	l.Log(LogLevelWarn, message, details)
}

// Error logs an error message
func (l *Logger) Error(message string, details any) {
	l.Log(LogLevelError, message, details)
}

// Flush sends all buffered logs to the API and clears the buffer
func (l *Logger) Flush() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.entries) == 0 {
		if l.verbose {
			fmt.Println("[Logger] No logs to flush")
		}
		return nil
	}

	if l.apiKey == "" {
		// No API key, can't send logs
		if l.verbose {
			fmt.Println("[Logger] Skipping log upload - no API key configured")
		}
		return nil
	}

	if l.verbose {
		fmt.Printf("[Logger] Preparing to send %d log entries to API...\n", len(l.entries))
	}

	// Prepare payload
	payload := map[string][]LogEntry{
		"logs": l.entries,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal logs: %w", err)
	}

	// Send to API
	url := l.baseURL + "/api/log"
	if l.verbose {
		fmt.Printf("[Logger] Sending logs to %s\n", url)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Access-Token", l.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		if l.verbose {
			fmt.Printf("[Logger] Failed to send logs: %v\n", err)
		}
		return fmt.Errorf("failed to send logs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		if l.verbose {
			fmt.Printf("[Logger] Unexpected response status: %d\n", resp.StatusCode)
		}
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	if l.verbose {
		fmt.Printf("[Logger] ✓ Successfully sent %d log entries to API (status: %d)\n", len(l.entries), resp.StatusCode)
	}

	// Clear buffer after successful send
	l.entries = make([]LogEntry, 0, 20)

	return nil
}

// FlushOnError sends logs only if there are error-level entries
func (l *Logger) FlushOnError() error {
	l.mu.Lock()
	hasError := false
	for _, entry := range l.entries {
		if entry.Level == LogLevelError {
			hasError = true
			break
		}
	}
	l.mu.Unlock()

	if hasError {
		return l.Flush()
	}

	return nil
}

// printEntry prints a log entry to console
func (l *Logger) printEntry(entry LogEntry) {
	prefix := ""
	switch entry.Level {
	case LogLevelInfo:
		prefix = "[INFO]"
	case LogLevelWarn:
		prefix = "[WARN]"
	case LogLevelError:
		prefix = "[ERROR]"
	}

	if entry.Details != nil {
		detailsJSON, _ := json.Marshal(entry.Details)
		fmt.Printf("%s %s: %s | %s\n", prefix, entry.Timestamp, entry.Message, string(detailsJSON))
	} else {
		fmt.Printf("%s %s: %s\n", prefix, entry.Timestamp, entry.Message)
	}
}
