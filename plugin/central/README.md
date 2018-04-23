# optikon-central

## Name

*optikon-central* - manages Kubernetes service to edge cluster IP mappings.

## Description

This plugin is responsible for managing a table that maps external service domain names to the list of edge sites running that service. When proxied requests come in from edge sites, this plugin will return the list of services running the request service as a DNS message.

## Syntax

~~~ txt
optikon-central
~~~

## Examples

An example Corefile might look like

~~~ corefile
.:53 {
    errors
    health
    log
    kubernetes cluster.local {
       fallthrough
    }
    optikon-central
    proxy . 8.8.8.8:53
}
~~~
