#!/bin/bash

here=$(pwd)

# STEP 1: Add vendored libraries.
dep init
dep ensure
dep ensure -add github.com/coredns/coredns
cd ${here}/vendor/github.com/coredns/coredns
git init
# dep ensure -add github.com/alecthomas/gometalinter
# dep ensure -add github.com/mholt/caddy
# dep ensure -add github.com/miekg/dns
# dep ensure -add github.com/prometheus/client_golang/prometheus/promhttp
# dep ensure -add github.com/prometheus/client_golang/prometheus
# dep ensure -add golang.org/x/net/context
# dep ensure -add golang.org/x/text
# dep ensure -add github.com/flynn/go-shlex

# STEP 2: Comment out `go get`s in coredns Makefile.

# STEP 3: Build with `make` or `make fresh`.
