# optikon-edge

## Name

*optikon-edge* - resolves client requests to the nearest edge cluster running the requested service.

## Description

This plugin is responsible for resolving incoming client requests by either returning it own IP if the service is already running on this cluster; otherwise it forwards the request up to its (possibly many) central proxies, received a list of all the clusters running the requested service, performs a proximity calculation, and returns the IP of the closest edge cluster serving the request service.

This plugin also perform a routine daemon process that calls the Kubernetes cluster API to fetch its running services, and pushes that list up to the central proxies so they can update their global tables.

## Syntax

~~~ txt
optikon-edge ${MY_IP} ${LON} ${LAT} ${SVC_READ_INTERVAL} ${SVC_PUSH_INTERVAL} . ${CENTRAL_IP}:53
~~~

## Examples

An example Corefile might look like

~~~ corefile
.:53 {
    errors
    health
    log
    cache 30
    kubernetes cluster.local {
       fallthrough
    }
    optikon-edge 172.16.7.102 43.264 36.694 2 4 . 172.16.7.101:53
    proxy . 8.8.8.8:53
}
~~~
