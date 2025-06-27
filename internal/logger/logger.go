package logger

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
)

// ********************************************************
// ********* LOGGING **************************************
// ********************************************************

var showDateTime bool
var defaultLogger *Logger
var logFile *os.File

type LogLevel int

const (
	colorReset   = "\033[0m"
	colorRed     = "\033[31m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorBlue    = "\033[34m"
	colorMagenta = "\033[35m"
	colorCyan    = "\033[36m"
	colorWhite   = "\033[37m"
	colorOrange  = "\033[38;5;208m"
)

const (
	DEBUG LogLevel = iota
	INFO
	INFORM
	HIGHLIGHT
	WARN
	ERROR
	FATAL
)

type Logger struct {
	infoLogger  *log.Logger
	errorLogger *log.Logger
	level       LogLevel
}

func init() {
	defaultLogger = NewLogger(INFO)  // Reverted back to INFO
	showDateTime = false
}

// Add this helper function
func updateLoggerFlags(l *Logger) {
	var flags int
	if showDateTime {
		flags = log.Ldate | log.Ltime
	} else {
		flags = 0
	}
	l.infoLogger.SetFlags(flags)
	l.errorLogger.SetFlags(flags)
}

func SetShowDateTime(value bool) {
	showDateTime = value
	updateLoggerFlags(defaultLogger)
}

// SetLogOutput sets the output destination for logs
// 'c' for console, 'f' for file, 'b' for both
func SetLogOutput(outputType rune) {
	// Close any existing log file
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}

	var infoWriter, errorWriter *os.File

	switch outputType {
	case 'c': // Console only
		infoWriter = os.Stdout
		errorWriter = os.Stderr
	case 'f': // File only
		var err error
		logFile, err = os.OpenFile("/tmp/mcp.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
			os.Exit(1)
		}
		infoWriter = logFile
		errorWriter = logFile
	case 'b': // Both console and file
		var err error
		logFile, err = os.OpenFile("/tmp/mcp.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
			os.Exit(1)
		}
		// Use MultiWriter to write to both console and file
		infoWriter = os.Stdout
		errorWriter = os.Stderr
		// Note: This is simplified; for a real implementation you'd use io.MultiWriter
	default:
		fmt.Fprintf(os.Stderr, "Invalid log output type: %c\n", outputType)
		os.Exit(1)
	}

	// Update the loggers
	var flags int
	if showDateTime {
		flags = log.Ldate | log.Ltime
	} else {
		flags = 0
	}

	defaultLogger.infoLogger = log.New(infoWriter, "", flags)
	defaultLogger.errorLogger = log.New(errorWriter, "", flags)
}

func NewLogger(level LogLevel) *Logger {
	var flags int
	if showDateTime {
		flags = log.Ldate | log.Ltime
	} else {
		flags = 0
	}

	return &Logger{
		infoLogger:  log.New(os.Stdout, "", flags),
		errorLogger: log.New(os.Stderr, "", flags),
		level:       level,
	}
}

func (l *Logger) log(level LogLevel, format string, v ...any) {
	if level < l.level {
		return
	}

	// Get caller information
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "unknown"
		line = 0
	}

	// Get just the base filename instead of full path
	file = filepath.Base(file)

	// Format message with any additional arguments
	var msg string
	var jsonObjects []string // To store JSON representations of complex objects

	if len(v) > 0 {
		// Process arguments, converting non-primitives to JSON
		processedArgs, jsonStrings := processArgs(v...)
		jsonObjects = jsonStrings

		if len(processedArgs) > 0 {
			msg = fmt.Sprintf(format+" %s", strings.Join(processedArgs, " "))
		} else {
			msg = format
		}
	} else {
		msg = format
	}

	// Get color based on log level
	var colorCode string
	switch level {
	case DEBUG:
		colorCode = colorBlue
	case INFO:
		colorCode = colorGreen
	case INFORM:
		colorCode = colorMagenta
	case HIGHLIGHT:
		colorCode = colorCyan
	case WARN:
		colorCode = colorYellow
	case ERROR:
		colorCode = colorOrange
	case FATAL:
		colorCode = colorRed
	default:
		colorCode = colorReset
	}

	// Format with metadata in white and message in color
	logMsg := fmt.Sprintf("[%s] %s:%d: %s%s%s",
		level.String(),
		file,
		line,
		colorCode,
		msg,
		colorReset)

	// Write to appropriate output
	if level >= ERROR {
		l.errorLogger.Println(logMsg)
		// Print any JSON objects on separate lines
		for _, jsonObj := range jsonObjects {
			l.errorLogger.Println(fmt.Sprintf("[%s] %s:%d: %s%s%s",
				level.String(),
				file,
				line,
				colorCode,
				jsonObj,
				colorReset))
		}
	} else {
		l.infoLogger.Println(logMsg)
		// Print any JSON objects on separate lines
		for _, jsonObj := range jsonObjects {
			l.infoLogger.Println(fmt.Sprintf("[%s] %s:%d: %s%s%s",
				level.String(),
				file,
				line,
				colorCode,
				jsonObj,
				colorReset))
		}
	}
}

