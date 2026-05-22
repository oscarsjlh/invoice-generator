package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"os"
	"time"
)

type contextKey string

const loggerKey contextKey = "logger"

func NewLogger(level, format string) *slog.Logger {
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
	switch format {
	case "text":
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: l})
	default:
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: l})
	}
	return slog.New(handler)
}

func LoggerMiddleware(baseLogger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := generateRequestID()
			logger := baseLogger.With("request_id", requestID)
			ctx := context.WithValue(r.Context(), loggerKey, logger)
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

func RequestLoggingMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, statusCode: 0}
			next.ServeHTTP(rw, r)
			duration := time.Since(start)

			logger := LoggerFromContext(r.Context())
			attrs := []slog.Attr{
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rw.statusCode),
				slog.Int64("duration_ms", duration.Milliseconds()),
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("user_agent", r.UserAgent()),
				slog.Int("bytes_written", rw.bytes),
			}
			if r.Header.Get("HX-Request") == "true" {
				attrs = append(attrs, slog.Bool("hx_request", true))
				if target := r.Header.Get("HX-Target"); target != "" {
					attrs = append(attrs, slog.String("hx_target", target))
				}
			}
			logger.LogAttrs(r.Context(), slog.LevelInfo, "request", attrs...)
		})
	}
}
