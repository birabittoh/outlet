# syntax=docker/dockerfile:1

FROM golang:1.23-alpine AS builder

WORKDIR /build

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Transfer source code
COPY *.go ./

# Build
RUN CGO_ENABLED=0 go build -trimpath -o /dist/outlet

# Test
FROM builder AS run-test-stage
RUN go test -v ./...

FROM alpine:3 AS build-release-stage

COPY --from=builder /dist /app

WORKDIR /app
ENTRYPOINT ["/app/outlet"]
