# Stage 1: Build the Go executable
FROM golang:1.20.3-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Install GCC
RUN apk add --no-cache gcc musl-dev

# Copy go.mod and go.sum files to download dependencies
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code to the container
COPY . .


# Build the Go executable
RUN CGO_ENABLED=1 GOOS=linux go build -a -o cross-nostter ./cmd/cross-nostter

# Stage 2: Create a minimal runtime image
FROM alpine:latest

# Set the working directory inside the container
WORKDIR /app

# Copy the Go executable from the builder stage
COPY --from=builder /app/cross-nostter .

# Make the Go executable executable
RUN chmod +x cross-nostter

# Set the entry point for the container
CMD ["./cross-nostter"]
