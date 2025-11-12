# Stage 1: build binary
FROM golang:1.25-alpine AS builder
WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary aplikasi, nonaktifkan CGO
RUN CGO_ENABLED=0 GOOS=linux go build -o course-service ./cmd

# Stage 2: Runtime
FROM alpine:latest
WORKDIR /app

# Copy binary dari builder
COPY --from=builder /app/course-service .

# Set env port (opsional)
ENV PORT=8080

EXPOSE 8080
CMD ["./course-service"]
