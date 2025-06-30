FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o media-server .

FROM alpine:latest

RUN apk add --no-cache tzdata ffmpeg && \
    cp /usr/share/zoneinfo/UTC /etc/localtime && \
    echo "UTC" > /etc/timezone

COPY --from=builder /app/media-server /usr/local/bin/media-server

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/media-server"]
