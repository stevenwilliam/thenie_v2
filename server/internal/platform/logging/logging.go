// Package logging provides structured JSON logging with a request id carried
// through context.
//
// The site is public and anonymous -- this service stores no customer data, so
// there is far less to redact than in healthy_catering. What is still worth
// keeping out of a log line is the database URL (it carries a password) and any
// value from sys_parameters, some of which are contact details.
package logging

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

type ctxKey int

const (
	ctxKeyRequestID ctxKey = iota
	ctxKeyLogger
)

// redactKeys are replaced wholesale wherever they appear as an attribute key.
var redactKeys = map[string]bool{
	"database_url": true,
	"dsn":          true,
	"password":     true,
	"secret":       true,
	"token":        true,
	"api_key":      true,
	"admin_token":  true,
}

const redacted = "[REDACTED]"

// New builds the application logger. level is one of debug, info, warn, error.
func New(level string, jsonOutput bool) *slog.Logger {
	var lv slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lv = slog.LevelDebug
	case "warn":
		lv = slog.LevelWarn
	case "error":
		lv = slog.LevelError
	default:
		lv = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: lv, ReplaceAttr: redact}
	var h slog.Handler
	if jsonOutput {
		h = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		h = slog.NewTextHandler(os.Stdout, opts)
	}
	return slog.New(h)
}

// redact replaces sensitive values at write time rather than trusting every
// call site to remember.
func redact(_ []string, a slog.Attr) slog.Attr {
	if redactKeys[strings.ToLower(a.Key)] {
		return slog.String(a.Key, redacted)
	}
	// A connection string can also arrive as a plain message value.
	if a.Value.Kind() == slog.KindString {
		if s := a.Value.String(); strings.Contains(s, "://") && strings.Contains(s, "@") {
			if i := strings.Index(s, "://"); i >= 0 {
				if j := strings.Index(s[i:], "@"); j >= 0 {
					return slog.String(a.Key, s[:i+3]+redacted+s[i+j:])
				}
			}
		}
	}
	return a
}

// WithRequestID returns a context carrying a request id.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKeyRequestID, id)
}

// RequestID reads the request id, or "" when absent.
func RequestID(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyRequestID).(string)
	return v
}

// WithLogger stores a logger on the context.
func WithLogger(ctx context.Context, log *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKeyLogger, log)
}

// FromContext returns the context's logger, or the default one.
func FromContext(ctx context.Context) *slog.Logger {
	if log, ok := ctx.Value(ctxKeyLogger).(*slog.Logger); ok && log != nil {
		return log
	}
	return slog.Default()
}
