package handler

import (
	"context"
	"sync"

	"invoice-app/internal/config"
	"invoice-app/internal/ocr"
	"invoice-app/internal/ocrimport"
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

func (r *OCRJobRunner) Start(sessionID int64, userID int64) {
	r.wg.Add(1)
	go r.process(sessionID, userID)
}

func (r *OCRJobRunner) Wait() {
	r.wg.Wait()
}

func (r *OCRJobRunner) process(sessionID int64, userID int64) {
	defer r.wg.Done()

	store, err := r.stores.ForUser(userID)
	if err != nil {
		return
	}

	_ = r.importer(store).ProcessSession(context.Background(), sessionID)
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
