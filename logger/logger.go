package logger

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// SystemLog represents a system-level log entry
type SystemLog struct {
	LogID   string   `json:"logId,omitempty"`
	Level   string   `json:"level"`
	User    *LogUser `json:"user,omitempty"`
	Message string   `json:"message,omitempty"`
}

// LogUser represents a user in a log entry
type LogUser struct {
	ID       string `json:"id"`
	Username string `json:"username,omitempty"`
}

// LogData extends SystemLog with HTTP request/response metadata
type LogData struct {
	SystemLog
	IP            string      `json:"ip,omitempty"`
	ClientAgent   string      `json:"clientAgent,omitempty"`
	RequestMethod string      `json:"requestMethod"`
	URL           string      `json:"url,omitempty"`
	Status        int         `json:"status,omitempty"`
	Length        string      `json:"length,omitempty"`
	Duration      interface{} `json:"duration,omitempty"`
	Timestamp     string      `json:"timestamp"`
}

// Logger is a singleton logger with daily log rotation
type Logger struct {
	mu          sync.Mutex
	logStream   *os.File
	currentDate string
}

var (
	instance *Logger
	once     sync.Once
)

// Instance returns the singleton Logger instance
func Instance() *Logger {
	once.Do(func() {
		l := &Logger{}
		l.ensureLogDir()
		l.setStream()
		go l.scheduleRotation()
		instance = l
	})
	return instance
}

func (l *Logger) ensureLogDir() {
	logDir, _ := filepath.Abs("logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Fatalf("failed to create log directory: %v", err)
	}

	configDir, _ := filepath.Abs("config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		log.Fatalf("failed to create config directory: %v", err)
	}
}

func (l *Logger) scheduleRotation() {
	for {
		now := time.Now()
		tomorrow := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		duration := time.Until(tomorrow)

		time.Sleep(duration)
		l.rotateLog()
	}
}

func (l *Logger) rotateLog() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.logStream != nil {
		l.logStream.Close()
	}
	l.setStream()
}

func (l *Logger) getLogFileName() string {
	now := time.Now()
	return fmt.Sprintf("%d-%d-%d.log", now.Year(), int(now.Month()), now.Day())
}

func (l *Logger) setStream() {
	logDir, _ := filepath.Abs("logs")
	l.currentDate = l.getLogFileName()
	filePath := filepath.Join(logDir, l.currentDate)

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("failed to open log file: %v", err)
	}
	l.logStream = f
}

// Write writes a LogData entry to the log output
func (l *Logger) Write(entry LogData) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Refresh the stream in case date changed
	currentFile := l.getLogFileName()
	if currentFile != l.currentDate {
		if l.logStream != nil {
			l.logStream.Close()
		}
		l.setStream()
	}

	entry.Timestamp = time.Now().Format("15:04:05")

	loggerMode := os.Getenv("LOGGER")
	if loggerMode == "" {
		loggerMode = "file"
	}

	if loggerMode == "console" {
		b, _ := json.MarshalIndent(entry, "", "  ")
		fmt.Println(string(b))
	} else {
		b, _ := json.MarshalIndent(entry, "", "  ")
		l.logStream.WriteString("\n" + string(b))
	}
}

// Log logs a general message or system log
func (l *Logger) Log(message string) {
	l.Write(LogData{
		SystemLog: SystemLog{
			Message: message,
			Level:   "Log",
		},
		RequestMethod: "-",
	})
}

// LogSystem logs a structured system log entry
func (l *Logger) LogSystem(entry SystemLog) {
	if entry.Level == "" {
		entry.Level = "Log"
	}
	if entry.Message == "" {
		entry.Message = "Those are system logs"
	}
	l.Write(LogData{
		SystemLog:     entry,
		RequestMethod: "-",
	})
}

// Warn logs a warning-level system log
func (l *Logger) Warn(entry SystemLog) {
	entry.Level = "Warning"
	l.Write(LogData{
		SystemLog:     entry,
		RequestMethod: "-",
	})
}

// Error logs an error, accepting either an error or a string message
func (l *Logger) Error(err error) {
	msg := "An unknown error occurred."
	if err != nil {
		msg = err.Error()
	}
	l.Write(LogData{
		SystemLog: SystemLog{
			Message: msg,
			Level:   "Error",
		},
		RequestMethod: "-",
	})
}

// ErrorMsg logs a raw error message string
func (l *Logger) ErrorMsg(msg string) {
	l.Write(LogData{
		SystemLog: SystemLog{
			Message: msg,
			Level:   "Error",
		},
		RequestMethod: "-",
	})
}

// getIP extracts the client IP from a request
func getIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}

// getContentLength extracts the Content-Length header
func getContentLength(w http.ResponseWriter) string {
	cl := w.Header().Get("Content-Length")
	if cl == "" {
		return "0"
	}
	return cl
}

// getUserAgent extracts the User-Agent header
func getUserAgent(r *http.Request) string {
	ua := r.Header.Get("User-Agent")
	if ua == "" {
		return "Unknown"
	}
	return ua
}

// responseWriter wraps http.ResponseWriter to capture status code and size
type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	return n, err
}

// EndpointLogger returns an HTTP middleware that logs each request
func (l *Logger) GinEndpointLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next() // process the request

		duration := time.Since(start).Milliseconds()

		l.Write(LogData{
			SystemLog: SystemLog{
				LogID: c.GetHeader("x-request-id"),
				Level: "Endpoint Log",
			},
			RequestMethod: c.Request.Method,
			IP:            c.ClientIP(), // Gin handles X-Forwarded-For automatically
			ClientAgent:   c.Request.UserAgent(),
			URL:           c.Request.RequestURI,
			Status:        c.Writer.Status(),
			Duration:      fmt.Sprintf("%d ms", duration),
			Length:        fmt.Sprintf("%d bytes", c.Writer.Size()),
		})
	}
}
