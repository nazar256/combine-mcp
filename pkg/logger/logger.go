package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/nazar256/combine-mcp/pkg/config"
)

var (
	logFile        *os.File
	errorLog       *log.Logger
	infoLog        *log.Logger
	debugLog       *log.Logger
	traceLog       *log.Logger
	errorLogStdout *log.Logger
	infoLogStdout  *log.Logger
	logLevel       config.LogLevel
	initOnce       sync.Once
)

// Init initializes the logger with the specified log level and optional log file
func Init(level config.LogLevel, logFilePath string) error {
	var err error
	initOnce.Do(func() {
		logLevel = level

		// Set up stdout writers for essential output only
		errorLogStdout = log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime)
		infoLogStdout = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)

		// Set up full logging (including debug/trace) to file only
		var logWriter io.Writer
		if logFilePath != "" {
			// Create directory if it doesn't exist
			if err = os.MkdirAll(filepath.Dir(logFilePath), 0755); err != nil {
				err = fmt.Errorf("failed to create log directory: %w", err)
				return
			}

			// Open log file
			logFile, err = os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				err = fmt.Errorf("failed to open log file: %w", err)
				return
			}
			logWriter = logFile
		} else {
			// If no log file is specified, use a null writer for debug/trace logs
			logWriter = io.Discard
		}

		// Create full loggers with appropriate prefixes (file-only)
		errorLog = log.New(logWriter, "ERROR: ", log.Ldate|log.Ltime)
		infoLog = log.New(logWriter, "INFO: ", log.Ldate|log.Ltime)
		debugLog = log.New(logWriter, "DEBUG: ", log.Ldate|log.Ltime)
		traceLog = log.New(logWriter, "TRACE: ", log.Ldate|log.Ltime)

		// Log initialization only to file to avoid corrupting JSON
		if logFile != nil {
			Info("Logger initialized with level: %v, log file: %v", level, logFilePath)
		}
	})
	return err
}

// Close closes the log file if one is open
func Close() {
	if logFile != nil {
		logFile.Close()
	}
}

// Error logs an error message to both stdout and log file
func Error(format string, v ...interface{}) {
	// Always log errors to the file
	errorLog.Printf(format, v...)

	// Only log to stdout if we're not in debug/trace mode, to avoid corrupting JSON
	if logLevel < config.LogLevelDebug {
		errorLogStdout.Printf(format, v...)
	}
}

// Info logs an info message if log level is Info or higher
func Info(format string, v ...interface{}) {
	if logLevel >= config.LogLevelInfo {
		// Always log to file
		infoLog.Printf(format, v...)

		// Only log to stdout if we're not in debug/trace mode, to avoid corrupting JSON
		if logLevel < config.LogLevelDebug {
			infoLogStdout.Printf(format, v...)
		}
	}
}

// Debug logs a debug message if log level is Debug or higher
// Debug messages only go to the log file, never stdout
func Debug(format string, v ...interface{}) {
	if logLevel >= config.LogLevelDebug {
		debugLog.Printf(format, v...)
	}
}

// Trace logs a trace message if log level is Trace
// Trace messages only go to the log file, never stdout
func Trace(format string, v ...interface{}) {
	if logLevel >= config.LogLevelTrace {
		traceLog.Printf(format, v...)
	}
}

// LogRequest logs incoming JSON-RPC requests
func LogRequest(method string, id interface{}, params interface{}) {
	if logLevel >= config.LogLevelDebug {
		debugLog.Printf("Request: method=%s, id=%v", method, id)
		if logLevel >= config.LogLevelTrace {
			traceLog.Printf("Request params: %+v", params)
		}
	}
}

// LogResponse logs outgoing JSON-RPC responses
func LogResponse(id interface{}, result interface{}, err error) {
	if logLevel >= config.LogLevelDebug {
		if err != nil {
			debugLog.Printf("Response: id=%v, error=%v", id, err)
		} else {
			debugLog.Printf("Response: id=%v, success=true", id)
			if logLevel >= config.LogLevelTrace {
				traceLog.Printf("Response result: %+v", result)
			}
		}
	}
}

// LogRPC logs the complete JSON-RPC message for maximum visibility
// RPC messages only go to the log file, never stdout
func LogRPC(direction string, message []byte) {
	if logLevel >= config.LogLevelTrace {
		// Add timestamp
		timestamp := time.Now().Format("2006-01-02 15:04:05.000")
		traceLog.Printf("%s RPC [%s]: %s", direction, timestamp, string(message))

		// Try to parse and log structured information about the message
		var jsonMsg map[string]interface{}
		if err := json.Unmarshal(message, &jsonMsg); err == nil {
			// Pretty print the parsed JSON for better readability
			prettyJSON, err := json.MarshalIndent(jsonMsg, "", "  ")
			if err == nil {
				traceLog.Printf("%s RPC PARSED [%s]:\n%s", direction, timestamp, string(prettyJSON))
			}
		}
	}
}

// Fatal logs an error message and exits the program
func Fatal(format string, v ...interface{}) {
	// Log to file if logger is initialized
	// Do NOT call Error() as it might write to stdout
	if errorLog != nil {
		errorLog.Printf(format, v...)
	}

	// Always write to stderr, never stdout
	fmt.Fprintf(os.Stderr, "FATAL: "+format+"\n", v...)

	// Close any open log files
	Close()

	// Exit the program
	os.Exit(1)
}
