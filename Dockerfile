FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o geocoder cmd/server/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata curl bash make

WORKDIR /root/

# Copy the binary
COPY --from=builder /app/geocoder .

# Copy migrations
COPY --from=builder /app/migrations ./migrations

# Copy web templates
COPY --from=builder /app/web ./web

# Expose port
EXPOSE 8080

# Health check removed - handled by Coolify

# Run the application
CMD ["./geocoder"]
