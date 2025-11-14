# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install ca-certificates for go mod download
RUN apk add --no-cache ca-certificates git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application (static binary)
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags="-w -s" -o jazz_standards_db .

# Runtime stage - use scratch for smallest image
FROM scratch

WORKDIR /app

# Copy the static binary
COPY --from=builder /app/jazz_standards_db .

# Copy static files
COPY --from=builder /app/static ./static

# Expose the port
EXPOSE 8000

# Command to run the application
CMD ["./jazz_standards_db"]