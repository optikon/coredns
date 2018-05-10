# Makefile for Optikon DNS.

IMAGE ?= optikon/coredns
TAG ?= local

# Build the custom CoreDNS Docker image.
.PHONY: container
container:
	docker build -t $(IMAGE):$(TAG) .

# Removes all object and executable files.
.PHONY: clean
clean:
	docker image rm -f $(IMAGE):$(TAG)

# Removes and rebuilds everything.
.PHONY: fresh
fresh: clean container
