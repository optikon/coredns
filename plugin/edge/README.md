# optikon-edge

## Name

*optikon-edge* - resolves client requests to the nearest edge cluster running the requested service.

## Description

[FINISH]

## Syntax

~~~ txt
optikon-edge
~~~

## Examples

The optikon-edge Corefile entry requires three arguments

~~~ corefile
. {
    optikon-edge [CENTRAL CLUSTER IP] [MY LONGITUDE] [MY LATITUDE]
}
~~~

An example Corefile might look like

~~~ corefile
.:53 {
    errors
    health
    kubernetes cluster.local in-addr.arpa ip6.arpa {
       pods insecure
       upstream
       fallthrough in-addr.arpa ip6.arpa
    }
    prometheus :9153
    proxy . /etc/resolv.conf
    cache 3600
    optikon-edge 172.16.7.101 55.680770 12.543006
}
~~~
