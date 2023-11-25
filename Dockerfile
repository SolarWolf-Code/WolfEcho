# Use an official Golang runtime as a parent image
FROM golang:latest

# Set the working directory to /go/src/app
WORKDIR /go/src/app

# Copy the current directory contents into the container at /go/src/app
COPY . .

# Build the Go app
RUN go build -o app

# Define a volume for persistent storage
VOLUME ["/data"]

# Command to run the executable
CMD ["./app"]
