# Build stage
FROM golang:1.21.4-bookworm AS builder

WORKDIR /app

# Copy over files
COPY . .

# Build the binary
RUN go build -o wolfecho

# Final stage
FROM debian:buster-slim

WORKDIR /app

# Copy only the built binary from the builder stage
COPY --from=builder /app/wolfecho .

# Volume for persistent data
VOLUME [ "/app/data" ]

# Run the binary
CMD ["./wolfecho"]
