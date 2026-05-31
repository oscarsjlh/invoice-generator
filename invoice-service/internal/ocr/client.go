package ocr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

// Extractor is the interface for extracting data from OCR images.
type Extractor interface {
	Extract(ctx context.Context, images []string, hints ContextHint, rates []RateHint, sessionID int64) (*OCRResponse, error)
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout:   120 * time.Second,
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
	}
}

func (c *Client) Extract(ctx context.Context, images []string, hints ContextHint, rates []RateHint, sessionID int64) (*OCRResponse, error) {
	ctx, span := otel.Tracer("invoice-app/ocr").Start(ctx, "ocr.extract")
	defer span.End()
	span.SetAttributes(
		attribute.Int64("ocr.session_id", sessionID),
		attribute.Int("ocr.image_count", len(images)),
	)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	reqData := OCRRequest{
		Hints:       hints,
		Rates:       rates,
		SessionID:   sessionID,
		CurrentYear: time.Now().Year(),
	}
	jsonBytes, err := json.Marshal(reqData)
	if err != nil {
		recordSpanError(span, err)
		return nil, fmt.Errorf("marshal ocr request: %w", err)
	}
	if err := writer.WriteField("request", string(jsonBytes)); err != nil {
		recordSpanError(span, err)
		return nil, fmt.Errorf("write request field: %w", err)
	}

	for _, imgPath := range images {
		file, err := os.Open(imgPath)
		if err != nil {
			recordSpanError(span, err)
			return nil, fmt.Errorf("open image %s: %w", imgPath, err)
		}
		part, err := writer.CreateFormFile("images", filepath.Base(imgPath))
		if err != nil {
			_ = file.Close()
			recordSpanError(span, err)
			return nil, fmt.Errorf("create form file: %w", err)
		}
		if _, err := io.Copy(part, file); err != nil {
			_ = file.Close()
			recordSpanError(span, err)
			return nil, fmt.Errorf("copy image data: %w", err)
		}
		if err := file.Close(); err != nil {
			recordSpanError(span, err)
			return nil, fmt.Errorf("close image %s: %w", imgPath, err)
		}
	}

	if err := writer.Close(); err != nil {
		recordSpanError(span, err)
		return nil, fmt.Errorf("close writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/ocr/extract", &buf)
	if err != nil {
		recordSpanError(span, err)
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		recordSpanError(span, err)
		return nil, fmt.Errorf("ocr service request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		recordSpanError(span, err)
		return nil, fmt.Errorf("read ocr response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("ocr service error %d: %s", resp.StatusCode, string(body))
		recordSpanError(span, err)
		return nil, err
	}

	var result OCRResponse
	if err := json.Unmarshal(body, &result); err != nil {
		recordSpanError(span, err)
		return nil, fmt.Errorf("unmarshal ocr response: %w", err)
	}
	span.SetAttributes(attribute.Int("ocr.entries_extracted", len(result.Entries)))

	return &result, nil
}

type StubClient struct{}

func NewStubClient() *StubClient {
	return &StubClient{}
}

func (s *StubClient) Extract(ctx context.Context, images []string, hints ContextHint, rates []RateHint, sessionID int64) (*OCRResponse, error) {
	entries := []OCRExtractedEntry{
		{
			Date: OCRExtractedField{
				Raw: "2026-05-15", Normalized: "2026-05-15", Confidence: 0.95, NeedsReview: false,
			},
			Category: OCRExtractedField{
				Raw: "Acro4", Normalized: "Acro4", Candidates: []string{"Acro4", "Acro 4"}, Confidence: 0.88, NeedsReview: false,
			},
			Hours: OCRExtractedField{
				Raw: "7.5", Normalized: "7.5", Confidence: 0.92, NeedsReview: false,
			},
			Notes: OCRExtractedField{
				Raw: "session prep", Normalized: "session prep", Confidence: 0.70, NeedsReview: false,
			},
		},
		{
			Date: OCRExtractedField{
				Raw: "2026-05-1\u00a06", Normalized: "2026-05-16", Confidence: 0.78, NeedsReview: false,
			},
			Category: OCRExtractedField{
				Raw: "Rncp+n", Normalized: "", Candidates: []string{"Reception"}, Confidence: 0.45, NeedsReview: true,
			},
			Hours: OCRExtractedField{
				Raw: "3", Normalized: "3.0", Confidence: 0.85, NeedsReview: false,
			},
			Notes: OCRExtractedField{
				Raw: "coyer", Normalized: "cover", Confidence: 0.60, NeedsReview: false,
			},
		},
	}

	return &OCRResponse{
		SessionID: sessionID,
		Status:    "success",
		Entries:   entries,
		Metadata: OCRMetadata{
			ModelUsed:        "olmOCR-2-7B-1025-FP8 (stub)",
			ProcessingTimeMs: 2500,
			PagesProcessed:   len(images),
			AvgConfidence:    0.76,
		},
	}, nil
}

type spanErrorRecorder interface {
	RecordError(error, ...trace.EventOption)
	SetStatus(codes.Code, string)
}

func recordSpanError(span spanErrorRecorder, err error) {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}
