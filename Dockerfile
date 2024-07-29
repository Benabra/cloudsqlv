# Stage 1: Build the Go app
FROM golang:1.22.5-alpine AS builder

# Install git (required for fetching dependencies)
RUN apk add --no-cache git

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go.mod and go.sum to the working directory
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the application code
COPY . .

# Build the Go application
RUN go build -o main .

# Stage 2: Create a minimal runtime image with Google Cloud SDK
FROM google/cloud-sdk:latest

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the built Go application from the builder stage
COPY --from=builder /app/main .

# Run the Go application and then copy CSV files to Google Cloud Storage
CMD ["sh", "-c", "./main -output csv -limit 10 && gsutil cp /app/*.csv gs://lv-gcs-prd-cloudsql-export/"]
