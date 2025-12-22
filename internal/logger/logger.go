package logger

import (
	"context"
	"log/slog"
	"os"
)

type contextKey struct{}

var loggerKey = contextKey{}

func Initialize(debug, verbose bool) {
	level := slog.LevelWarn

	if debug {
		level = slog.LevelDebug
	} else if verbose {
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: debug,
	}

	if debug {
		opts.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
			return a
		}
	}

	handler := slog.NewTextHandler(os.Stderr, opts)
	slog.SetDefault(slog.New(handler))
}

func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

func With(ctx context.Context, args ...any) context.Context {
	l := FromContext(ctx).With(args...)
	return WithLogger(ctx, l)
}

func Debug(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).Debug(msg, args...)
}

func Info(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).Info(msg, args...)
}

func Warn(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).Warn(msg, args...)
}

func Error(ctx context.Context, msg string, err error, args ...any) {
	if err != nil {
		args = append(args, slog.Any("error", err))
	}
	FromContext(ctx).Error(msg, args...)
}
