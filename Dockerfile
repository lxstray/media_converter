FROM alpine:latest

RUN apk add --no-cache \
    ffmpeg \
    yt-dlp \
    go \
    git \
    bash

WORKDIR /app

COPY . .

CMD ["go", "run", "cmd/main.go"]
