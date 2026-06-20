# Stage 1: Build the Go monolith binary
FROM golang:alpine AS builder

WORKDIR /app

# Download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build monolith binary statically
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o monolith cmd/monolith/*.go

# Stage 2: Minimal runtime image
FROM alpine:latest

WORKDIR /app

# Copy statically linked binary
COPY --from=builder /app/monolith /app/monolith

# Expose HTTP ports for modular monolith listeners
# Ingest: 8081, Marketing: 8082, Analytics: 8083, Mock CRM: 8084
EXPOSE 8081 8082 8083 8084

# Default environment configurations
ENV DATABASE_TYPE=memory
ENV BROKER_TYPE=memory

ENTRYPOINT ["/app/monolith"]
