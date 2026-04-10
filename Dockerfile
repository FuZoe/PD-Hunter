# === Stage 1: Build Go CLI ===
FROM golang:1.22-alpine AS go-builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY pkg/ pkg/
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /hunter ./cmd/hunter

# === Stage 2: Runtime ===
FROM python:3.11-slim
WORKDIR /app

# Install Go binary
COPY --from=go-builder /hunter /usr/local/bin/hunter

# Install Python dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application files
COPY mapping.json .
COPY enrich_bounties.py .

ENTRYPOINT ["hunter"]
CMD ["--help"]
