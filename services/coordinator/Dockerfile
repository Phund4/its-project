FROM golang:1.23-bookworm AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/coordinator ./cmd/coordinator

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/coordinator /coordinator
USER nonroot:nonroot
ENTRYPOINT ["/coordinator"]
