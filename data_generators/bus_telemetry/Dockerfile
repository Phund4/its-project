# Build from repository root: docker build -f data_generators/bus_telemetry/Dockerfile .
FROM golang:1.24-bookworm AS builder
ENV GOTOOLCHAIN=auto
WORKDIR /src
COPY services/data_ingestion ./services/data_ingestion
COPY data_generators/bus_telemetry ./data_generators/bus_telemetry
WORKDIR /src/data_generators/bus_telemetry
RUN go mod tidy && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /generator .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /generator /generator
USER nonroot:nonroot
ENTRYPOINT ["/generator"]
