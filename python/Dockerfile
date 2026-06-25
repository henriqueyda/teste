FROM python:3.12-slim

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends build-essential \
    && rm -rf /var/lib/apt/lists/*

# Install deps before copying source for better layer caching.
# The stub __init__ lets pip resolve the local package without the real source.
COPY pyproject.toml .
RUN mkdir -p app && touch app/__init__.py && pip install --no-cache-dir .

COPY . .

EXPOSE 8000
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000"]
