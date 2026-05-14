# Stage 1: Build marrow binary
FROM golang:1.26-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o marrow ./cmd/marrow

# Stage 2: Minimal runtime
FROM alpine:3.21
RUN apk add --no-cache curl bash

# Install Engram
RUN curl -sSL https://raw.githubusercontent.com/Gentleman-Programming/engram/main/install.sh | bash

# Copy marrow binary
COPY --from=builder /app/marrow /usr/local/bin/marrow

# Create config dir
RUN mkdir -p /root/.config/marrow

# Default config
RUN echo -e "profile: default\nengram_path: engram" > /root/.config/marrow/config.yaml

ENTRYPOINT ["marrow"]
