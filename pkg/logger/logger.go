package logger

import (
	"fmt"
	"io"
	"jarvist/internal/common/models"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

// LogLevel represents the severity level of a log message
type LogLevel int

// Log levels
const (
	LevelTrace LogLevel = iota // Most detailed level, for tracing code execution
	LevelDebug                 // Debug information for developers
	LevelInfo                  // General information about normal operation
	LevelWarn                  // Warning messages that don't cause failure
	LevelError                 // Error messages that affect operation
	LevelFatal                 // Critical errors that stop the application
)

// MQTTPublisher defines an interface for publishing log messages to MQTT
type MQTTPublisher interface {
	// Publish sends a message to an MQTT topic
	Publish(topic string, payload interface{}) error
}

// String returns the string representation of a log level
func (l LogLevel) String() string {
	switch l {
	case LevelTrace:
		return "TRACE"
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", l)
	}
}

// ParseLogLevel parses a string into a LogLevel
func ParseLogLevel(level string) LogLevel {
	switch strings.ToUpper(level) {
	case "TRACE":
		return LevelTrace
	case "DEBUG":
		return LevelDebug
	case "INFO":
		return LevelInfo
	case "WARN", "WARNING":
		return LevelWarn
	case "ERROR":
		return LevelError
	case "FATAL":
		return LevelFatal
	default:
		return LevelInfo
	}
}

// LoggerOptions configures a Logger
type LoggerOptions struct {
	Level           LogLevel  // Minimum log level to display
	EnableConsole   bool      // Whether to log to console
	EnableFile      bool      // Whether to log to file
	EnableDatabase  bool      // Whether to log to database
	EnableMQTT      bool      // Whether to log to MQTT
	LogDir          string    // Log directory path
	LogFileName     string    // Log file name
	TimeFormat      string    // Format for timestamp
	IncludeLocation bool      // Whether to include file/line location
	MaxSizeMB       int       // Maximum size of log file in MB before rotation
	MaxBackups      int       // Maximum number of old log files to keep
	MaxAgeDays      int       // Maximum age of old log files in days
	OutputWriter    io.Writer // Custom output writer (optional)
	DatePattern     string    // Date pattern for log file rotation
	MQTTTopic       string    // MQTT topic for logs (if MQTT enabled)
	MQTTMinLevel    LogLevel  // Minimum log level to send to MQTT
	DbMinLevel      LogLevel  // Minimum log level to send to database
	FileMinLevel    LogLevel  // Minimum log level to write to file
	TableName       string    // Database table name for logs
}

// DefaultOptions returns the default logger options
func DefaultOptions() *LoggerOptions {
	return &LoggerOptions{
		Level:           LevelInfo,
		EnableConsole:   true,
		EnableFile:      true,
		EnableDatabase:  false,
		EnableMQTT:      false, // Disabled by default
		LogDir:          "",    // Will use fallback if not provided
		LogFileName:     "app.log",
		TimeFormat:      "2006-01-02 15:04:05.000",
		IncludeLocation: true,
		MaxSizeMB:       5,             // 10MB max file size
		MaxBackups:      5,             // Keep 5 old log files
		MaxAgeDays:      30,            // Keep logs for 30 days
		OutputWriter:    nil,           // No custom writer by default
		DatePattern:     "2006-01-02",  // Daily rotation by default
		MQTTTopic:       "logs",        // Default MQTT topic
		MQTTMinLevel:    LevelWarn,     // Only send WARN and above to MQTT by default
		DbMinLevel:      LevelWarn,     // Only send INFO and above to database by default
		FileMinLevel:    LevelInfo,     // By default, file level is the same as global level
		TableName:       "log_entries", // Default table name
	}
}

// Logger provides logging functionality
type Logger struct {
	options       *LoggerOptions
	logFile       *os.File
	mu            sync.Mutex
	writers       []io.Writer
	fileSize      int64 // Current log file size
	lastRotated   time.Time
	db            *gorm.DB
	mqttPublisher MQTTPublisher
	dbMu          sync.Mutex
	mqttMu        sync.Mutex
	hostname      string // Cache hostname for MQTT logs
}

// New creates a new Logger with the specified options
func New(options *LoggerOptions) *Logger {
	if options == nil {
		options = DefaultOptions()
	}

	// Use default log directory if not provided
	if options.LogDir == "" {
		options.LogDir = getDefaultLogDir()
	}

	// Ensure log directory exists
	if options.EnableFile && options.LogDir != "" {
		if err := os.MkdirAll(options.LogDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating log directory %s: %v\n", options.LogDir, err)
			options.LogDir = "."
		}
	}

	logger := &Logger{
		options:     options,
		writers:     []io.Writer{},
		lastRotated: time.Now(),
	}

	// Get hostname for MQTT logs
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	logger.hostname = hostname

	// Setup writers
	if options.EnableConsole {
		logger.writers = append(logger.writers, os.Stdout)
	}

	if options.OutputWriter != nil {
		logger.writers = append(logger.writers, options.OutputWriter)
	}

	if options.EnableFile && options.LogDir != "" {
		logger.setupLogFile()
	}

	return logger
}

// SetDB sets the database connection for logging to database
func (l *Logger) SetDB(db *gorm.DB) {
	l.dbMu.Lock()
	defer l.dbMu.Unlock()
	l.db = db
}

// SetMQTTPublisher sets the MQTT publisher implementation
func (l *Logger) SetMQTTPublisher(publisher MQTTPublisher) {
	l.mqttMu.Lock()
	defer l.mqttMu.Unlock()
	l.mqttPublisher = publisher

	if publisher != nil {
		l.options.EnableMQTT = true
	}
}

// SetMQTTOptions configures MQTT logging options
func (l *Logger) SetMQTTOptions(enabled bool, topic string, minLevel LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.options.EnableMQTT = enabled
	if topic != "" {
		l.options.MQTTTopic = topic
	}
	l.options.MQTTMinLevel = minLevel
}

func (l *Logger) logToMQTT(level LogLevel, component string, message string, fields map[string]interface{}) {
	if !l.options.EnableMQTT || level < l.options.MQTTMinLevel {
		return
	}

	l.mqttMu.Lock()
	publisher := l.mqttPublisher
	l.mqttMu.Unlock()

	if publisher == nil {
		return
	}

	if strings.Contains(message, "data/2") && strings.Contains(message, ".json.bson") {
		return
	}

	logPayload := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"level":     level.String(),
		"component": component,
		"message":   message,
		"hostname":  l.hostname,
		"service":   "jarvist",
	}

	if len(fields) > 0 {
		logPayload["fields"] = fields
	}

	topic := l.options.MQTTTopic
	if topic == "" {
		topic = "logs"
	}

	topic = fmt.Sprintf("%s/%s", topic, strings.ToLower(level.String()))

	go func(p MQTTPublisher, t string, payload map[string]interface{}) {
		err := p.Publish(t, payload)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error sending log to MQTT: %v\n", err)
		}
	}(publisher, topic, logPayload)
}

