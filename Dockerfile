# BUILD
FROM golang:1.20 AS build-stage

WORKDIR /app

# Setup env
COPY go.mod go.sum ./
RUN go mod download


# Setup source code
COPY HokTseBunBot/  ./
RUN go build -o /HokTseBunBot
ENTRYPOINT ["/HokTseBunBot"]