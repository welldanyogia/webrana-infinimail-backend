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

// DomainSetupLogLevel represents the severity of a log entry
type DomainSetupLogLevel string

const (
	LogInfo    DomainSetupLogLevel = "INFO"
	LogWarning DomainSetupLogLevel = "WARNING"
	LogError   DomainSetupLogLevel = "ERROR"
	LogDebug   DomainSetupLogLevel = "DEBUG"
)

// DomainSetupStep represents the step in domain setup process
type DomainSetupStep string

const (
	StepDomainCreate     DomainSetupStep = "domain_create"
	StepDNSGuide         DomainSetupStep = "dns_guide"
	StepDNSExport        DomainSetupStep = "dns_export"
	StepDNSVerify        DomainSetupStep = "dns_verify"
	StepDNSVerifyMX      DomainSetupStep = "dns_verify_mx"
	StepDNSVerifyA       DomainSetupStep = "dns_verify_a"
	StepDNSVerifyTXT     DomainSetupStep = "dns_verify_txt"
	StepACMEInit         DomainSetupStep = "acme_init"
	StepACMEAccount      DomainSetupStep = "acme_account"
	StepACMEOrder        DomainSetupStep = "acme_order"
	StepACMEChallenge    DomainSetupStep = "acme_challenge"
	StepACMEDNSVerify    DomainSetupStep = "acme_dns_verify"
	StepACMEFinalize     DomainSetupStep = "acme_finalize"
	StepCertStore        DomainSetupStep = "cert_store"
	StepDomainActivate   DomainSetupStep = "domain_activate"
	StepSessionStart     DomainSetupStep = "session_start"
	StepSessionEnd       DomainSetupStep = "session_end"
	// New comprehensive steps
	StepRequestChallenge DomainSetupStep = "request_challenge"
	StepChallengeStored  DomainSetupStep = "challenge_stored"
	StepWaitingDNS       DomainSetupStep = "waiting_dns"
	StepDNSPropagation   DomainSetupStep = "dns_propagation"
	StepChallengeSubmit  DomainSetupStep = "challenge_submit"
	StepChallengeVerify  DomainSetupStep = "challenge_verify"
	StepCertGenerate     DomainSetupStep = "cert_generate"
	StepCertDownload     DomainSetupStep = "cert_download"
	StepCertInstall      DomainSetupStep = "cert_install"
	StepHTTPRequest      DomainSetupStep = "http_request"
	StepHTTPResponse     DomainSetupStep = "http_response"
	StepAPICall          DomainSetupStep = "api_call"
	StepValidation       DomainSetupStep = "validation"
)

// DomainSetupLogEntry represents a single log entry
type DomainSetupLogEntry struct {
	Timestamp   time.Time           `json:"timestamp"`
	Level       DomainSetupLogLevel `json:"level"`
	Domain      string              `json:"domain,omitempty"`
	Step        DomainSetupStep     `json:"step"`
	Message     string              `json:"message"`
	Details     interface{}         `json:"details,omitempty"`
	Duration    string              `json:"duration,omitempty"`
	RequestID   string              `json:"request_id,omitempty"`
	ErrorCode   string              `json:"error_code,omitempty"`
	StackTrace  string              `json:"stack_trace,omitempty"`
}

// DomainSetupLog represents all logs for a specific domain setup
type DomainSetupLog struct {
	Domain       string                `json:"domain"`
	StartedAt    time.Time             `json:"started_at"`
	UpdatedAt    time.Time             `json:"updated_at"`
	Status       string                `json:"status"` // "in_progress", "success", "failed"
	Entries      []DomainSetupLogEntry `json:"entries"`
	ErrorSummary string                `json:"error_summary,omitempty"`
	TotalSteps   int                   `json:"total_steps"`
	CurrentStep  int                   `json:"current_step"`
	Metadata     map[string]string     `json:"metadata,omitempty"`
}

// DomainSetupLogger handles logging for domain setup operations
type DomainSetupLogger struct {
	mu       sync.RWMutex
	logDir   string
	logs     map[string]*DomainSetupLog // domain -> log
	maxLogs  int                        // max number of domain logs to keep in memory
}

