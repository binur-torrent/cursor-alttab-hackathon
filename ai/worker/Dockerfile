# LumiCity AI worker. Build context must be ./ai so the shared `pipeline` package
# is available to the worker (see deployments/render.yaml -> dockerContext: ./ai).
FROM python:3.11-slim

WORKDIR /app

# System deps for opencv-python-headless.
RUN apt-get update && apt-get install -y --no-install-recommends \
    libgl1 libglib2.0-0 && rm -rf /var/lib/apt/lists/*

COPY worker/requirements.txt /app/worker/requirements.txt
RUN pip install --no-cache-dir -r /app/worker/requirements.txt

# Copy shared pipeline package + worker code.
COPY pipeline /app/pipeline
COPY worker /app/worker

ENV MODEL_BACKEND=heuristic
EXPOSE 8000

# Render injects $PORT; default to 8000 locally.
CMD ["sh", "-c", "uvicorn worker.main:app --host 0.0.0.0 --port ${PORT:-8000}"]
