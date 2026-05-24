package ocr

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestClientExtractInjectsTraceparent(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "ocr-*.jpg")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	if _, err := tmp.WriteString("image data"); err != nil {
		t.Fatalf("write temp image: %v", err)
	}
	if err := tmp.Close(); err != nil {
		t.Fatalf("close temp image: %v", err)
	}

	provider := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()))
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	defer func() {
		_ = provider.Shutdown(context.Background())
	}()

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if got := req.Header.Get("traceparent"); got == "" {
			t.Fatal("traceparent header was not injected")
		}
		if got := req.Header.Get("Content-Type"); !strings.HasPrefix(got, "multipart/form-data;") {
			t.Fatalf("Content-Type = %q, want multipart/form-data", got)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`{
				"session_id": 42,
				"status": "success",
				"entries": [],
				"metadata": {"model_used": "test"}
			}`)),
			Header: make(http.Header),
		}, nil
	})

	client := &Client{
		baseURL: "http://ocr-service",
		httpClient: &http.Client{
			Transport: otelhttp.NewTransport(transport),
		},
	}

	ctx, span := otel.Tracer("invoice-app/ocr-test").Start(context.Background(), "test")
	defer span.End()

	if _, err := client.Extract(ctx, []string{tmp.Name()}, ContextHint{}, nil, 42); err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
