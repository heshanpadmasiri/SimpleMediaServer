# Build stage
FROM golang:1.22.6-alpine AS builder

# Install git for Go modules that might need it
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o media-server .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests if needed
RUN apk --no-cache add ca-certificates

# Create a non-root user
RUN addgroup -g 1001 -S mediaserver && \
    adduser -u 1001 -S mediaserver -G mediaserver

# Create necessary directories
RUN mkdir -p /app/static /app/templates /config

# Set working directory
WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/media-server .

# Copy static assets and templates
COPY --from=builder /app/static ./static
COPY --from=builder /app/templates ./templates

# Copy container config
COPY --from=builder /app/config.container.toml ./config.toml

# Change ownership of app directory
RUN chown -R mediaserver:mediaserver /app

# Switch to non-root user
USER mediaserver

# Expose port 8080
EXPOSE 8080

# Command to run the application
CMD ["./media-server"]