// logToDatabase logs a message to the database if enabled
func (l *Logger) logToDatabase(timestamp time.Time, level LogLevel, component string, message string) {
	if !l.options.EnableDatabase || level < l.options.DbMinLevel {
		return
	}

	l.dbMu.Lock()
	db := l.db
	l.dbMu.Unlock()

	if db == nil {
		return
	}

	// Truncate message if too long
	maxLength := 4000
	if len(message) > maxLength {
		message = message[:maxLength-3] + "..."
	}

	// Create log entry
	logEntry := models.LogEntry{
		Timestamp: timestamp,
		Level:     level.String(),
		Component: component,
		Message:   message,
	}

	// Log to database asynchronously to avoid blocking
	go func(database *gorm.DB, entry models.LogEntry) {
		if err := database.Create(&entry).Error; err != nil {
			// If there's an error, log to stderr to avoid recursive logging
			fmt.Fprintf(os.Stderr, "Error logging to database: %v\n", err)
		}
	}(db, logEntry)
}

// log logs a message at the specified level
func (l *Logger) log(level LogLevel, component string, message string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Skip if level is below the minimum
	if level < l.options.Level {
		return
	}

	// Check if log file needs rotation based on date pattern
	l.checkRotation()

	// Format message with component
	componentPrefix := ""
	if component != "" {
		componentPrefix = fmt.Sprintf("[%s] ", component)
	}

	// Format message if arguments provided
	formattedMsg := message
	if len(args) > 0 {
		formattedMsg = fmt.Sprintf(message, args...)
	}

	// Create timestamp
	now := time.Now()
	timestamp := now.Format(l.options.TimeFormat)

	// Get caller information if enabled
	location := ""
	if l.options.IncludeLocation {
		location = getCallerInfo(3) // Adjust level to get the actual caller, not the logger method
	}

	// Format the log message
	var logMessage string
	if location != "" {
		logMessage = fmt.Sprintf("[%s] [%s] [%s] %s%s\n", timestamp, level.String(), location, componentPrefix, formattedMsg)
	} else {
		logMessage = fmt.Sprintf("[%s] [%s] %s%s\n", timestamp, level.String(), componentPrefix, formattedMsg)
	}

	// Write to configured outputs
	for _, writer := range l.writers {
		// For logFile, check against FileMinLevel
		if writer == l.logFile {
			if level < l.options.FileMinLevel {
				continue // Skip writing to file if below FileMinLevel
			}

			n, err := writer.Write([]byte(logMessage))
			if err != nil {
				// If console logging is enabled, try to write error to stderr
				if l.options.EnableConsole {
					fmt.Fprintf(os.Stderr, "Error writing to log file: %v\n", err)
				}
			} else {
				l.fileSize += int64(n)

				// Check if we need to rotate
				if l.fileSize > int64(l.options.MaxSizeMB*1024*1024) {
					l.rotateLogFile()
				}
			}
		} else {
			// For non-file writers (console, custom writer)
			_, err := writer.Write([]byte(logMessage))
			if err != nil {
				// If console logging is enabled, try to write error to stderr
				if l.options.EnableConsole {
					fmt.Fprintf(os.Stderr, "Error writing to log: %v\n", err)
				}
			}
		}
	}

	// Store fields for context loggers
	fields := make(map[string]interface{})

	// Log to database if enabled
	if l.options.EnableDatabase {
		l.logToDatabase(now, level, component, formattedMsg)
	}

	// Log to MQTT if enabled
	if l.options.EnableMQTT {
		l.logToMQTT(level, component, formattedMsg, fields)
	}

	// For fatal logs, terminate the application
	if level == LevelFatal {
		os.Exit(1)
	}
}

