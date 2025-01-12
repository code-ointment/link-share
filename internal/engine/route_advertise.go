package engine

/*
* Wait for the route manager to send an update. Fetch the updates form the
* route manager and send on our multicast channel.
 */
import (
	"log/slog"
	"net"
	"os"

	"github.com/code-ointment/link-share/internal/consts"
	"github.com/code-ointment/link-share/internal/inet"
	"github.com/code-ointment/link-share/link_proto"
	"google.golang.org/protobuf/proto"
)

/*
* Go routine that advertises route updates.
 */
func (pe *ProtocolEngine) AdvertiseUpdates() {

	// Advertise routes the router manager found on initialization.
	if pe.rm.LearnedCount() > 0 {
		pe.AdvertiseRoutes()
	}

	// Wait for an update and advertise.
	for {
		pe.rm.WaitForUpdate()
		pe.AdvertiseRoutes()
	}
}

/*
*Advertise all route advertisements
 */
func (pe *ProtocolEngine) AdvertiseRoutes() {

	rts := pe.rm.GetRouteUpdates()
	pe.dnsConfig.ReadConfig()

	slog.Debug("dns config", "nameserver", pe.dnsConfig.GetNameServers(),
		"domains", pe.dnsConfig.GetDomains())
	for _, rt := range rts {
		pe.SendAdvertisement(&rt)
	}
}

/*
* Send advertisement.  Destination fetched from route update  - presumably
* the VPN link.  Gateway is the local host.
 */
func (pe *ProtocolEngine) SendAdvertisement(rt *inet.RouteUpdate) {

	mgroup := net.ParseIP(consts.GroupAddr)
	dst := &net.UDPAddr{IP: mgroup, Port: consts.ListenPort}

	pe.mutex.Lock()
	defer pe.mutex.Unlock()

	for _, c := range pe.connections {

		var me net.IP

		// Advertise IPv4 routes using an IPv4 gateway address
		if len(rt.Dst.IP) == int(net.IPv4len) {
			me = c.GetIPv4Addr()
		} else {
			me = c.GetIPv6Addr()
		}
		slog.Info("advertise", "me", me, "op", rt.Op, "dst", inet.IPNetToCidr(&rt.Dst))

		route := link_proto.Route{
			Op:   int32(rt.Op),
			Dest: inet.IPNetToCidr(&rt.Dst),
		}
		announce := link_proto.Announce{
			Lstate:        link_proto.LinkState_UP,
			Gateway:       me.String(),
			Domain:        pe.domain,
			Nameservers:   pe.dnsConfig.GetNameServers(),
			Searchdomains: pe.dnsConfig.GetDomains(),
			Routes:        []*link_proto.Route{&route},
		}

		pph := link_proto.Packet_Announce{Announce: &announce}
		pkt := link_proto.Packet{
			Pkttype: &pph,
		}

		out, err := proto.Marshal(&pkt)
		if err != nil {
			slog.Error("Failed marshaling", "error", err)
			os.Exit(1)
		}

		_, err = c.PktConn.WriteTo(out, nil, dst)
		if err != nil {
			slog.Error("failed writing", "error", err)
		}
	}
}
