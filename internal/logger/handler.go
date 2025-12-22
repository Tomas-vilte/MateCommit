package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fatih/color"
)

// PrettyHandler is a custom slog.Handler for human-friendly CLI output
type PrettyHandler struct {
	opts   *slog.HandlerOptions
	w      io.Writer
	attrs  []slog.Attr
	groups []string
}

func NewPrettyHandler(w io.Writer, opts *slog.HandlerOptions) *PrettyHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	return &PrettyHandler{
		opts:  opts,
		w:     w,
		attrs: []slog.Attr{},
	}
}

func (h *PrettyHandler) Enabled(_ context.Context, level slog.Level) bool {
	minLevel := slog.LevelWarn
	if h.opts.Level != nil {
		minLevel = h.opts.Level.Level()
	}
	return level >= minLevel
}

func (h *PrettyHandler) Handle(_ context.Context, r slog.Record) error {
	var buf strings.Builder

	// Level badge with color
	levelStr := h.formatLevel(r.Level)
	buf.WriteString(levelStr)
	buf.WriteString(" ")

	// Message
	buf.WriteString(r.Message)

	// Attributes
	attrs := make([]string, 0)
	r.Attrs(func(a slog.Attr) bool {
		attrs = append(attrs, h.formatAttr(a))
		return true
	})

	// Add pre-existing attributes
	for _, a := range h.attrs {
		attrs = append(attrs, h.formatAttr(a))
	}

	if len(attrs) > 0 {
		buf.WriteString(" ")
		buf.WriteString(strings.Join(attrs, " "))
	}

	// Source (only in debug mode)
	if h.opts.AddSource && r.PC != 0 {
		fs := slog.Source{
			Function: "",
			File:     "",
			Line:     0,
		}
		// Get source from PC using runtime
		if r.PC != 0 {
			frame, _ := runtime.CallersFrames([]uintptr{r.PC}).Next()
			fs.File = frame.File
			fs.Line = frame.Line
			fs.Function = frame.Function
		}

		if fs.File != "" {
			file := filepath.Base(fs.File)
			source := color.HiBlackString("(%s:%d)", file, fs.Line)
			buf.WriteString(" ")
			buf.WriteString(source)
		}
	}

	buf.WriteString("\n")
	_, err := h.w.Write([]byte(buf.String()))
	return err
}

func (h *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)

	return &PrettyHandler{
		opts:   h.opts,
		w:      h.w,
		attrs:  newAttrs,
		groups: h.groups,
	}
}

func (h *PrettyHandler) WithGroup(name string) slog.Handler {
	newGroups := make([]string, len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups[len(h.groups)] = name

	return &PrettyHandler{
		opts:   h.opts,
		w:      h.w,
		attrs:  h.attrs,
		groups: newGroups,
	}
}

func (h *PrettyHandler) formatLevel(level slog.Level) string {
	var badge string

	switch level {
	case slog.LevelDebug:
		badge = color.HiBlackString("[DEBUG]")
	case slog.LevelInfo:
		badge = color.CyanString("[INFO] ")
	case slog.LevelWarn:
		badge = color.YellowString("[WARN] ")
	case slog.LevelError:
		badge = color.RedString("[ERROR]")
	default:
		badge = fmt.Sprintf("[%s]", level.String())
	}

	return badge
}

func (h *PrettyHandler) formatAttr(a slog.Attr) string {
	key := a.Key
	val := a.Value.String()

	// Apply group prefix if any
	if len(h.groups) > 0 {
		key = strings.Join(h.groups, ".") + "." + key
	}

	// Color-code certain keys
	switch key {
	case "error", "err":
		return color.RedString("%s=%s", key, val)
	case "duration_ms", "duration":
		return color.MagentaString("%s=%s", key, val)
	case "count", "total", "size":
		return color.GreenString("%s=%s", key, val)
	default:
		return color.HiBlackString("%s=%s", key, val)
	}
}
