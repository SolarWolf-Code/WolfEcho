FROM golang:1.21.4-bookworm

# Path: /app
WORKDIR /app

# Copy over files
COPY . .

# build the binary
RUN go build -o wolfecho .

# run wolfecho
CMD ["./wolfecho"]