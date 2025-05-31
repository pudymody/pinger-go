# Use Go 1.24 alpine as base image
FROM docker.io/library/golang:1.24-alpine AS build

RUN apk add --no-cache build-base 

# Move to working directory /build
WORKDIR /build

# Copy the go.mod and go.sum files to the /build directory
COPY go.mod go.sum ./

# Install dependencies
RUN go mod download

# Copy the entire source code into the container
COPY . .

# Build the application
RUN CGO_ENABLED=1 go build -ldflags "-s -w -extldflags '-static'" -o ./pinger-go

FROM docker.io/library/alpine:latest
RUN apk add --no-cache tzdata

COPY --from=build /build/pinger-go /pinger-go

# Document the port that may need to be published
EXPOSE 8000

# Start the application
ENTRYPOINT ["/pinger-go"]
