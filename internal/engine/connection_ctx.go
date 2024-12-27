package engine

import (
	"net"

	"github.com/code-ointment/link-share/internal/inet"
	"golang.org/x/net/ipv6"
)

/*
* Assocaite IP Addresses, Interface, and IPv6 packet connection.
 */

type ConnectionCtx struct {
	Addrs   []net.IP
	Intf    *net.Interface
	PktConn *ipv6.PacketConn
}

func NewConnectionCtx(
	eth *net.Interface,
	addrList []net.Addr,
	pc *ipv6.PacketConn) ConnectionCtx {

	ctx := ConnectionCtx{Intf: eth,
		PktConn: pc}
	for _, ipa := range addrList {
		ip := inet.AddrToIP(ipa)
		ctx.Addrs = append(ctx.Addrs, ip)
	}
	return ctx
}

/*
* Return the first IPv4 addr in the list.
* TODO: upgrade to use something like the netip library.  ip.To4 can give
* misleading results (IPv4 in IPv6\)
 */
func (ce *ConnectionCtx) GetIPv4Addr() net.IP {

	for _, ip := range ce.Addrs {
		if ip.To4() != nil {
			return ip
		}
	}
	return nil
}

func (ce *ConnectionCtx) GetIPv6Addr() net.IP {

	for _, ip := range ce.Addrs {
		if ip.To4() == nil {
			return ip
		}
	}
	return nil
}
