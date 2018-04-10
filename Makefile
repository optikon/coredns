# Makefile for Optikon DNS.

IMAGE ?= dockerhub.cisco.com/intelligent-edge-dev-docker-local/optikon-dns
TAG ?= 1.0.0

# Build the custom CoreDNS Docker image.
.PHONY: all
all:
	docker build -t $(IMAGE):$(TAG) .
	docker rmi -f $$(docker images -q -f dangling=true)

# Removes all object and executable files.
.PHONY: clean
clean:
	docker image rm -f $(IMAGE):$(TAG)

# Removes and rebuilds everything.
.PHONY: fresh
fresh: clean all
