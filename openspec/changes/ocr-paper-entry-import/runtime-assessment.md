## OCR Runtime Assessment

This document records the evaluation of `olmocr` (https://github.com/allenai/olmocr) as the OCR foundation for the paper-entry import workflow.

### Runtime Option Evaluation

**Option A: Local GPU Inference**

olmOCR is a 7B-parameter vision-language model (VLM) based on Qwen2.5-VL, using vLLM for inference.

| Requirement | Minimum | Recommended |
|---|---|---|
| GPU | NVIDIA GPU with 12 GB VRAM (RTX 4090, L40S) | A100 (40/80 GB), H100 |
| Disk | 30 GB (model + dependencies) | 50 GB |
| RAM | 16 GB system | 32 GB |
| Software | CUDA 12.8, Python 3.11, vLLM | Docker with `alleninstituteforai/olmocr:latest-with-model` |

Local inference uses `pip install olmocr[gpu]` and runs vLLM locally on port 8000.

**Option B: Remote Inference (Hosted Provider)**

Use a hosted vLLM endpoint via the `--server` flag. Verified providers:

| Provider | Input $/1M tokens | Output $/1M tokens |
|---|---|---|
| Cirrascale | $0.07 | $0.15 |
| DeepInfra | $0.09 | $0.19 |
| Parasail | $0.10 | $0.20 |

The `--server` flag points to an OpenAI-compatible `/v1` endpoint.

**Option C: Docker Deployment**

Pre-built Docker image (30 GB with model): `alleninstituteforai/olmocr:latest-with-model`. Requires `--gpus all` flag.

**Option D: Lightweight Client Only**

`pip install olmocr` installs only the client (~50 MB, no GPU deps). Requires an external vLLM server.

### Recommendation

For the invoice-generator use case, the recommended approach is:

1. **Primary**: Use a hosted vLLM endpoint (Option B) with the lightweight client. This avoids GPU requirements and is cost-effective for intermittent usage (~few pages per session).
2. **Alternative for offline/private use**: Docker deployment (Option C) with a GPU-capable machine. Minimum 12 GB VRAM.
3. **Not recommended**: Running vLLM locally without a dedicated GPU — the 7B model is too large for CPU inference at acceptable speed.

### Handwriting Suitability

olmOCR advertises handwriting support in its feature list. The model is trained on diverse document types including handwritten content. However, for the specific use case of handwritten work-log pages (rotated, spiral-bound, mixed ink), additional image preprocessing (rotation correction, cropping) is recommended before OCR.

### Model Details

- Current model: `allenai/olmOCR-2-7B-1025-FP8` (v0.4.0, FP8 quantization)
- API: OpenAI-compatible (`/v1/chat/completions`)
- Max context: 16384 tokens (recommended `--max-model-len`)
- Guided decoding available for structured YAML output
