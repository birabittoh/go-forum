# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS builder

WORKDIR /build

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Transfer source code
COPY *.go ./
COPY internal ./internal

# Build
RUN CGO_ENABLED=0 go build -o /dist/go-forum

# Test
FROM build-stage AS run-test-stage
RUN go test -v ./...

FROM scratch AS build-release-stage

WORKDIR /app

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY templates ./templates
COPY static ./static
COPY xtitles ./xtitles
COPY --from=builder /dist .

ENTRYPOINT ["./go-forum"]
