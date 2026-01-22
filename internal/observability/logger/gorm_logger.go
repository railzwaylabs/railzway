package logger

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	gormlogger "gorm.io/gorm/logger"
)

// GormLoggerConfig configures the GORM zap logger.
type GormLoggerConfig struct {
	Level                gormlogger.LogLevel
	SlowThreshold        time.Duration
	IgnoreRecordNotFound bool
}

// DefaultGormLoggerConfig returns production-safe defaults.
func DefaultGormLoggerConfig() GormLoggerConfig {
	return GormLoggerConfig{
		Level:                gormlogger.Warn,
		SlowThreshold:        200 * time.Millisecond,
		IgnoreRecordNotFound: false,
	}
}

// GormLogger implements gormlogger.Interface with zap-backed structured logging.
type GormLogger struct {
	level                gormlogger.LogLevel
	slowThreshold        time.Duration
	ignoreRecordNotFound bool
}

// NewGormLogger builds a new GormLogger.
func NewGormLogger(cfg GormLoggerConfig) *GormLogger {
	return &GormLogger{
		level:                cfg.Level,
		slowThreshold:        cfg.SlowThreshold,
		ignoreRecordNotFound: cfg.IgnoreRecordNotFound,
	}
}

// LogMode returns a logger with the updated level.
func (l *GormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	copy := *l
	copy.level = level
	return &copy
}

// Info logs informational messages from GORM.
func (l *GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.level < gormlogger.Info {
		return
	}
	fields := []zap.Field{zap.String("component", "gorm")}
	if len(data) > 0 {
		fields = append(fields, zap.Any("data", data))
	}
	FromContext(ctx).Info(msg, fields...)
}

// Warn logs warning messages from GORM.
func (l *GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.level < gormlogger.Warn {
		return
	}
	fields := []zap.Field{zap.String("component", "gorm")}
	if len(data) > 0 {
		fields = append(fields, zap.Any("data", data))
	}
	FromContext(ctx).Warn(msg, fields...)
}

// Error logs error messages from GORM.
func (l *GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.level < gormlogger.Error {
		return
	}
	fields := []zap.Field{zap.String("component", "gorm")}
	if len(data) > 0 {
		fields = append(fields, zap.Any("data", data))
	}
	FromContext(ctx).Error(msg, fields...)
}

// Trace logs SQL statements with structured fields.
func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.level <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	switch {
	case err != nil && l.level >= gormlogger.Error && (!errors.Is(err, gormlogger.ErrRecordNotFound) || !l.ignoreRecordNotFound):
		l.logQuery(ctx, fc, elapsed, err, zap.ErrorLevel)
	case l.slowThreshold != 0 && elapsed > l.slowThreshold && l.level >= gormlogger.Warn:
		l.logQuery(ctx, fc, elapsed, nil, zap.WarnLevel)
	case l.level >= gormlogger.Info:
		l.logQuery(ctx, fc, elapsed, nil, zap.DebugLevel)
	}
}

// ParamsFilter strips bound values to avoid logging sensitive data.
func (l *GormLogger) ParamsFilter(ctx context.Context, sql string, params ...interface{}) (string, []interface{}) {
	_ = ctx
	_ = params
	return sql, nil
}

func (l *GormLogger) logQuery(ctx context.Context, fc func() (string, int64), elapsed time.Duration, err error, level zapcore.Level) {
	sql, rows := fc()
	fields := []zap.Field{
		zap.String("component", "gorm"),
		zap.String("sql", strings.TrimSpace(sql)),
		zap.String("operation", operationFromSQL(sql)),
		zap.Int64("duration_ms", elapsed.Milliseconds()),
	}
	if rows >= 0 {
		fields = append(fields, zap.Int64("rows_affected", rows))
	}
	if err != nil {
		fields = append(fields, zap.Error(err))
	}

	log := FromContext(ctx)
	switch level {
	case zap.ErrorLevel:
		log.Error("gorm.query", fields...)
	case zap.WarnLevel:
		log.Warn("gorm.query", fields...)
	default:
		log.Debug("gorm.query", fields...)
	}
}

func operationFromSQL(sql string) string {
	normalized := strings.ToUpper(strings.TrimSpace(sql))
	if normalized == "" {
		return "UNKNOWN"
	}
	tokens := strings.Fields(normalized)
	for _, token := range tokens {
		token = strings.Trim(token, "();")
		switch token {
		case "SELECT", "INSERT", "UPDATE", "DELETE", "MERGE":
			return token
		case "WITH":
			continue
		}
	}
	return "UNKNOWN"
}

var _ gormlogger.Interface = (*GormLogger)(nil)
