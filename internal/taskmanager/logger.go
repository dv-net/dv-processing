package taskmanager

import (
	"context"
	"log/slog"

	"github.com/dv-net/mx/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

/*

	Code below implements slog.Handler interface for zap.Logger

	TODO: Move code below to external package

*/

type zapHandler struct {
	level     slog.Level
	zapLogger *zap.Logger
}

func newZapHandler(zapLogger *zap.Logger, level slog.Level) *zapHandler {
	return &zapHandler{
		zapLogger: zapLogger,
		level:     level,
	}
}

func (h *zapHandler) Enabled(_ context.Context, level slog.Level) bool {
	if level < h.level {
		return false
	}

	switch level {
	case slog.LevelDebug:
		return h.zapLogger.Core().Enabled(zap.DebugLevel)
	case slog.LevelInfo:
		return h.zapLogger.Core().Enabled(zap.InfoLevel)
	case slog.LevelWarn:
		return h.zapLogger.Core().Enabled(zap.WarnLevel)
	case slog.LevelError:
		return h.zapLogger.Core().Enabled(zap.ErrorLevel)
	default:
		return true
	}
}

func (h *zapHandler) Handle(_ context.Context, record slog.Record) error {
	var zapLevel zapcore.Level
	switch record.Level {
	case slog.LevelDebug:
		zapLevel = zap.DebugLevel
	case slog.LevelInfo:
		zapLevel = zap.InfoLevel
	case slog.LevelWarn:
		zapLevel = zap.WarnLevel
	case slog.LevelError:
		zapLevel = zap.ErrorLevel
	default:
		zapLevel = zap.InfoLevel
	}

	fields := []zapcore.Field{}
	record.Attrs(func(attr slog.Attr) bool {
		fields = append(fields, zap.Any(attr.Key, attr.Value))
		return true
	})

	h.zapLogger.Log(zapLevel, record.Message, fields...)
	return nil
}

func (h *zapHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newZapLogger := h.zapLogger.With(convertAttrsToZapFields(attrs)...)
	return newZapHandler(newZapLogger, h.level)
}

func (h *zapHandler) WithGroup(name string) slog.Handler {
	newZapLogger := h.zapLogger.With(zap.Namespace(name))
	return newZapHandler(newZapLogger, h.level)
}

func convertAttrsToZapFields(attrs []slog.Attr) []zap.Field {
	fields := make([]zap.Field, len(attrs))
	for i, attr := range attrs {
		fields[i] = zap.Any(attr.Key, attr.Value)
	}
	return fields
}

func newLogger(l logger.ExtendedLogger) *slog.Logger {
	return slog.New(newZapHandler(l.Sugar().Desugar(), slog.LevelWarn))
}
