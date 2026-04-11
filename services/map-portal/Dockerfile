# Build context: services/ (docker compose: context ../services)
FROM golang:1.23-bookworm AS builder
WORKDIR /src/map-portal
COPY map-portal/ .
COPY analytics/ ../analytics/
RUN go mod download
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/map-portal ./cmd/map-portal

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/map-portal /map-portal
USER nonroot:nonroot
ENTRYPOINT ["/map-portal"]