// NewDomainSetupLogger creates a new domain setup logger
func NewDomainSetupLogger(logDir string) (*DomainSetupLogger, error) {
	if logDir == "" {
		logDir = "./logs/domain-setup"
	}

	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create domain setup log directory: %w", err)
	}

	return &DomainSetupLogger{
		logDir:  logDir,
		logs:    make(map[string]*DomainSetupLog),
		maxLogs: 100, // Keep last 100 domain logs in memory
	}, nil
}

// Aliases for backward compatibility with ACME logger
type ACMELogLevel = DomainSetupLogLevel
type ACMELogEntry = DomainSetupLogEntry
type ACMEDomainLog = DomainSetupLog
type ACMELogger = DomainSetupLogger

const (
	ACMELogInfo    = LogInfo
	ACMELogWarning = LogWarning
	ACMELogError   = LogError
	ACMELogDebug   = LogDebug
)

// NewACMELogger creates a new ACME logger (alias for backward compatibility)
func NewACMELogger(logDir string) (*DomainSetupLogger, error) {
	if logDir == "" {
		logDir = "./logs/domain-setup"
	}
	return NewDomainSetupLogger(logDir)
}


// StartDomainLog starts a new log session for a domain
func (l *DomainSetupLogger) StartDomainLog(domain string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	l.logs[domain] = &DomainSetupLog{
		Domain:      domain,
		StartedAt:   now,
		UpdatedAt:   now,
		Status:      "in_progress",
		Entries:     []DomainSetupLogEntry{},
		TotalSteps:  0,
		CurrentStep: 0,
		Metadata:    make(map[string]string),
	}

	l.addEntryLocked(domain, LogInfo, StepSessionStart, "Domain setup session started", nil)
}

// SetMetadata sets metadata for a domain log
func (l *DomainSetupLogger) SetMetadata(domain, key, value string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if log, exists := l.logs[domain]; exists {
		if log.Metadata == nil {
			log.Metadata = make(map[string]string)
		}
		log.Metadata[key] = value
		l.saveToFileLocked(domain)
	}
}

// SetTotalSteps sets the total number of steps for progress tracking
func (l *DomainSetupLogger) SetTotalSteps(domain string, total int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if log, exists := l.logs[domain]; exists {
		log.TotalSteps = total
		l.saveToFileLocked(domain)
	}
}

// IncrementStep increments the current step counter
func (l *DomainSetupLogger) IncrementStep(domain string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if log, exists := l.logs[domain]; exists {
		log.CurrentStep++
		l.saveToFileLocked(domain)
	}
}

// Log adds a log entry for a domain
func (l *DomainSetupLogger) Log(domain string, level DomainSetupLogLevel, step DomainSetupStep, message string, details interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.addEntryLocked(domain, level, step, message, details)
}

// LogInfo adds an info log entry
func (l *DomainSetupLogger) LogInfo(domain string, step DomainSetupStep, message string) {
	l.Log(domain, LogInfo, step, message, nil)
}

// LogInfoWithDetails adds an info log entry with details
func (l *DomainSetupLogger) LogInfoWithDetails(domain string, step DomainSetupStep, message string, details interface{}) {
	l.Log(domain, LogInfo, step, message, details)
}

// LogWarning adds a warning log entry
func (l *DomainSetupLogger) LogWarning(domain string, step DomainSetupStep, message string, details interface{}) {
	l.Log(domain, LogWarning, step, message, details)
}

// LogError adds an error log entry
func (l *DomainSetupLogger) LogError(domain string, step DomainSetupStep, message string, err error) {
	details := map[string]string{"error": err.Error()}
	l.Log(domain, LogError, step, message, details)
	
	// Also update error summary
	l.mu.Lock()
	if log, exists := l.logs[domain]; exists {
		log.ErrorSummary = err.Error()
	}
	l.mu.Unlock()
}

// LogErrorWithCode adds an error log entry with error code
func (l *DomainSetupLogger) LogErrorWithCode(domain string, step DomainSetupStep, message string, err error, errorCode string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// Ensure domain log exists
	if _, exists := l.logs[domain]; !exists {
		now := time.Now()
		l.logs[domain] = &DomainSetupLog{
			Domain:      domain,
			StartedAt:   now,
			UpdatedAt:   now,
			Status:      "in_progress",
			Entries:     []DomainSetupLogEntry{},
			TotalSteps:  0,
			CurrentStep: 0,
			Metadata:    make(map[string]string),
		}
	}

	entry := DomainSetupLogEntry{
		Timestamp: time.Now(),
		Level:     LogError,
		Domain:    domain,
		Step:      step,
		Message:   message,
		Details:   map[string]string{"error": err.Error()},
		ErrorCode: errorCode,
	}

	l.logs[domain].Entries = append(l.logs[domain].Entries, entry)
	l.logs[domain].UpdatedAt = entry.Timestamp
	l.logs[domain].ErrorSummary = err.Error()
	
	l.saveToFileLocked(domain)
}

