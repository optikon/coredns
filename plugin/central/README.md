# optikon-central

## Name

*optikon-central* - manages Kubernetes service to edge cluster IP mappings.

## Description

This plugin is responsible for managing a table that maps external service domain names to the list of edge sites running that service. When proxied requests come in from edge sites, this plugin will return the list of services running the request service as a DNS message.

This plugin also runs a daemon process to listen for incoming edge cluster POST requests, containing the list of Kubernetes services they are currently running. This information is then used to update the internal Table used to maintain the global state across all clusters.

## Syntax

~~~ txt
optikon-central ${MY_IP} ${LON} ${LAT} ${SVC_READ_INTERVAL}
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
    optikon-central 172.16.7.101 55.643 64.264 3
    proxy . 8.8.8.8:53
}
~~~
