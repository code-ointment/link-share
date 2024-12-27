# link-share
## Sharing VPN links on a home network for Fun and Profit.
The objective is to share a single VPN link with multiple linux hosts.

One host will manage the VPN link and will annouce link availability.  Other
hosts will use the announcement to create a route through the VPN link host.

As this is a home project, there are a number of experiments embedded.  

1. Using Multicast IPv6 on local links as transport.
2. Using google protobuf to form protocol packets.

To build:

    make generate
    make

