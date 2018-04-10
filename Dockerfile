# Start with a Golang-enabled base image.
FROM golang:1.10.0 as builder

# Fetch the CoreDNS repo.
RUN go get github.com/coredns/coredns
WORKDIR /go/src/github.com/coredns

# Mount the central and edge plugins.
COPY plugin/central plugin/central
COPY plugin/edge plugin/edge

# Mount the custom plugin.cfg file.
COPY plugin/plugin.cfg plugin.cfg

# Build the custom CoreDNS binary.
RUN make

# Build a runtime container to use the custom binary.
FROM alpine:latest
MAINTAINER Ross Flieger-Allison

# Only need ca-certificates & openssl if want to use DNS over TLS (RFC 7858).
RUN apk --no-cache add bind-tools ca-certificates openssl && update-ca-certificates

# Copy the custom binary from the build container.
COPY --from=builder /go/bin/coredns /coredns

# Expose DNS ports.
EXPOSE 53 53/udp

# Mount the executable for entry.
ENTRYPOINT ["/coredns"]
