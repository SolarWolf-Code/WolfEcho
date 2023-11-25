FROM golang:1.21.4-bookworm

WORKDIR /app

# Copy over files
COPY . ./

# build the binary
RUN go build -o wolfecho/wolfecho .

RUN ls
RUN pwd

# run wolfecho
CMD ["wolfecho/wolfecho"]