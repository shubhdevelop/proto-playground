# Use Debian bookworm instead of Alpine to provide glibc support for valkey-glide
FROM golang:1.24-bookworm AS builder

WORKDIR /app

# Install build dependencies
RUN apt-get update && apt-get install -y gcc libc6-dev

# Copy dependency files and download
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o main .

# Stage 2: Runtime (using debian slim for a smaller final image)
FROM debian:bookworm-slim
WORKDIR /root/
COPY --from=builder /app/main .

EXPOSE 8080
CMD ["./main"]
