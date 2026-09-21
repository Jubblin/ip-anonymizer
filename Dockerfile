FROM golang:1.26-bookworm@sha256:a688600ca24f8a4d3ca77f95b0dd40704a9fc787c826660eb7ba0b641b8b175d AS builder

WORKDIR /src

COPY go.mod ./
COPY main.go ./
COPY internal/ ./internal/

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/ip-anonymizer .

FROM alpine:3.23@sha256:85fe1e81d6758c208f3e1eed4338a1997e19d4be002d4dd32d3100c9a8c010a0

RUN adduser -D -u 65532 -g 65532 appuser \
    && mkdir -p /data /mapping \
    && chown -R appuser:appuser /data /mapping

COPY --from=builder /out/ip-anonymizer /usr/local/bin/ip-anonymizer

USER appuser
WORKDIR /data
VOLUME ["/mapping"]

ENTRYPOINT ["/usr/local/bin/ip-anonymizer"]
