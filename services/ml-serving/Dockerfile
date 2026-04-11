FROM python:3.12-slim-bookworm

RUN apt-get update && apt-get install -y --no-install-recommends libgomp1 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
# CPU-сборки torch/torchvision без CUDA (меньше образ и быстрее сборка).
RUN pip install --no-cache-dir --upgrade pip && \
    pip install --no-cache-dir \
      "torch>=2.9.0,<3" "torchvision>=0.24.0,<1" \
      --index-url https://download.pytorch.org/whl/cpu && \
    pip install --no-cache-dir \
      "numpy>=2.1.0" "Pillow>=10.4.0" "fastapi>=0.115.0" "uvicorn[standard]>=0.32.0" \
      "python-multipart>=0.0.9" "httpx>=0.27.0" "python-dotenv>=1.0.0" "prometheus-client>=0.20.0"

COPY api/ ./api/
COPY inference_core.py .

ENV PYTHONPATH=/app
EXPOSE 8000
CMD ["uvicorn", "api.main:app", "--host", "0.0.0.0", "--port", "8000"]