// CleanupOldLogs deletes log entries from the database older than the specified number of days
func (l *Logger) CleanupOldLogs(days int) (int64, error) {
	l.dbMu.Lock()
	db := l.db
	l.dbMu.Unlock()

	if db == nil {
		return 0, fmt.Errorf("database connection not set")
	}

	cutoffTime := time.Now().AddDate(0, 0, -days)

	// Delete logs older than cutoff
	result := db.Where("timestamp < ?", cutoffTime).Delete(&models.LogEntry{})
	if result.Error != nil {
		return 0, result.Error
	}

	return result.RowsAffected, nil
}

// GetLogStats returns statistics about logs in the database
func (l *Logger) GetLogStats() (map[string]interface{}, error) {
	l.dbMu.Lock()
	db := l.db
	l.dbMu.Unlock()

	if db == nil {
		return nil, fmt.Errorf("database connection not set")
	}

	stats := make(map[string]interface{})

	// Count total logs
	var totalCount int64
	if err := db.Model(&models.LogEntry{}).Count(&totalCount).Error; err != nil {
		return nil, err
	}
	stats["total_logs"] = totalCount

	// Count logs by level
	levels := []string{"INFO", "WARN", "ERROR", "DEBUG", "TRACE", "FATAL"}
	for _, level := range levels {
		var count int64
		if err := db.Model(&models.LogEntry{}).Where("level = ?", level).Count(&count).Error; err != nil {
			continue
		}
		stats[strings.ToLower(level)+"_logs"] = count
	}

	// Get oldest and newest log
	var oldestLog, newestLog models.LogEntry
	if err := db.Model(&models.LogEntry{}).Order("timestamp ASC").First(&oldestLog).Error; err == nil {
		stats["oldest_log"] = oldestLog.Timestamp.Format(time.RFC3339)
	}

	if err := db.Model(&models.LogEntry{}).Order("timestamp DESC").First(&newestLog).Error; err == nil {
		stats["newest_log"] = newestLog.Timestamp.Format(time.RFC3339)
	}

	return stats, nil
}

// EnableMQTTLogging turns on MQTT logging with the given topic and minimum level
func (l *Logger) EnableMQTTLogging(topic string, minLevel LogLevel) {
	l.SetMQTTOptions(true, topic, minLevel)
}

// DisableMQTTLogging turns off MQTT logging
func (l *Logger) DisableMQTTLogging() {
	l.SetMQTTOptions(false, "", LevelInfo)
}

