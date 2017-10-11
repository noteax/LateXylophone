# Decsription
My implmentation of load balancer uses round-robin algorithm, which means that requests distributed across servers one 
by one, ie request1 goes to server1, request2 goes to server3 and so one. That leads to unfair work distribution and 
doesn't consider that one server could be more efficient than another one. As result, backends wouldn't be loaded with 
requests optimally. One of the ways to improve that - use round-robin algorithm with weights.

Next decision which I've made was about health-checking. My solution does passive health checking, which means, that 
server status will be updated to "inactive" after request failure (or timeout) and will stay inactive for some period.
There is also ways to improve that - for example, separate call in background could be used check is server alive or 
not.

Another one thing to do was failover. My algorithm just tries all active servers one by one and fails if all servers are
tried and there is no other active servers at the moment. In current CLI that could lead to timeouts when inactive 
server became active (because we haven't method to make server alive after `kill`). That also could be improved by using
active health-checking and graceful starts of failed servers - when started server serves only small amount of requests.
