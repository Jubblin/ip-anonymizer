FROM golang:1.27-bookworm@sha256:dc0963f32053e4bc451d24d87e3892d42d01021b61f093e98eb95c0352bb84b0 AS builder

WORKDIR /src

COPY go.mod ./
COPY main.go ./
COPY internal/ ./internal/

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/ip-anonymizer .

FROM alpine:3.23

RUN adduser -D -u 65532 -g 65532 appuser \
    && mkdir -p /data /mapping \
    && chown -R appuser:appuser /data /mapping

COPY --from=builder /out/ip-anonymizer /usr/local/bin/ip-anonymizer

USER appuser
WORKDIR /data
VOLUME ["/mapping"]

ENTRYPOINT ["/usr/local/bin/ip-anonymizer"]
