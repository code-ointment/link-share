#!/bin/bash
#
# Configure linux host as a router.  Notably, VPN into Riverbed, configure host as 
# router, and add the 'router' ip address as a static route.
# EX:
# Host 192.168.1.98 is VPN'd in and configured as a router 
#    ip route add 10.0.0.0/8 via 192.168.1.98
#
sysctl net.ipv4.ip_forward=1
sysctl net.ipv6.conf.all.forwarding=1

systectl list-unit-files firewalld.service > /dev/null 2>/dev/null
if [ $? -eq 0 ]; then
    systemctl stop firewalld
else
    systemctl stop ufw
fi

nft flush ruleset
nft add table inet filter
nft add chain inet filter input '{type filter hook input priority 0; policy accept; }'
nft add chain inet filter forward '{type filter hook forward priority 0; policy accept; }'
nft add chain inet filter output '{type filter hook output priority 0; policy accept; }'

#nft add chain inet filter forward '{type filter hook forward priority 0; policy accept; }'

nft add table inet nat
nft add chain inet nat prerouting '{ type nat hook prerouting priority -100; }'
nft add chain inet nat postrouting '{ type nat hook postrouting priority 100; }'
#nft add rule inet nat postrouting masquerade
nft add rule inet nat postrouting oifname "ens33" masquerade
nft add rule inet nat postrouting iifname "ens33" masquerade



