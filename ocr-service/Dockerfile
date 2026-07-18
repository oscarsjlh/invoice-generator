FROM ghcr.io/astral-sh/uv:python3.12-bookworm-slim AS builder

WORKDIR /app
COPY pyproject.toml ./

RUN --mount=type=cache,target=/root/.cache/uv \
    uv sync --no-dev

FROM python:3.12-slim-bookworm AS runtime

RUN adduser --system --uid 1000 app

COPY --from=builder --chown=app:app /app/.venv /app/.venv
COPY --chown=app:app main.py ocr_agent.py /app/

ENV PATH="/app/.venv/bin:$PATH" \
    OCR_LISTEN=:8000

USER app
EXPOSE 8000

CMD ["python", "/app/main.py"]
