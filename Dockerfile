FROM golang:1.21-bullseye

# Set destination for COPY
WORKDIR /app

ENV CGO_ENABLED=0

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

COPY assets ./assets
COPY cmd ./cmd
COPY internal ./internal
COPY assets ./assets

RUN mkdir tmp && chmod 777 tmp

RUN apt-get update
RUN apt-get upgrade -y
RUN DEBIAN_FRONTEND=noninteractive apt-get install -y build-essential libssl-dev ffmpeg

# Build
RUN GOOS=linux go build ./cmd/api/main.go

EXPOSE 80

# Run
CMD ["/app/main"]