// LogWithDuration adds a log entry with duration tracking
func (l *DomainSetupLogger) LogWithDuration(domain string, level DomainSetupLogLevel, step DomainSetupStep, message string, duration time.Duration, details interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// Ensure domain log exists
	if _, exists := l.logs[domain]; !exists {
		now := time.Now()
		l.logs[domain] = &DomainSetupLog{
			Domain:      domain,
			StartedAt:   now,
			UpdatedAt:   now,
			Status:      "in_progress",
			Entries:     []DomainSetupLogEntry{},
			TotalSteps:  0,
			CurrentStep: 0,
			Metadata:    make(map[string]string),
		}
	}

	entry := DomainSetupLogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Domain:    domain,
		Step:      step,
		Message:   message,
		Details:   details,
		Duration:  duration.String(),
	}

	l.logs[domain].Entries = append(l.logs[domain].Entries, entry)
	l.logs[domain].UpdatedAt = entry.Timestamp
	
	l.saveToFileLocked(domain)
}

// LogHTTPRequest logs an HTTP request for debugging
func (l *DomainSetupLogger) LogHTTPRequest(domain string, method, url string, statusCode int, duration time.Duration, requestID string) {
	details := map[string]interface{}{
		"method":      method,
		"url":         url,
		"status_code": statusCode,
		"duration_ms": duration.Milliseconds(),
	}
	
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// Ensure domain log exists
	if _, exists := l.logs[domain]; !exists {
		now := time.Now()
		l.logs[domain] = &DomainSetupLog{
			Domain:      domain,
			StartedAt:   now,
			UpdatedAt:   now,
			Status:      "in_progress",
			Entries:     []DomainSetupLogEntry{},
			TotalSteps:  0,
			CurrentStep: 0,
			Metadata:    make(map[string]string),
		}
	}

	level := LogInfo
	if statusCode >= 400 {
		level = LogError
	}

	entry := DomainSetupLogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Domain:    domain,
		Step:      StepHTTPRequest,
		Message:   fmt.Sprintf("%s %s -> %d", method, url, statusCode),
		Details:   details,
		Duration:  duration.String(),
		RequestID: requestID,
	}

	l.logs[domain].Entries = append(l.logs[domain].Entries, entry)
	l.logs[domain].UpdatedAt = entry.Timestamp
	
	l.saveToFileLocked(domain)
}

// LogAPIResponse logs an API response with full details
func (l *DomainSetupLogger) LogAPIResponse(domain string, endpoint string, responseBody interface{}, statusCode int) {
	details := map[string]interface{}{
		"endpoint":    endpoint,
		"status_code": statusCode,
		"response":    responseBody,
	}
	
	level := LogDebug
	if statusCode >= 400 {
		level = LogError
	}
	
	l.Log(domain, level, StepHTTPResponse, fmt.Sprintf("API Response from %s: %d", endpoint, statusCode), details)
}

// LogDebug adds a debug log entry
func (l *DomainSetupLogger) LogDebug(domain string, step DomainSetupStep, message string, details interface{}) {
	l.Log(domain, LogDebug, step, message, details)
}

func (l *DomainSetupLogger) addEntryLocked(domain string, level DomainSetupLogLevel, step DomainSetupStep, message string, details interface{}) {
	// Ensure domain log exists
	if _, exists := l.logs[domain]; !exists {
		now := time.Now()
		l.logs[domain] = &DomainSetupLog{
			Domain:      domain,
			StartedAt:   now,
			UpdatedAt:   now,
			Status:      "in_progress",
			Entries:     []DomainSetupLogEntry{},
			TotalSteps:  0,
			CurrentStep: 0,
			Metadata:    make(map[string]string),
		}
	}

	entry := DomainSetupLogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Domain:    domain,
		Step:      step,
		Message:   message,
		Details:   details,
	}

	l.logs[domain].Entries = append(l.logs[domain].Entries, entry)
	l.logs[domain].UpdatedAt = entry.Timestamp
	
	// Auto-save after each entry for real-time viewing
	l.saveToFileLocked(domain)
}

