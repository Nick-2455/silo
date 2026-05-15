# Stage 1: Build silo binary
FROM golang:1.26-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o silo ./cmd/silo

# Stage 2: Minimal runtime
FROM alpine:3.21
RUN apk add --no-cache curl bash

# Install Engram
RUN ENGRAM_VERSION=$(curl -sL https://api.github.com/repos/Gentleman-Programming/engram/releases/latest | grep '"tag_name"' | sed 's/.*"v\(.*\)".*/\1/') && \
    curl -sSL "https://github.com/Gentleman-Programming/engram/releases/download/v${ENGRAM_VERSION}/engram_${ENGRAM_VERSION}_linux_amd64.tar.gz" | tar xz -C /usr/local/bin engram

# Copy silo binary
COPY --from=builder /app/silo /usr/local/bin/silo

# Create config dir
RUN mkdir -p /root/.config/silo

# Default config
RUN echo -e "profile: default\nengram_path: engram" > /root/.config/silo/config.yaml

ENTRYPOINT ["silo"]