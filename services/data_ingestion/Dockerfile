# Build
FROM golang:1.22-bookworm AS builder
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /ingest ./cmd/ingest

# Runtime: ffmpeg for RTSP / demux
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ffmpeg ca-certificates \
    && rm -rf /var/lib/apt/lists/*
COPY --from=builder /ingest /usr/local/bin/ingest
WORKDIR /work
ENTRYPOINT ["/usr/local/bin/ingest"]
