package handler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel/trace"
)

type contextKey string

const loggerKey contextKey = "logger"

func NewLogger(level, format string, includeSource bool) *slog.Logger {
	var l slog.Level
	switch level {
	case "debug":
		l = slog.LevelDebug
	case "info":
		l = slog.LevelInfo
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}

	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: l, AddSource: includeSource}
	switch format {
	case "text":
		handler = slog.NewTextHandler(os.Stdout, opts)
	default:
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}
	return slog.New(handler)
}

func LoggerMiddleware(baseLogger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := generateRequestID()
			logger := baseLogger.With("request_id", requestID)
			if spanContext := trace.SpanContextFromContext(r.Context()); spanContext.IsValid() {
				logger = logger.With(
					"trace_id", spanContext.TraceID().String(),
					"span_id", spanContext.SpanID().String(),
				)
			}
			ctx := context.WithValue(r.Context(), loggerKey, logger)
			w.Header().Set("X-Request-ID", requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func LoggerFromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return logger
	}
	return slog.New(slog.NewJSONHandler(noopWriter{}, &slog.HandlerOptions{Level: slog.LevelError + 1}))
}

type noopWriter struct{}

func (w noopWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func generateRequestID() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "000000000000"
	}
	return hex.EncodeToString(b)
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	bytes      int
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.statusCode != 0 {
		return
	}
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.bytes += n
	return n, err
}

func (rw *responseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (rw *responseWriter) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := rw.ResponseWriter.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}
	return http.ErrNotSupported
}

func RequestLoggingMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, statusCode: 0}
			next.ServeHTTP(rw, r)
			duration := time.Since(start)

			logger := LoggerFromContext(r.Context())
			attrs := []slog.Attr{
				slog.String("event", "http_request"),
				slog.String("component", "http"),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("route", r.Pattern),
				slog.Int("status", rw.statusCode),
				slog.Int64("duration_ms", duration.Milliseconds()),
				slog.String("remote_addr", redactRemoteAddr(r.RemoteAddr)),
				slog.String("user_agent", r.UserAgent()),
				slog.Int("bytes_written", rw.bytes),
			}
			if user := UserFromContext(r.Context()); user != nil {
				attrs = append(attrs, slog.Int64("user_id", user.ID))
			}
			if r.Header.Get("HX-Request") == "true" {
				attrs = append(attrs, slog.Bool("hx_request", true))
				if target := r.Header.Get("HX-Target"); target != "" {
					attrs = append(attrs, slog.String("hx_target", target))
				}
			}
			logger.LogAttrs(r.Context(), slog.LevelInfo, "http_request", attrs...)
		})
	}
}

func redactRemoteAddr(addr string) string {
	if addr == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(addr))
	return fmt.Sprintf("sha256:%x", sum[:8])
}

func RedactEmail(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(trimmed))
	return fmt.Sprintf("sha256:%x", sum[:8])
}
