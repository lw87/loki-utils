FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src

COPY loki-chunk-verify/go.mod loki-chunk-verify/go.sum ./loki-chunk-verify/

COPY loki-index-parser/go.mod loki-index-parser/go.sum ./loki-index-parser/

RUN cd /src/loki-chunk-verify && go mod download

RUN cd /src/loki-index-parser && go mod download

COPY loki-chunk-verify/ /src/loki-chunk-verify/ 

COPY loki-index-parser/ /src/loki-index-parser/

WORKDIR /src/loki-chunk-verify

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/loki-chunk-verify .

WORKDIR /src/loki-index-parser

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/loki-index-parser .

FROM alpine:3.22

RUN apk add --no-cache ca-certificates && addgroup -S loki && adduser -S -G loki loki

COPY --from=builder /out/loki-chunk-verify /usr/local/bin/loki-chunk-verify
COPY --from=builder /out/loki-index-parser /usr/local/bin/loki-index-parser

USER loki

WORKDIR /tmp

CMD ["sh", "-c", "while true; do sleep 3600; done"]

