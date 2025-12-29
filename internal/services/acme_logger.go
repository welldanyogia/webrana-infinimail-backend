package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ACMELogLevel represents the severity of a log entry
type ACMELogLevel string

const (
	ACMELogInfo    ACMELogLevel = "INFO"
	ACMELogWarning ACMELogLevel = "WARNING"
	ACMELogError   ACMELogLevel = "ERROR"
	ACMELogDebug   ACMELogLevel = "DEBUG"
)

// ACMELogEntry represents a single log entry
type ACMELogEntry struct {
	Timestamp time.Time    `json:"timestamp"`
	Level     ACMELogLevel `json:"level"`
	Domain    string       `json:"domain,omitempty"`
	Step      string       `json:"step"`
	Message   string       `json:"message"`
	Details   interface{}  `json:"details,omitempty"`
}

// ACMEDomainLog represents all logs for a specific domain
type ACMEDomainLog struct {
	Domain    string         `json:"domain"`
	StartedAt time.Time      `json:"started_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	Status    string         `json:"status"` // "in_progress", "success", "failed"
	Entries   []ACMELogEntry `json:"entries"`
}

// ACMELogger handles logging for ACME operations
type ACMELogger struct {
	mu       sync.RWMutex
	logDir   string
	logs     map[string]*ACMEDomainLog // domain -> log
	maxLogs  int                       // max number of domain logs to keep in memory
}

// NewACMELogger creates a new ACME logger
func NewACMELogger(logDir string) (*ACMELogger, error) {
	if logDir == "" {
		logDir = "./logs/acme"
	}

	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create ACME log directory: %w", err)
	}

	return &ACMELogger{
		logDir:  logDir,
		logs:    make(map[string]*ACMEDomainLog),
		maxLogs: 100, // Keep last 100 domain logs in memory
	}, nil
}


// StartDomainLog starts a new log session for a domain
func (l *ACMELogger) StartDomainLog(domain string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	l.logs[domain] = &ACMEDomainLog{
		Domain:    domain,
		StartedAt: now,
		UpdatedAt: now,
		Status:    "in_progress",
		Entries:   []ACMELogEntry{},
	}

	l.addEntryLocked(domain, ACMELogInfo, "session_start", "ACME certificate generation started", nil)
}

// Log adds a log entry for a domain
func (l *ACMELogger) Log(domain string, level ACMELogLevel, step, message string, details interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.addEntryLocked(domain, level, step, message, details)
}

// LogInfo adds an info log entry
func (l *ACMELogger) LogInfo(domain, step, message string) {
	l.Log(domain, ACMELogInfo, step, message, nil)
}

// LogWarning adds a warning log entry
func (l *ACMELogger) LogWarning(domain, step, message string, details interface{}) {
	l.Log(domain, ACMELogWarning, step, message, details)
}

// LogError adds an error log entry
func (l *ACMELogger) LogError(domain, step, message string, err error) {
	details := map[string]string{"error": err.Error()}
	l.Log(domain, ACMELogError, step, message, details)
}

// LogDebug adds a debug log entry
func (l *ACMELogger) LogDebug(domain, step, message string, details interface{}) {
	l.Log(domain, ACMELogDebug, step, message, details)
}

func (l *ACMELogger) addEntryLocked(domain string, level ACMELogLevel, step, message string, details interface{}) {
	// Ensure domain log exists
	if _, exists := l.logs[domain]; !exists {
		now := time.Now()
		l.logs[domain] = &ACMEDomainLog{
			Domain:    domain,
			StartedAt: now,
			UpdatedAt: now,
			Status:    "in_progress",
			Entries:   []ACMELogEntry{},
		}
	}

	entry := ACMELogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Domain:    domain,
		Step:      step,
		Message:   message,
		Details:   details,
	}

	l.logs[domain].Entries = append(l.logs[domain].Entries, entry)
	l.logs[domain].UpdatedAt = entry.Timestamp
}

// SetStatus sets the final status of a domain log
func (l *ACMELogger) SetStatus(domain, status string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if log, exists := l.logs[domain]; exists {
		log.Status = status
		log.UpdatedAt = time.Now()

		// Save to file when completed
		l.saveToFileLocked(domain)
	}
}

// MarkSuccess marks the domain log as successful
func (l *ACMELogger) MarkSuccess(domain string) {
	l.Log(domain, ACMELogInfo, "session_end", "ACME certificate generation completed successfully", nil)
	l.SetStatus(domain, "success")
}

// MarkFailed marks the domain log as failed
func (l *ACMELogger) MarkFailed(domain string, err error) {
	l.LogError(domain, "session_end", "ACME certificate generation failed", err)
	l.SetStatus(domain, "failed")
}


// GetDomainLog returns the log for a specific domain
func (l *ACMELogger) GetDomainLog(domain string) (*ACMEDomainLog, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	log, exists := l.logs[domain]
	if !exists {
		// Try to load from file
		l.mu.RUnlock()
		l.mu.Lock()
		defer func() {
			l.mu.Unlock()
			l.mu.RLock()
		}()
		return l.loadFromFileLocked(domain)
	}
	return log, exists
}

// GetAllLogs returns all domain logs (summary)
func (l *ACMELogger) GetAllLogs() []ACMEDomainLogSummary {
	l.mu.RLock()
	defer l.mu.RUnlock()

	summaries := make([]ACMEDomainLogSummary, 0, len(l.logs))
	for _, log := range l.logs {
		summaries = append(summaries, ACMEDomainLogSummary{
			Domain:     log.Domain,
			StartedAt:  log.StartedAt,
			UpdatedAt:  log.UpdatedAt,
			Status:     log.Status,
			EntryCount: len(log.Entries),
		})
	}
	return summaries
}

// ACMEDomainLogSummary is a summary of a domain log
type ACMEDomainLogSummary struct {
	Domain     string    `json:"domain"`
	StartedAt  time.Time `json:"started_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Status     string    `json:"status"`
	EntryCount int       `json:"entry_count"`
}

