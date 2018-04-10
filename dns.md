### optikon DNS

Load balancing traffic at the edge is essential - it's the point of deploying apps to the edge. Incoming requests should route to the geographically closest cluster to the requesting device.

Here is a general idea of how that could work:

![dnsplan](https://wwwin-github.cisco.com/edge/optikon/blob/master/docs/dnsplan.png)


To walk through this diagram:
1. Say that an OTT video company offers a website called lowlatency.com, or **ll.com**. Say this hostname should resolve to a Super Bowl livestream. The video company has deployed their edge video cache to several edge clusters - for simplicity, say there are 3 edge clusters all in the same metro area. All of them can serve the video at `ll.com`.
2. The requesting device is a phone. The user types in `ll.com` into their browser to watch the video.
3. This browser has anycast enabled and `ll.com` initially resolves to the topographically closest DNS server. *whether or not this edge cluster is running the ll.com cache*, this edge CoreDNS server doesn't know how to resolve `ll.com` yet. All the Edge CoreDNS knows, is that it's been preloaded with the geo coordinates of itself and all the other edge clusters. (This is probably a CoreDNS plugin / small datastore on the edge cluster.)
4. So it calls "home" to a central CoreDNS server that it's been preconfigured to talk to. Asking how to resolve `ll.com`.
5. Central CoreDNS has been preloaded with a mapping of `ll.com` --> `<list of IPs>`. This list of IPs is all the instances of the `ll.com` cache. All these instances live at different edge clusters.
6. CentralCoreDNS simply responds to the Edge CoreDNS forward with that list of IPs.
7. EdgeCoreDNS gets back that list of IPs. If "itself" (a cache running on its own cluster) *is* on the list, it says "great" and opens a connection with the device, serving the super bowl video to the phone from a cache running on its own cluster.
8. But if "itself" is *not* on that list of IPs (aka "i'm not running an `ll.com` cache on my cluster"), the edgeCoreDNS routes the request to the **geographically closest** cluster. It knows who is closest to itself because it's been preloaded with geo information.
9. And from there, the nearest edge cluster says "okay" and opens a connection to the requesting device.


Concerns and open issues:
- Edge CoreDNS obviously can't *just* be coreDNS. there will have to be a small plugin / additional server that is running and storing all this geo information about the clusters.
- Is there a faster way for step 7 to happen? edge coreDNS gets a list of where all the edge caches live. how does it know if "itself" is or is not on the list?
- Latency re: forwarding. There could be 3-4 forwards/proxies for the `ll.com` request, from device to actually serving the video.
