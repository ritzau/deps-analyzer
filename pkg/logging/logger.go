package logging

import (
	"context"
	"log/slog"
	"os"
)

// contextKey is a type for context keys to avoid collisions
type contextKey string

const requestIDKey contextKey = "requestID"

var logger *slog.Logger

func init() {
	// Initialize with compact handler for readable console output
	// Can be replaced with JSON handler for production
	handler := NewCompactHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo, // Default level
	})
	logger = slog.New(handler)
}

// SetLevel changes the logging level
func SetLevel(level slog.Level) {
	handler := NewCompactHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
	logger = slog.New(handler)
}

// SetJSONOutput switches to JSON format output
func SetJSONOutput(level slog.Level) {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
	logger = slog.New(handler)
}

// WithRequestID adds a request ID to the context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// GetRequestID retrieves the request ID from context
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(requestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// Helper function to add request ID to log attributes if present
func withRequestID(ctx context.Context, args []any) []any {
	requestID := GetRequestID(ctx)
	if requestID != "" {
		return append([]any{"requestID", requestID}, args...)
	}
	return args
}

// Trace logs at TRACE level (very verbose, debug-time only)
func Trace(msg string, args ...any) {
	logger.Log(context.Background(), slog.LevelDebug-4, msg, args...)
}

// TraceContext logs at TRACE level with context
func TraceContext(ctx context.Context, msg string, args ...any) {
	logger.Log(ctx, slog.LevelDebug-4, msg, withRequestID(ctx, args)...)
}

// Debug logs at DEBUG level (internal component behavior)
func Debug(msg string, args ...any) {
	logger.Debug(msg, args...)
}

// DebugContext logs at DEBUG level with context
func DebugContext(ctx context.Context, msg string, args ...any) {
	logger.DebugContext(ctx, msg, withRequestID(ctx, args)...)
}

// Info logs at INFO level (user-facing operations)
func Info(msg string, args ...any) {
	logger.Info(msg, args...)
}

// InfoContext logs at INFO level with context
func InfoContext(ctx context.Context, msg string, args ...any) {
	logger.InfoContext(ctx, msg, withRequestID(ctx, args)...)
}

// Warn logs at WARN level (should be monitored)
func Warn(msg string, args ...any) {
	logger.Warn(msg, args...)
}

// WarnContext logs at WARN level with context
func WarnContext(ctx context.Context, msg string, args ...any) {
	logger.WarnContext(ctx, msg, withRequestID(ctx, args)...)
}

// Error logs at ERROR level (logical bugs that shouldn't happen)
func Error(msg string, args ...any) {
	logger.Error(msg, args...)
}

// ErrorContext logs at ERROR level with context
func ErrorContext(ctx context.Context, msg string, args ...any) {
	logger.ErrorContext(ctx, msg, withRequestID(ctx, args)...)
}

// Fatal logs at ERROR level and exits (unrecoverable bugs)
func Fatal(msg string, args ...any) {
	logger.Error(msg, args...)
	os.Exit(1)
}

// FatalContext logs at ERROR level with context and exits
func FatalContext(ctx context.Context, msg string, args ...any) {
	logger.ErrorContext(ctx, msg, withRequestID(ctx, args)...)
	os.Exit(1)
}
