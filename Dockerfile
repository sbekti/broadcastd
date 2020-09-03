FROM golang:alpine AS build

# Set necessary environmet variables needed for our image
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

RUN apk update && \
    apk add --no-cache \
    git

# Move to working directory /build
WORKDIR /build

# Copy and download dependency using go mod
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy the code into the container
COPY . .

# Build the application
RUN go build -o main .

# Move to /dist directory as the place for resulting binary folder
WORKDIR /dist

# Copy binary from build to main folder
RUN cp /build/main .

# Base image with ffmpeg
FROM jrottenberg/ffmpeg:4.3.1-alpine38

# Copy the main binary
COPY --from=build /dist/main /broadcastd
COPY --from=build /build/public ./public

# Command to run
ENTRYPOINT ["/broadcastd"]