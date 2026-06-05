FROM golang:1.24-bookworm AS builder

WORKDIR /src

COPY go.mod ./
COPY main.go ./
COPY internal/ ./internal/

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/ip-anonymizer .

FROM alpine:3.21

RUN adduser -D -u 65532 -g 65532 appuser \
    && mkdir -p /data /mapping \
    && chown -R appuser:appuser /data /mapping

COPY --from=builder /out/ip-anonymizer /usr/local/bin/ip-anonymizer

USER appuser
WORKDIR /data
VOLUME ["/mapping"]

ENTRYPOINT ["/usr/local/bin/ip-anonymizer"]
