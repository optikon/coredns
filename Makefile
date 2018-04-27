# Makefile for Optikon DNS.

IMAGE ?= intelligentedgeadmin/optikon-dns
<<<<<<< HEAD
TAG ?= 2.0.0
=======
TAG ?= 1.0.0
>>>>>>> 5c6e46abd7f74e7ad7f985a7c18ecaeff134f99a

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