// saveToFileLocked saves the domain log to a file (must be called with lock held)
func (l *ACMELogger) saveToFileLocked(domain string) error {
	log, exists := l.logs[domain]
	if !exists {
		return nil
	}

	filename := l.getLogFilename(domain)
	data, err := json.MarshalIndent(log, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal log: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write log file: %w", err)
	}

	return nil
}

// loadFromFileLocked loads a domain log from file (must be called with lock held)
func (l *ACMELogger) loadFromFileLocked(domain string) (*ACMEDomainLog, bool) {
	filename := l.getLogFilename(domain)
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, false
	}

	var log ACMEDomainLog
	if err := json.Unmarshal(data, &log); err != nil {
		return nil, false
	}

	l.logs[domain] = &log
	return &log, true
}

func (l *ACMELogger) getLogFilename(domain string) string {
	// Sanitize domain name for filename - replace invalid characters with underscore
	safeDomain := domain
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, char := range invalidChars {
		safeDomain = strings.ReplaceAll(safeDomain, char, "_")
	}
	return filepath.Join(l.logDir, fmt.Sprintf("%s.json", safeDomain))
}

// ListLogFiles returns list of all log files
func (l *ACMELogger) ListLogFiles() ([]string, error) {
	files, err := os.ReadDir(l.logDir)
	if err != nil {
		return nil, err
	}

	var domains []string
	for _, f := range files {
		if !f.IsDir() && filepath.Ext(f.Name()) == ".json" {
			domain := f.Name()[:len(f.Name())-5] // Remove .json extension
			domains = append(domains, domain)
		}
	}
	return domains, nil
}

// Global ACME logger instance
var globalACMELogger *ACMELogger
var acmeLoggerOnce sync.Once

// GetACMELogger returns the global ACME logger instance
func GetACMELogger() *ACMELogger {
	acmeLoggerOnce.Do(func() {
		var err error
		globalACMELogger, err = NewACMELogger("./logs/acme")
		if err != nil {
			// Fallback to temp directory
			globalACMELogger, _ = NewACMELogger(os.TempDir() + "/acme-logs")
		}
	})
	return globalACMELogger
}
