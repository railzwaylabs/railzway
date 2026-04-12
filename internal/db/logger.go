package db

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type zapGormLogger struct {
	logger               *zap.Logger
	level                gormlogger.LogLevel
	slowThreshold        time.Duration
	ignoreRecordNotFound bool
}

func newZapGormLogger(logger *zap.Logger, cfg gormlogger.Config) gormlogger.Interface {
	if logger == nil {
		logger = zap.L()
	}
	return &zapGormLogger{
		logger:               logger,
		level:                cfg.LogLevel,
		slowThreshold:        cfg.SlowThreshold,
		ignoreRecordNotFound: cfg.IgnoreRecordNotFoundError,
	}
}

func (l *zapGormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	return &zapGormLogger{
		logger:               l.logger,
		level:                level,
		slowThreshold:        l.slowThreshold,
		ignoreRecordNotFound: l.ignoreRecordNotFound,
	}
}

func (l *zapGormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.level < gormlogger.Info {
		return
	}
	l.logger.Sugar().Infow(msg, "data", data)
}

func (l *zapGormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.level < gormlogger.Warn {
		return
	}
	l.logger.Sugar().Warnw(msg, "data", data)
}

func (l *zapGormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.level < gormlogger.Error {
		return
	}
	l.logger.Sugar().Errorw(msg, "data", data)
}

func (l *zapGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.level == gormlogger.Silent {
		return
	}
	elapsed := time.Since(begin)
	sql, rows := fc()

	if err != nil && (!errors.Is(err, gorm.ErrRecordNotFound) || !l.ignoreRecordNotFound) {
		if l.level >= gormlogger.Error {
			l.logger.Error(
				"gorm error",
				zap.Duration("elapsed", elapsed),
				zap.Int64("rows", rows),
				zap.String("sql", sql),
				zap.Error(err),
			)
		}
		return
	}

	if l.slowThreshold > 0 && elapsed > l.slowThreshold {
		if l.level >= gormlogger.Warn {
			l.logger.Warn(
				"gorm slow query",
				zap.Duration("elapsed", elapsed),
				zap.Int64("rows", rows),
				zap.String("sql", sql),
			)
		}
		return
	}

	if l.level >= gormlogger.Info {
		l.logger.Info(
			"gorm query",
			zap.Duration("elapsed", elapsed),
			zap.Int64("rows", rows),
			zap.String("sql", sql),
		)
	}
}
