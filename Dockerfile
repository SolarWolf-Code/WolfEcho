FROM golang:1.21.4-bookworm

WORKDIR /app

# Copy over files
COPY . .

# build the binary
RUN go build -o /app/wolfecho .

# run wolfecho
CMD ["./wolfecho"]