package geoserver

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
)

// Logger is the printf-style logging surface used by *GeoServer internally.
//
// In v1.0 this was a *logrus.Logger. In v1.1 it is a thin wrapper over
// stdlib log/slog that preserves the same method shape (Errorf, Warnf, Infof,
// Debugf, plus their non-printf siblings) so existing call sites continue to
// compile and behave the same way.
//
// To inject your own slog.Handler, use the [WithLogger] option on [New].
type Logger struct {
	sl *slog.Logger
}

// SLogger returns the underlying *slog.Logger.
func (l *Logger) SLogger() *slog.Logger {
	if l == nil {
		return slog.Default()
	}
	return l.sl
}

func (l *Logger) log(level slog.Level, msg string) {
	if l == nil || l.sl == nil {
		return
	}
	if l.sl.Enabled(context.Background(), level) {
		l.sl.Log(context.Background(), level, msg)
	}
}

// Errorf logs at Error level, formatting like fmt.Sprintf.
func (l *Logger) Errorf(format string, args ...any) {
	l.log(slog.LevelError, fmt.Sprintf(format, args...))
}

// Warnf logs at Warn level, formatting like fmt.Sprintf.
func (l *Logger) Warnf(format string, args ...any) {
	l.log(slog.LevelWarn, fmt.Sprintf(format, args...))
}

// Infof logs at Info level, formatting like fmt.Sprintf.
func (l *Logger) Infof(format string, args ...any) {
	l.log(slog.LevelInfo, fmt.Sprintf(format, args...))
}

// Debugf logs at Debug level, formatting like fmt.Sprintf.
func (l *Logger) Debugf(format string, args ...any) {
	l.log(slog.LevelDebug, fmt.Sprintf(format, args...))
}

// Error logs its arguments at Error level, formatted with fmt.Sprint.
func (l *Logger) Error(args ...any) { l.log(slog.LevelError, fmt.Sprint(args...)) }

// Warn logs its arguments at Warn level, formatted with fmt.Sprint.
func (l *Logger) Warn(args ...any) { l.log(slog.LevelWarn, fmt.Sprint(args...)) }

// Info logs its arguments at Info level, formatted with fmt.Sprint.
func (l *Logger) Info(args ...any) { l.log(slog.LevelInfo, fmt.Sprint(args...)) }

// Debug logs its arguments at Debug level, formatted with fmt.Sprint.
func (l *Logger) Debug(args ...any) { l.log(slog.LevelDebug, fmt.Sprint(args...)) }

// loggerFromHandler wraps an [slog.Handler] in a [*Logger]. Used by
// [WithLogger] in options.go.
func loggerFromHandler(h slog.Handler) *Logger {
	if h == nil {
		return &Logger{sl: slog.New(discardHandler())}
	}
	return &Logger{sl: slog.New(h)}
}

// discardHandler returns an slog.Handler that drops every log record. Used
// when the caller passes a nil handler to [WithLogger]. Equivalent to
// slog.DiscardHandler (Go 1.24+) but compatible with Go 1.23.
func discardHandler() slog.Handler {
	return slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError + 1})
}

// defaultHandler returns the slog.Handler used when no [WithLogger] option is
// supplied. It writes Info-and-above to stderr in text format, mimicking the
// shape of the v1.0 logrus default.
func defaultHandler() slog.Handler {
	return slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
}

// GetLogger returns a default [*Logger] writing Info-and-above to stderr.
//
// Deprecated: configure logging via [WithLogger] passed to [New]. GetLogger
// is preserved as a no-arg convenience and will be removed in v2.
//
// Note: in v1.0 this returned *logrus.Logger. In v1.1 it returns the package
// [*Logger] wrapper around log/slog. This is a soft compatibility break for
// any caller that pinned to the logrus type; the available method surface
// (Errorf, Warnf, Infof, Debugf, Error, Warn, Info, Debug) is preserved.
func GetLogger() *Logger {
	return &Logger{sl: slog.New(defaultHandler())}
}
