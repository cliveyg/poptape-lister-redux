# Build stage
FROM golang:1.23-alpine AS build

RUN mkdir /app
ADD . /app
WORKDIR /app

# Copy go mod files
#COPY go.mod go.sum ./

# Download dependencies
#RUN go mod download

# Copy source code
#COPY . .
COPY .env .

RUN rm -f go.mod go.sum
RUN go mod init github.com/cliveyg/poptape-lister-redux
RUN go mod tidy

RUN go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build -a -ldflags '-w' -o lister

# Final stage - use busybox for smaller image and better compatibility
FROM busybox:latest
#FROM alpine:latest

# Set working directory
RUN mkdir -p /lister
COPY --from=build /app/lister /lister
COPY --from=build /app/.env /lister
WORKDIR /lister

# Create log directory
RUN mkdir -p /lister/log

# Make port 8400 available to the world outside this container
EXPOSE $PORT

# Run the binary
CMD ["./lister"]