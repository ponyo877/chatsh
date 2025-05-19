FROM golang:1.24-alpine3.21 AS builder
WORKDIR /app

ENV LITESTREAM_VERSION=v0.3.13

ADD https://github.com/benbjohnson/litestream/releases/download/$LITESTREAM_VERSION/litestream-$LITESTREAM_VERSION-linux-amd64.tar.gz /tmp/litestream.tar.gz
RUN tar -C /usr/local/bin -xzf /tmp/litestream.tar.gz

ENV CGO_ENABLED=1
RUN apk add --no-cache gcc musl-dev
COPY . ./
RUN go mod download
RUN go build -o chatsh server/main.go


FROM alpine:3.21
WORKDIR /app

COPY --from=builder /app/chatsh ./
COPY --from=builder /usr/local/bin/litestream /usr/local/bin/litestream

COPY litestream.yml /etc/litestream.yml
COPY schema/chatsh.sql ./schema/chatsh.sql

COPY run.sh ./
RUN chmod +x run.sh

CMD ["/app/run.sh"]