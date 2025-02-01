# link-share
## Sharing VPN links on a home network for Fun and Profit.
The objective is to share a single VPN link with multiple linux hosts.

A gateway host manages a VPN link and annouces link availability.  
Other hosts will use the announcement to create a route through the gateway host.

The gateway host's DNS configuration is advertised as well  Listeners update
their DNS to follow the gateway DNS configuration.

The specific use case is that certain VPN vendor's Linux offering does
not work partiuclarly well on most Linux's.  Find a Linux verion where 
the VPN does work and share its link with more up to date Linux's.

As this is a home project, there are a number of experiments embedded.  

1. Using Multicast IPv6 on local links as transport.
2. Using google protobuf to form protocol packets.


Compiling

    make depends
    make

Packaging

    # if you have not done this yet.
    make depends
    # build a rpm in build-output directory
    VERSION=<x.x.x> BUILD_NUMBER=<y> make rpm
    # build a deb in build-output directory
    VERSION=<x.x.x> BUILD_NUMBER=<y> make deb

TODO
- No integration with firewalls, nftables rules over written.
- Infer tunnel device to track or user configures?
- Add unit tests
- Test on more VPNs

