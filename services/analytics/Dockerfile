FROM golang:1.22-bookworm AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/analytics ./cmd/analytics

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/analytics /analytics
USER nonroot:nonroot
ENTRYPOINT ["/analytics"]
