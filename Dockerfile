FROM golang:1.26-bookworm@sha256:9fdc884aacc3bec89b20ffc69f4bb369c78210e3e4f600387b5128b12c199f81 AS builder

WORKDIR /src

COPY go.mod ./
COPY main.go ./
COPY internal/ ./internal/

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/ip-anonymizer .

FROM alpine:3.23@sha256:fd791d74b68913cbb027c6546007b3f0d3bc45125f797758156952bc2d6daf40

RUN adduser -D -u 65532 -g 65532 appuser \
    && mkdir -p /data /mapping \
    && chown -R appuser:appuser /data /mapping

COPY --from=builder /out/ip-anonymizer /usr/local/bin/ip-anonymizer

USER appuser
WORKDIR /data
VOLUME ["/mapping"]

ENTRYPOINT ["/usr/local/bin/ip-anonymizer"]
