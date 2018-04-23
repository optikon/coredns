# optikon-central

## Name

*optikon-central* - manages Kubernetes service to edge cluster IP mappings.

## Description

[FINISH]

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
    kubernetes cluster.local in-addr.arpa ip6.arpa {
       pods insecure
       upstream
       fallthrough in-addr.arpa ip6.arpa
    }
    optikon-central
}
~~~
