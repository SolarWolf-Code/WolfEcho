# Use an official Golang runtime as a parent image
FROM golang:latest as builder

# Set the working directory to /app
WORKDIR /app

# Copy the current directory contents into the container at /app
COPY . .

RUN go build -o main .

# Create a directory to store the database file
RUN mkdir /data

CMD [ "./main" ]