// SetStatus sets the final status of a domain log
func (l *DomainSetupLogger) SetStatus(domain, status string) {
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
func (l *DomainSetupLogger) MarkSuccess(domain string) {
	l.Log(domain, LogInfo, StepSessionEnd, "Domain setup completed successfully", nil)
	l.SetStatus(domain, "success")
}

// MarkFailed marks the domain log as failed
func (l *DomainSetupLogger) MarkFailed(domain string, err error) {
	l.LogError(domain, StepSessionEnd, "Domain setup failed", err)
	l.SetStatus(domain, "failed")
}


// GetDomainLog returns the log for a specific domain
func (l *DomainSetupLogger) GetDomainLog(domain string) (*DomainSetupLog, bool) {
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
func (l *DomainSetupLogger) GetAllLogs() []DomainSetupLogSummary {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Also load logs from files that aren't in memory
	l.loadAllFromFilesLocked()

	summaries := make([]DomainSetupLogSummary, 0, len(l.logs))
	for _, log := range l.logs {
		summaries = append(summaries, DomainSetupLogSummary{
			Domain:       log.Domain,
			StartedAt:    log.StartedAt,
			UpdatedAt:    log.UpdatedAt,
			Status:       log.Status,
			EntryCount:   len(log.Entries),
			ErrorSummary: log.ErrorSummary,
			TotalSteps:   log.TotalSteps,
			CurrentStep:  log.CurrentStep,
		})
	}
	return summaries
}

// DomainSetupLogSummary is a summary of a domain log
type DomainSetupLogSummary struct {
	Domain       string    `json:"domain"`
	StartedAt    time.Time `json:"started_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Status       string    `json:"status"`
	EntryCount   int       `json:"entry_count"`
	ErrorSummary string    `json:"error_summary,omitempty"`
	TotalSteps   int       `json:"total_steps"`
	CurrentStep  int       `json:"current_step"`
}

// Alias for backward compatibility
type ACMEDomainLogSummary = DomainSetupLogSummary

// saveToFileLocked saves the domain log to a file (must be called with lock held)
func (l *DomainSetupLogger) saveToFileLocked(domain string) error {
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
func (l *DomainSetupLogger) loadFromFileLocked(domain string) (*DomainSetupLog, bool) {
	filename := l.getLogFilename(domain)
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, false
	}

	var log DomainSetupLog
	if err := json.Unmarshal(data, &log); err != nil {
		return nil, false
	}

	l.logs[domain] = &log
	return &log, true
}

// loadAllFromFilesLocked loads all log files into memory
func (l *DomainSetupLogger) loadAllFromFilesLocked() {
	files, err := os.ReadDir(l.logDir)
	if err != nil {
		return
	}

	for _, f := range files {
		if !f.IsDir() && filepath.Ext(f.Name()) == ".json" {
			domain := f.Name()[:len(f.Name())-5] // Remove .json extension
			if _, exists := l.logs[domain]; !exists {
				l.loadFromFileLocked(domain)
			}
		}
	}
}

func (l *DomainSetupLogger) getLogFilename(domain string) string {
	// Sanitize domain name for filename - replace invalid characters with underscore
	safeDomain := domain
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, char := range invalidChars {
		safeDomain = strings.ReplaceAll(safeDomain, char, "_")
	}
	return filepath.Join(l.logDir, fmt.Sprintf("%s.json", safeDomain))
}

// ListLogFiles returns list of all log files
func (l *DomainSetupLogger) ListLogFiles() ([]string, error) {
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

// Global domain setup logger instance
var globalDomainSetupLogger *DomainSetupLogger
var domainSetupLoggerOnce sync.Once

// GetDomainSetupLogger returns the global domain setup logger instance
func GetDomainSetupLogger() *DomainSetupLogger {
	domainSetupLoggerOnce.Do(func() {
		var err error
		globalDomainSetupLogger, err = NewDomainSetupLogger("./logs/domain-setup")
		if err != nil {
			// Fallback to temp directory
			globalDomainSetupLogger, _ = NewDomainSetupLogger(os.TempDir() + "/domain-setup-logs")
		}
	})
	return globalDomainSetupLogger
}

// GetACMELogger returns the global ACME logger instance (alias for backward compatibility)
func GetACMELogger() *DomainSetupLogger {
	return GetDomainSetupLogger()
}
