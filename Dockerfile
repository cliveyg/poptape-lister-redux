# Build stage
FROM golang:1.21-alpine AS builder

# Add bash etc as alpine version doesn't have these
RUN apk add --no-cache bash git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o lister .

# Final stage
FROM alpine:latest

# Add bash and other utilities
RUN apk --no-cache add bash ca-certificates tzdata

# Set working directory
WORKDIR /root/

# Copy binary from builder stage
COPY --from=builder /app/lister .

# Create log directory
RUN mkdir -p /root/log

# Make port 8400 available to the world outside this container
EXPOSE 8400

# Run the binary
CMD ["./lister"]