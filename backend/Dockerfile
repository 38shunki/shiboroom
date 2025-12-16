FROM golang:1.23-alpine

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy source code first
COPY . .

# Initialize dependencies
RUN go mod tidy

# Download dependencies
RUN go mod download

# Build the application
RUN go build -o main ./cmd/api

EXPOSE 8084

CMD ["./main"]
