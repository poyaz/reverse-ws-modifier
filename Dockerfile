FROM golang:1.22.1-alpine3.19 AS build

RUN set -eux; \
    \
    apk add --no-cache git gcc make

WORKDIR /tmp/go

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN make build

FROM alpine:3.19

COPY --from=build /tmp/go/build/sni /app/sni

WORKDIR /app

ENTRYPOINT ["/app/reverse-ws-modifier"]
