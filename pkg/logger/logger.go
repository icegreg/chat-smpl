package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Config for logger initialization
type Config struct {
	Level       string
	Development bool
}

// Logger interface for structured logging
type Logger interface {
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
	Fatal(msg string, keysAndValues ...interface{})
	With(keysAndValues ...interface{}) Logger
	Sync() error
}

// zapLogger wraps zap.SugaredLogger to implement Logger interface
type zapLogger struct {
	sugar *zap.SugaredLogger
}

func (l *zapLogger) Debug(msg string, keysAndValues ...interface{}) {
	l.sugar.Debugw(msg, keysAndValues...)
}

func (l *zapLogger) Info(msg string, keysAndValues ...interface{}) {
	l.sugar.Infow(msg, keysAndValues...)
}

func (l *zapLogger) Warn(msg string, keysAndValues ...interface{}) {
	l.sugar.Warnw(msg, keysAndValues...)
}

func (l *zapLogger) Error(msg string, keysAndValues ...interface{}) {
	l.sugar.Errorw(msg, keysAndValues...)
}

func (l *zapLogger) Fatal(msg string, keysAndValues ...interface{}) {
	l.sugar.Fatalw(msg, keysAndValues...)
}

func (l *zapLogger) With(keysAndValues ...interface{}) Logger {
	return &zapLogger{sugar: l.sugar.With(keysAndValues...)}
}

func (l *zapLogger) Sync() error {
	return l.sugar.Sync()
}

var log *zap.Logger

func Init(level string, development bool) error {
	var config zap.Config

	if development {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		config = zap.NewProductionConfig()
	}

	parsedLevel, err := zapcore.ParseLevel(level)
	if err != nil {
		parsedLevel = zapcore.InfoLevel
	}
	config.Level = zap.NewAtomicLevelAt(parsedLevel)

	logger, err := config.Build(zap.AddCallerSkip(1))
	if err != nil {
		return err
	}

	log = logger
	return nil
}

func InitDefault() {
	env := os.Getenv("ENV")
	development := env == "" || env == "development"

	level := os.Getenv("LOG_LEVEL")
	if level == "" {
		if development {
			level = "debug"
		} else {
			level = "info"
		}
	}

	if err := Init(level, development); err != nil {
		panic(err)
	}
}

func Get() *zap.Logger {
	if log == nil {
		InitDefault()
	}
	return log
}

// New creates a new Logger with the given config
func New(cfg Config) Logger {
	if err := Init(cfg.Level, cfg.Development); err != nil {
		panic(err)
	}
	return &zapLogger{sugar: Get().Sugar()}
}

func Sync() {
	if log != nil {
		_ = log.Sync()
	}
}

func Debug(msg string, fields ...zap.Field) {
	Get().Debug(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	Get().Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	Get().Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	Get().Error(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	Get().Fatal(msg, fields...)
}

func With(fields ...zap.Field) *zap.Logger {
	return Get().With(fields...)
}