// GetMQTTStatus returns the current MQTT logging status
func (l *Logger) GetMQTTStatus() map[string]interface{} {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.mqttMu.Lock()
	mqttPublisher := l.mqttPublisher != nil
	l.mqttMu.Unlock()

	return map[string]interface{}{
		"enabled":       l.options.EnableMQTT,
		"topic":         l.options.MQTTTopic,
		"min_level":     l.options.MQTTMinLevel.String(),
		"publisher_set": mqttPublisher,
		"hostname":      l.hostname,
	}
}

// NewLogger creates a new Logger with default options
func NewLogger() *Logger {
	return New(DefaultOptions())
}

// setupLogFile opens or creates a log file and adds it to writers
func (l *Logger) setupLogFile() {
	// Generate log file name with date pattern if enabled
	logFileName := l.options.LogFileName
	if l.options.DatePattern != "" {
		// Add date prefix to log file name
		dateStr := time.Now().Format(l.options.DatePattern)
		ext := filepath.Ext(logFileName)
		baseName := strings.TrimSuffix(logFileName, ext)
		logFileName = fmt.Sprintf("%s_%s%s", baseName, dateStr, ext)
	}

	logFilePath := filepath.Join(l.options.LogDir, logFileName)

	// Check if log file exists and its size
	if info, err := os.Stat(logFilePath); err == nil {
		l.fileSize = info.Size()

		// Check if we need to rotate the log file
		if l.fileSize > int64(l.options.MaxSizeMB*1024*1024) {
			l.rotateLogFile()
		}
	} else {
		l.fileSize = 0
	}

	// Open or create log file
	if logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666); err == nil {
		l.logFile = logFile
		l.writers = append(l.writers, logFile)
	} else {
		fmt.Fprintf(os.Stderr, "Error opening log file %s: %v\n", logFilePath, err)
	}
}

// checkRotation checks if log file needs to be rotated based on date pattern
func (l *Logger) checkRotation() {
	if l.options.DatePattern == "" {
		return
	}

	now := time.Now()

	// Format current date and last rotated date
	currentDateStr := now.Format(l.options.DatePattern)
	lastDateStr := l.lastRotated.Format(l.options.DatePattern)

	// If date has changed, rotate the log file
	if currentDateStr != lastDateStr {
		l.rotateLogFile()
		l.lastRotated = now
	}
}

// rotateLogFile rotates log files, keeping up to MaxBackups old logs
func (l *Logger) rotateLogFile() {
	if l.logFile == nil {
		return
	}

	l.logFile.Close()
	l.logFile = nil

	// Remove from writers
	for i, w := range l.writers {
		if w == l.logFile {
			l.writers = append(l.writers[:i], l.writers[i+1:]...)
			break
		}
	}

	// Generate current log file name with date pattern
	logFileName := l.options.LogFileName
	if l.options.DatePattern != "" {
		// Add date prefix to log file name
		dateStr := time.Now().Format(l.options.DatePattern)
		ext := filepath.Ext(logFileName)
		baseName := strings.TrimSuffix(logFileName, ext)
		logFileName = fmt.Sprintf("%s_%s%s", baseName, dateStr, ext)
	}

	logFilePath := filepath.Join(l.options.LogDir, logFileName)

	// Generate a unique backup suffix based on timestamp
	timestamp := time.Now().Format("20060102-150405")
	backupPath := fmt.Sprintf("%s.%s", logFilePath, timestamp)

	// Rename current log file if it exists
	if _, err := os.Stat(logFilePath); err == nil {
		os.Rename(logFilePath, backupPath)
	}

	// Reset file size
	l.fileSize = 0

	// Clean up old log files if we have too many
	l.cleanupOldLogFiles()

	// Setup new log file
	l.setupLogFile()
}

