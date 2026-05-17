# OCR Paper Entry Import

This document covers setup and operation of the optional OCR-based paper entry import feature.

## Overview

The OCR import workflow lets you photograph handwritten timesheet pages and have the system extract draft entries using AWS Bedrock (Claude). Extracted entries are presented for review and confirmation before being saved to the ledger.

## Architecture

```
[Web App] --HTTP--> [OCR Service] --> [AWS Bedrock]
                         |                  |
                   (Python, stdlib      (Claude 3.5 Haiku)
                    HTTP server)
```

The Go web app calls the OCR service over HTTP. The OCR service uses AWS Bedrock with Claude for handwriting extraction.

## Enabling OCR

Set these environment variables:

```bash
export OCR_ENABLED=true
export OCR_SERVICE_URL=http://localhost:8000   # OCR service endpoint
export OCR_UPLOAD_DIR=data/ocr-uploads          # image storage directory
```

Without `OCR_SERVICE_URL`, the app uses a built-in stub client that returns sample data for development and testing.

## LLM Backend: AWS Bedrock

Uses Claude 3.5 Haiku via Bedrock Converse API. No GPU required. Zero infrastructure to maintain.

```bash
export BEDROCK_REGION=us-east-1
export BEDROCK_MODEL=us.anthropic.claude-3-5-haiku-20241022-v1:0
```

IAM permissions needed: `bedrock:InvokeModel` on the model resource.

**Pricing (Claude 3.5 Haiku, on-demand):**

| Direction | $/1M tokens |
|---|---|
| Input | $0.25 |
| Output | $1.25 |

**Cost estimate for 20 handwritten pages:**

| Item | Tokens | Cost |
|---|---|---|
| 20 images (~1500 tokens each) | 30,000 input | $0.008 |
| Structured output (~300 tokens/page) | 6,000 output | $0.008 |
| **Total per 20-page scan** | | **~$0.016** |

With Claude 3.5 Sonnet ($3/$15 per 1M tokens): ~$0.10–$0.20 per 20-page scan.

Images are resized to max 2048px before sending, reducing token cost. Actual token counts vary with page content density and handwriting amount.

## Docker Compose

The `docker-compose.yml` includes an `ocr-service` container. Start both:

```bash
docker compose up --build
```

Mount your AWS credentials (ensure `~/.aws/credentials` is available in the container or set `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_REGION` env vars).

## Image Recommendations

For best OCR results with handwritten timesheet pages:

1. **Lighting**: Good even lighting, no strong shadows
2. **Orientation**: Pages should be roughly upright (the system does basic scaling but not rotation correction)
3. **Contrast**: Dark ink on light paper works best
4. **Format**: JPEG or PNG, captured at reasonable resolution (smartphone photos work well)

The system preprocesses images by:
- Resizing to max 2048px on the longest dimension
- Validating JPEG/PNG format

## Workflow

1. Navigate to `/ocr/import`
2. Upload one or more photographed timesheet pages
3. The system processes images using OCR with your existing rates/categories as hints
4. Review extracted entries on the session page:
   - Green confidence badge (≥80%): Good match against known categories
   - Yellow badge (50-79%): Partial match, review recommended
   - Red badge (<50%): Low confidence, manual correction needed
5. Check the boxes for entries you want to keep
6. Click "Confirm Selected Entries" to save them to your entries list

## Cleanup

Failed and fully-confirmed sessions older than a configured threshold are automatically cleaned up. Call `CleanupStaleOCRSessions(hours)` to remove old sessions and their associated images.

## Fallback

When `OCR_ENABLED=false` (default), the OCR import routes return 404 and the Import link is hidden. The existing manual entry workflow is unaffected.
