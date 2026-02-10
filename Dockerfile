# Build Stage
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o main cmd/main.go

# Final Stage
FROM debian:bookworm-slim

# Install system dependencies
RUN apt-get update && apt-get install -y \
    ffmpeg \
    libreoffice \
    poppler-utils \
    fonts-noto-cjk \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy binary and templates from builder
COPY --from=builder /app/main .
COPY --from=builder /app/templates ./templates
# Create uploads directory
RUN mkdir -p uploads

# Configuration for LibreOffice (to run in headless mode)
ENV HOME=/tmp

EXPOSE 8080

CMD ["./main"]