func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case INFORM:
		return "INFORM"
	case HIGHLIGHT:
		return "HIGHLIGHT"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// processArgs processes arguments, converting non-primitives to JSON
// Returns a slice of string representations for primitive types and a slice of JSON strings for complex types
func processArgs(args ...any) ([]string, []string) {
	if len(args) == 0 {
		return nil, nil
	}

	var primitives []string
	var jsonObjects []string

	for _, arg := range args {
		// Check if the argument is a primitive type
		if isPrimitive(arg) {
			// Format primitive types as before
			switch v := arg.(type) {
			case float32:
				primitives = append(primitives, fmt.Sprintf("%.2f", v))
			case float64:
				primitives = append(primitives, fmt.Sprintf("%.2f", v))
			case int:
				primitives = append(primitives, fmt.Sprintf("%d", v))
			case bool:
				primitives = append(primitives, fmt.Sprintf("%v", v))
			case string:
				primitives = append(primitives, v)
			case error:
				primitives = append(primitives, v.Error())
			case nil:
				primitives = append(primitives, "nil")
			default:
				// This shouldn't happen if isPrimitive is correct
				primitives = append(primitives, fmt.Sprintf("%v", v))
			}
		} else {
			// For non-primitive types, convert to JSON
			jsonBytes, err := json.MarshalIndent(arg, "", "  ")
			if err != nil {
				// If JSON conversion fails, use standard formatting
				primitives = append(primitives, fmt.Sprintf("%v", arg))
			} else {
				// Add a placeholder in the primitives list
				primitives = append(primitives, fmt.Sprintf("[Object of type %s]", reflect.TypeOf(arg)))
				// Add the JSON to the jsonObjects list
				jsonObjects = append(jsonObjects, string(jsonBytes))
			}
		}
	}
	return primitives, jsonObjects
}

// isPrimitive checks if a value is a primitive type
func isPrimitive(v any) bool {
	if v == nil {
		return true
	}

	switch v.(type) {
	case string, bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, error:
		return true
	default:
		return false
	}
}

// formatArgs converts any number of interface{} arguments into a formatted string
// Kept for backward compatibility
func formatArgs(args ...any) string {
	if len(args) == 0 {
		return ""
	}
	var parts []string
	for _, arg := range args {
		switch v := arg.(type) {
		case float32:
			parts = append(parts, fmt.Sprintf("%.2f", v))
		case float64:
			parts = append(parts, fmt.Sprintf("%.2f", v))
		case int:
			parts = append(parts, fmt.Sprintf("%d", v))
		case bool:
			parts = append(parts, fmt.Sprintf("%v", v))
		case error:
			parts = append(parts, v.Error())
		case nil:
			parts = append(parts, "nil")
		default:
			parts = append(parts, fmt.Sprintf("%v", v))
		}
	}
	return strings.Join(parts, " ")
}

// Convenience methods using the default logger
func Debug(format string, v ...any) {
	defaultLogger.log(DEBUG, format, v...)
}

func Info(format string, v ...any) {
	defaultLogger.log(INFO, format, v...)
}

func Inform(format string, v ...any) {
	defaultLogger.log(INFORM, format, v...)
}

func Highlight(format string, v ...any) {
	defaultLogger.log(HIGHLIGHT, format, v...)
}

func Warn(format string, v ...any) {
	defaultLogger.log(WARN, format, v...)
}

func Error(format string, v ...any) {
	defaultLogger.log(ERROR, format, v...)
}

func Fatal(format string, v ...any) {
	defaultLogger.log(FATAL, format, v...)
	os.Exit(1)
}