// cleanupOldLogFiles removes old log files beyond MaxBackups or older than MaxAgeDays
func (l *Logger) cleanupOldLogFiles() {
	logFilePath := filepath.Join(l.options.LogDir, l.options.LogFileName)
	logDir := filepath.Dir(logFilePath)
	logBase := filepath.Base(l.options.LogFileName)

	// Get base name without extension for pattern matching
	ext := filepath.Ext(logBase)
	baseName := strings.TrimSuffix(logBase, ext)

	// Read the directory
	files, err := os.ReadDir(logDir)
	if err != nil {
		return
	}

	// Find all log files (both date pattern files and rotated backups)
	type logFileInfo struct {
		path     string
		modTime  time.Time
		isBackup bool
	}

	var logFiles []logFileInfo

	for _, file := range files {
		if strings.HasPrefix(file.Name(), baseName) {
			fileInfo, err := file.Info()
			if err != nil {
				continue
			}

			isBackup := strings.Contains(file.Name(), ".20") // Matches timestamp format

			logFiles = append(logFiles, logFileInfo{
				path:     filepath.Join(logDir, file.Name()),
				modTime:  fileInfo.ModTime(),
				isBackup: isBackup,
			})
		}
	}

	// Sort by modification time (newest first)
	for i := 0; i < len(logFiles); i++ {
		for j := i + 1; j < len(logFiles); j++ {
			if logFiles[i].modTime.Before(logFiles[j].modTime) {
				logFiles[i], logFiles[j] = logFiles[j], logFiles[i]
			}
		}
	}

	// Keep track of backups to retain
	backupsToKeep := l.options.MaxBackups
	if backupsToKeep <= 0 {
		backupsToKeep = 5 // Default
	}

	// Cutoff time for old files
	cutoffTime := time.Now().Add(-time.Duration(l.options.MaxAgeDays) * 24 * time.Hour)

	// Count backups and remove old ones
	backupCount := 0
	for _, fileInfo := range logFiles {
		// Skip current log file
		if fileInfo.path == filepath.Join(l.options.LogDir, logBase) {
			continue
		}

		// Check if it's a backup
		if fileInfo.isBackup {
			backupCount++
			// Remove if too many or too old
			if backupCount > backupsToKeep || fileInfo.modTime.Before(cutoffTime) {
				os.Remove(fileInfo.path)
			}
		} else if fileInfo.modTime.Before(cutoffTime) {
			// Remove old non-backup files
			os.Remove(fileInfo.path)
		}
	}
}

// getDefaultLogDir returns a suitable default directory for logs
func getDefaultLogDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "."
	}

	// Choose appropriate directory per OS
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(homeDir, "AppData", "Local", "Jarvist", "logs")
	case "darwin":
		return filepath.Join(homeDir, "Library", "Logs", "Jarvist")
	default: // Linux and others
		return filepath.Join(homeDir, ".jarvist", "logs")
	}
}

// getCallerInfo gets file and line information for the log caller
func getCallerInfo(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown:0"
	}
	// Get just the filename and line
	filename := filepath.Base(file)
	return fmt.Sprintf("%s:%d", filename, line)
}

// SetLogLevel sets the minimum log level
func (l *Logger) SetLogLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.options.Level = level
}

// GetLogLevel returns the current log level
func (l *Logger) GetLogLevel() LogLevel {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.options.Level
}

// UpdateLogPath updates the log directory and file name
func (l *Logger) UpdateLogPath(logDir string, logFileName string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Close existing log file if any
	if l.logFile != nil {
		l.logFile.Close()
		l.logFile = nil

		// Remove logFile from writers
		for i, w := range l.writers {
			if w == l.logFile {
				l.writers = append(l.writers[:i], l.writers[i+1:]...)
				break
			}
		}
	}

	// Update paths
	if logDir != "" {
		l.options.LogDir = logDir
	}
	if logFileName != "" {
		l.options.LogFileName = logFileName
	}

	// Create directory if needed
	if err := os.MkdirAll(l.options.LogDir, 0755); err != nil {
		return err
	}

	// Setup log file again
	l.setupLogFile()
	return nil
}

// CleanupLogs removes old log files
func (l *Logger) CleanupLogs() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.cleanupOldLogFiles()

	// Also clean up database logs
	if l.options.EnableDatabase && l.db != nil {
		_, err := l.CleanupOldLogs(l.options.MaxAgeDays)
		return err
	}

	return nil
}

// ForceRotate forces log rotation regardless of size
func (l *Logger) ForceRotate() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.rotateLogFile()
	return nil
}

// Close closes the log file
func (l *Logger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.logFile != nil {
		l.logFile.Close()
		l.logFile = nil
	}
}

// Trace logs a message at the TRACE level
func (l *Logger) Trace(component string, message string, args ...interface{}) {
	l.log(LevelTrace, component, message, args...)
}

// Debug logs a message at the DEBUG level
func (l *Logger) Debug(component string, message string, args ...interface{}) {
	l.log(LevelDebug, component, message, args...)
}

// Info logs a message at the INFO level
func (l *Logger) Info(component string, message string, args ...interface{}) {
	l.log(LevelInfo, component, message, args...)
}

// Warn logs a message at the WARN level
func (l *Logger) Warn(component string, message string, args ...interface{}) {
	l.log(LevelWarn, component, message, args...)
}

