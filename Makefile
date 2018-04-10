# Makefile for Optikon DNS.

IMAGE ?= dockerhub.cisco.com/intelligent-edge-dev-docker-local/optikon-dns
TAG ?= 1.0.0

# Build the custom CoreDNS Docker image.
all:
	docker build -t $(IMAGE):$(TAG) .

# Removes all object and executable files.
clean:
	docker image rm -f $(IMAGE):$(TAG)

# Removes and rebuilds everything.
fresh: clean all

# Specifies which rule targets don't actually refer to filenames, but
# are just commands instead.
.PHONY: all clean fresh
