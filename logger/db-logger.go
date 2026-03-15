package logger

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gorm.io/gorm/logger"
)

// DBLogger is the generic interface your db logger satisfies.
// Nothing in here is GORM-specific — swap out the ORM and just
// write a new thin adapter that calls these same methods.
type DBLogger interface {
	LogQuery(query string, params []interface{})
	LogQueryError(err error, query string, params []interface{})
	LogQuerySlow(duration time.Duration, query string, params []interface{})
	LogSchema(message string)
	LogMigration(message string)
	Log(level, message string)
}

// FileDBLogger is the singleton implementation that writes to daily rotating files.
type FileDBLogger struct {
	Logger
}

var (
	dbInstance *FileDBLogger
	dbOnce     sync.Once
)

// DBLoggerInstance returns the singleton FileDBLogger.
func DBLoggerInstance() *FileDBLogger {
	dbOnce.Do(func() {
		l := &FileDBLogger{}
		l.ensureLogDir()
		l.setStream()
		go l.scheduleRotation()
		dbInstance = l
	})
	return dbInstance
}

func (l *FileDBLogger) ensureLogDir() {
	logDir, _ := filepath.Abs("db-logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		panic(fmt.Sprintf("failed to create db-logs directory: %v", err))
	}
}

func (l *FileDBLogger) getLogFileName() string {
	now := time.Now()
	return fmt.Sprintf("%d-%d-%d.log", now.Year(), int(now.Month()), now.Day())
}

func (l *FileDBLogger) setStream() {
	logDir, _ := filepath.Abs("db-logs")
	l.currentDate = l.getLogFileName()
	filePath := filepath.Join(logDir, l.currentDate)

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(fmt.Sprintf("failed to open db log file: %v", err))
	}
	l.logStream = f
}

func (l *FileDBLogger) scheduleRotation() {
	for {
		now := time.Now()
		tomorrow := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		time.Sleep(time.Until(tomorrow))

		l.mu.Lock()
		if l.logStream != nil {
			l.logStream.Close()
		}
		l.setStream()
		l.mu.Unlock()
	}
}

// write is the internal method that formats and writes a log line.
func (l *FileDBLogger) write(level, message string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Rotate if date has changed
	if l.getLogFileName() != l.currentDate {
		if l.logStream != nil {
			l.logStream.Close()
		}
		l.setStream()
	}

	line := fmt.Sprintf("%s [%s] %s\n", time.Now().Format("15:04:05"), level, message)
	l.logStream.WriteString(line)
}

func formatParams(params []interface{}) string {
	if len(params) == 0 {
		return ""
	}
	return fmt.Sprintf("%v", params)
}

// --- DBLogger interface implementation ---

func (l *FileDBLogger) LogQuery(query string, params []interface{}) {
	// Commented out by default (same as the TS original) — uncomment to enable query logging.
	// l.write("QUERY", fmt.Sprintf("%s -- %s", query, formatParams(params)))
}

func (l *FileDBLogger) LogQueryError(err error, query string, params []interface{}) {
	msg := "unknown error"
	if err != nil {
		msg = err.Error()
	}
	l.write("ERROR", fmt.Sprintf("%s -- %s -- %s", msg, query, formatParams(params)))
}

func (l *FileDBLogger) LogQuerySlow(duration time.Duration, query string, params []interface{}) {
	l.write("SLOW", fmt.Sprintf("%s -- %s -- %s", duration, query, formatParams(params)))
}

func (l *FileDBLogger) LogSchema(message string) {
	l.write("SCHEMA", message)
}

func (l *FileDBLogger) LogMigration(message string) {
	l.write("MIGRATION", message)
}

func (l *FileDBLogger) Log(level, message string) {
	l.write(level, message)
}

// =============================================================================
// GORM adapter — this is the only GORM-coupled part.
// If you switch ORMs, delete this section and write a new adapter.
// =============================================================================

// GormAdapter wraps FileDBLogger and satisfies gorm.io/gorm/logger.Interface.
type GormAdapter struct {
	db          *FileDBLogger
	SlowQueryMs time.Duration // threshold for slow query logging (default: 200ms)
}

// NewGormLogger returns a GormAdapter ready to plug into GORM's config.
func NewGormLogger(slowThreshold ...time.Duration) *GormAdapter {
	threshold := 200 * time.Millisecond
	if len(slowThreshold) > 0 {
		threshold = slowThreshold[0]
	}
	return &GormAdapter{
		db:          DBLoggerInstance(),
		SlowQueryMs: threshold,
	}
}

// LogMode satisfies logger.Interface. We ignore the level since our logger
// controls verbosity itself.
func (g *GormAdapter) LogMode(level logger.LogLevel) logger.Interface {
	return g
}

func (g *GormAdapter) Info(ctx context.Context, msg string, data ...interface{}) {
	g.db.Log("INFO", fmt.Sprintf(msg, data...))
}

func (g *GormAdapter) Warn(ctx context.Context, msg string, data ...interface{}) {
	g.db.Log("WARN", fmt.Sprintf(msg, data...))
}

func (g *GormAdapter) Error(ctx context.Context, msg string, data ...interface{}) {
	g.db.Log("ERROR", fmt.Sprintf(msg, data...))
}

func (g *GormAdapter) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	elapsed := time.Since(begin)
	sql, _ := fc()

	switch {
	case err != nil && !errors.Is(err, logger.ErrRecordNotFound):
		g.db.LogQueryError(err, sql, nil)
	case elapsed > g.SlowQueryMs:
		g.db.LogQuerySlow(elapsed, sql, nil)
	default:
		g.db.LogQuery(sql, nil)
	}
}