// Warning logs a message at the WARN level (alias for Warn)
func (l *Logger) Warning(component string, message string, args ...interface{}) {
	l.log(LevelWarn, component, message, args...)
}

// Error logs a message at the ERROR level
func (l *Logger) Error(component string, message string, args ...interface{}) {
	l.log(LevelError, component, message, args...)
}

// Fatal logs a message at the FATAL level and exits the application
func (l *Logger) Fatal(component string, message string, args ...interface{}) {
	l.log(LevelFatal, component, message, args...)
}

// Field represents a key-value pair for structured logging
type Field struct {
	Key   string
	Value interface{}
}

// F creates a new Field
func F(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// WithFields returns a context logger with predefined fields
func (l *Logger) WithFields(fields ...Field) *ContextLogger {
	fieldsMap := make(map[string]interface{}, len(fields))
	for _, field := range fields {
		fieldsMap[field.Key] = field.Value
	}
	return &ContextLogger{
		logger:    l,
		fields:    fieldsMap,
		component: "",
	}
}

// WithComponent returns a context logger with a predefined component
func (l *Logger) WithComponent(component string) *ContextLogger {
	return &ContextLogger{
		logger:    l,
		fields:    make(map[string]interface{}),
		component: component,
	}
}

// ContextLogger extends Logger with predefined fields
type ContextLogger struct {
	logger    *Logger
	fields    map[string]interface{}
	component string
}

// formatMessage formats the message with the context fields
func (cl *ContextLogger) formatMessage(message string) string {
	if len(cl.fields) == 0 {
		return message
	}

	// Format fields into string
	var parts []string
	for k, v := range cl.fields {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}

	fieldsStr := strings.Join(parts, " ")
	return fmt.Sprintf("%s [%s]", message, fieldsStr)
}

// WithField adds a field to the context logger
func (cl *ContextLogger) WithField(key string, value interface{}) *ContextLogger {
	newFields := make(map[string]interface{}, len(cl.fields)+1)
	for k, v := range cl.fields {
		newFields[k] = v
	}
	newFields[key] = value

	return &ContextLogger{
		logger:    cl.logger,
		fields:    newFields,
		component: cl.component,
	}
}

// WithFields adds multiple fields to the context logger
func (cl *ContextLogger) WithFields(fields ...Field) *ContextLogger {
	newFields := make(map[string]interface{}, len(cl.fields)+len(fields))
	for k, v := range cl.fields {
		newFields[k] = v
	}
	for _, field := range fields {
		newFields[field.Key] = field.Value
	}

	return &ContextLogger{
		logger:    cl.logger,
		fields:    newFields,
		component: cl.component,
	}
}

// WithComponent sets the component for this context logger
func (cl *ContextLogger) WithComponent(component string) *ContextLogger {
	return &ContextLogger{
		logger:    cl.logger,
		fields:    cl.fields,
		component: component,
	}
}

// Trace logs a message at the TRACE level with context fields
func (cl *ContextLogger) Trace(message string, args ...interface{}) {
	cl.logger.log(LevelTrace, cl.component, cl.formatMessage(message), args...)
}

// Debug logs a message at the DEBUG level with context fields
func (cl *ContextLogger) Debug(message string, args ...interface{}) {
	cl.logger.log(LevelDebug, cl.component, cl.formatMessage(message), args...)
}

// Info logs a message at the INFO level with context fields
func (cl *ContextLogger) Info(message string, args ...interface{}) {
	cl.logger.log(LevelInfo, cl.component, cl.formatMessage(message), args...)
}

// Warn logs a message at the WARN level with context fields
func (cl *ContextLogger) Warn(message string, args ...interface{}) {
	cl.logger.log(LevelWarn, cl.component, cl.formatMessage(message), args...)
}

// Warning logs a message at the WARN level with context fields (alias for Warn)
func (cl *ContextLogger) Warning(message string, args ...interface{}) {
	cl.logger.log(LevelWarn, cl.component, cl.formatMessage(message), args...)
}

// Error logs a message at the ERROR level with context fields
func (cl *ContextLogger) Error(message string, args ...interface{}) {
	cl.logger.log(LevelError, cl.component, cl.formatMessage(message), args...)
}

// Fatal logs a message at the FATAL level with context fields and exits the application
func (cl *ContextLogger) Fatal(message string, args ...interface{}) {
	cl.logger.log(LevelFatal, cl.component, cl.formatMessage(message), args...)
}
