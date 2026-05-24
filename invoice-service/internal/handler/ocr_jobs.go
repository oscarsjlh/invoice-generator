package handler

import (
	"context"
	"sync"

	"invoice-app/internal/config"
	"invoice-app/internal/ocr"
	"invoice-app/internal/ocrimport"

	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/trace"
)

type OCRJobRunner struct {
	stores    *RequestStoreProvider
	cfg       *config.Config
	extractor ocr.Extractor
	wg        sync.WaitGroup
}

func NewOCRJobRunner(stores *RequestStoreProvider, cfg *config.Config) *OCRJobRunner {
	return &OCRJobRunner{stores: stores, cfg: cfg}
}

func (r *OCRJobRunner) SetExtractor(extractor ocr.Extractor) {
	r.extractor = extractor
}

func (r *OCRJobRunner) Start(ctx context.Context, sessionID int64, userID int64) {
	r.wg.Add(1)
	go r.process(detachedTraceContext(ctx), sessionID, userID)
}

func (r *OCRJobRunner) Wait() {
	r.wg.Wait()
}

func (r *OCRJobRunner) process(ctx context.Context, sessionID int64, userID int64) {
	defer r.wg.Done()

	store, err := r.stores.ForUser(userID)
	if err != nil {
		return
	}

	_ = r.importer(newTracedStore(ctx, store)).ProcessSession(ctx, sessionID)
}

func (r *OCRJobRunner) importer(store ocrimport.Store) *ocrimport.Importer {
	extractor := r.extractor
	if extractor == nil && r.cfg.OCRServiceURL != "" {
		extractor = ocr.NewClient(r.cfg.OCRServiceURL)
	}
	return &ocrimport.Importer{
		Store:     store,
		Extractor: extractor,
		UploadDir: r.cfg.OCRUploadDir,
	}
}

func detachedTraceContext(ctx context.Context) context.Context {
	detached := context.Background()
	if spanContext := trace.SpanContextFromContext(ctx); spanContext.IsValid() {
		detached = trace.ContextWithSpanContext(detached, spanContext)
	}
	if bag := baggage.FromContext(ctx); bag.Len() > 0 {
		detached = baggage.ContextWithBaggage(detached, bag)
	}
	return detached
}
