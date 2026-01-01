package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"
)

// CompactHandler formats logs in a compact, readable format for console output
// Format: [LEVEL] HH:MM:SS message key=value key=value
type CompactHandler struct {
	opts  slog.HandlerOptions
	mu    sync.Mutex
	out   io.Writer
	attrs []slog.Attr // accumulated attributes from WithAttrs
	group string      // current group name from WithGroup
}

// NewCompactHandler creates a new compact console handler
func NewCompactHandler(w io.Writer, opts *slog.HandlerOptions) *CompactHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	return &CompactHandler{
		opts: *opts,
		out:  w,
	}
}

func (h *CompactHandler) Enabled(ctx context.Context, level slog.Level) bool {
	minLevel := slog.LevelInfo
	if h.opts.Level != nil {
		minLevel = h.opts.Level.Level()
	}
	return level >= minLevel
}

func (h *CompactHandler) Handle(ctx context.Context, r slog.Record) error {
	buf := make([]byte, 0, 1024)

	// Level with fixed width
	level := r.Level.String()
	switch r.Level {
	case slog.LevelDebug:
		buf = append(buf, "[DEBUG] "...)
	case slog.LevelInfo:
		buf = append(buf, "[INFO]  "...)
	case slog.LevelWarn:
		buf = append(buf, "[WARN]  "...)
	case slog.LevelError:
		buf = append(buf, "[ERROR] "...)
	default:
		buf = append(buf, fmt.Sprintf("[%-5s] ", level)...)
	}

	// Time (just HH:MM:SS for readability)
	t := r.Time.Format("15:04:05")
	buf = append(buf, t...)
	buf = append(buf, ' ')

	// Message
	buf = append(buf, r.Message...)

	// Attributes
	hasAttrs := false
	r.Attrs(func(a slog.Attr) bool {
		// Skip empty attrs
		if a.Equal(slog.Attr{}) {
			return true
		}

		if !hasAttrs {
			// Add separator before first attribute
			buf = append(buf, " |"...)
			hasAttrs = true
		}

		buf = append(buf, ' ')
		buf = h.appendAttr(buf, a)
		return true
	})

	// Newline
	buf = append(buf, '\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.out.Write(buf)
	return err
}

func (h *CompactHandler) appendAttr(buf []byte, a slog.Attr) []byte {
	// Handle special cases
	switch a.Key {
	case "requestID":
		// Shorten request IDs to first 8 chars
		if s, ok := a.Value.Any().(string); ok && len(s) > 8 {
			buf = append(buf, "req="...)
			buf = append(buf, s[:8]...)
			return buf
		}
	case "durationMs":
		// Format duration with "ms" suffix
		buf = append(buf, "duration="...)
		buf = append(buf, a.Value.String()...)
		buf = append(buf, "ms"...)
		return buf
	case "error":
		// Format errors nicely
		buf = append(buf, "error="...)
		buf = append(buf, fmt.Sprintf("%q", a.Value.Any())...)
		return buf
	}

	// Default formatting: key=value
	buf = append(buf, a.Key...)
	buf = append(buf, '=')

	// Format value based on type
	v := a.Value
	switch v.Kind() {
	case slog.KindString:
		s := v.String()
		// Quote strings with spaces or special chars
		if needsQuoting(s) {
			buf = append(buf, fmt.Sprintf("%q", s)...)
		} else {
			buf = append(buf, s...)
		}
	case slog.KindInt64:
		buf = append(buf, fmt.Sprintf("%d", v.Int64())...)
	case slog.KindUint64:
		buf = append(buf, fmt.Sprintf("%d", v.Uint64())...)
	case slog.KindFloat64:
		buf = append(buf, fmt.Sprintf("%g", v.Float64())...)
	case slog.KindBool:
		buf = append(buf, fmt.Sprintf("%t", v.Bool())...)
	case slog.KindDuration:
		buf = append(buf, v.Duration().String()...)
	case slog.KindTime:
		buf = append(buf, v.Time().Format(time.RFC3339)...)
	default:
		buf = append(buf, fmt.Sprintf("%v", v.Any())...)
	}

	return buf
}

func needsQuoting(s string) bool {
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '"' || r == '=' {
			return true
		}
	}
	return false
}

func (h *CompactHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &CompactHandler{
		opts:  h.opts,
		out:   h.out,
		attrs: append(h.attrs, attrs...),
		group: h.group,
	}
}

func (h *CompactHandler) WithGroup(name string) slog.Handler {
	return &CompactHandler{
		opts:  h.opts,
		out:   h.out,
		attrs: h.attrs,
		group: name,
	}
}
