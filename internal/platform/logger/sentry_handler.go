package logger

import (
	"context"
	"log/slog"
	"runtime"

	"github.com/getsentry/sentry-go"
)

// WrapWithSentry returns a logger that forwards error logs to Sentry.
func WrapWithSentry(base *slog.Logger) *slog.Logger {
	if base == nil {
		return base
	}
	return slog.New(&sentryHandler{next: base.Handler()})
}

type sentryHandler struct {
	next slog.Handler
}

func (h *sentryHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *sentryHandler) Handle(ctx context.Context, record slog.Record) error {
	err := h.next.Handle(ctx, record)
	if record.Level < slog.LevelError {
		return err
	}

	var capturedErr error
	extras := map[string]any{}
	record.Attrs(func(attr slog.Attr) bool {
		if attr.Key != "" {
			extras[attr.Key] = attrValue(attr.Value, &capturedErr)
		}
		return true
	})
	if record.PC != 0 {
		frame, _ := runtime.CallersFrames([]uintptr{record.PC}).Next()
		if frame.PC != 0 {
			extras["source.file"] = frame.File
			extras["source.line"] = frame.Line
			extras["source.function"] = frame.Function
		}
	}

	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetLevel(sentry.LevelError)
		scope.SetExtras(extras)
		scope.SetExtra("message", record.Message)
		if capturedErr != nil {
			sentry.CaptureException(capturedErr)
			return
		}
		sentry.CaptureMessage(record.Message)
	})

	return err
}

func (h *sentryHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &sentryHandler{next: h.next.WithAttrs(attrs)}
}

func (h *sentryHandler) WithGroup(name string) slog.Handler {
	return &sentryHandler{next: h.next.WithGroup(name)}
}

func attrValue(value slog.Value, capturedErr *error) any {
	switch value.Kind() {
	case slog.KindAny:
		anyValue := value.Any()
		if err, ok := anyValue.(error); ok {
			if *capturedErr == nil {
				*capturedErr = err
			}
			return err.Error()
		}
		return anyValue
	case slog.KindBool:
		return value.Bool()
	case slog.KindDuration:
		return value.Duration()
	case slog.KindFloat64:
		return value.Float64()
	case slog.KindInt64:
		return value.Int64()
	case slog.KindString:
		return value.String()
	case slog.KindTime:
		return value.Time()
	case slog.KindUint64:
		return value.Uint64()
	case slog.KindGroup:
		group := map[string]any{}
		for _, attr := range value.Group() {
			if attr.Key == "" {
				continue
			}
			group[attr.Key] = attrValue(attr.Value, capturedErr)
		}
		return group
	default:
		return value.String()
	}
}
