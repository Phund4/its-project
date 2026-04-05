# Build from repository root: docker build -f data-generators/bus-telemetry/Dockerfile .
FROM golang:1.24-bookworm AS builder
ENV GOTOOLCHAIN=auto
WORKDIR /src
COPY services/data-ingestion ./services/data-ingestion
COPY data-generators/bus-telemetry ./data-generators/bus-telemetry
WORKDIR /src/data-generators/bus-telemetry
RUN go mod tidy && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /generator .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /generator /generator
USER nonroot:nonroot
ENTRYPOINT ["/generator"]
