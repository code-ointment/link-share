package engine

import (
	"log/slog"
	"net"
	"os"
	"sync"

	"github.com/code-ointment/link-share/internal/consts"
	"github.com/code-ointment/link-share/internal/inet"
	"github.com/code-ointment/link-share/link_proto"
	"golang.org/x/net/ipv6"
	"google.golang.org/protobuf/proto"
)

type ProtocolEngine struct {
	ifm         *inet.InterfaceManager
	rm          *inet.RouteManager
	connections []ConnectionCtx
	mutex       sync.Mutex
	localAddrs  []net.Addr
	domain      string
	hosts       []*Host
}

func NewProtocolEngine() *ProtocolEngine {

	pe := ProtocolEngine{}
	pe.ifm = inet.NewInterfaceManager()
	pe.rm = inet.NewRouteManager(pe.ifm)

	pe.domain = "placeholder"

	return &pe
}

/*
* Set up multicast and start all of the listeners.
 */
func (pe *ProtocolEngine) Start() {
	pe.setupMulticast()
	pe.listen()

	go pe.AdvertiseUpdates()
}

/*
* Allocating socket which is used to set up packet connections on a per
* interface basis.
 */
func (pe *ProtocolEngine) setupMulticast() {

	var err error
	// Get Netlinks idea of an interface
	interfaces := pe.ifm.GetInterfaces()

	listener, err := net.ListenPacket("udp6", consts.ListenAddr)
	if err != nil {
		slog.Error("listen packet failed", "addr", consts.ListenAddr, "error", err)
		os.Exit(0)
	}

	for _, intf := range interfaces {

		// Converting netlink link to net.Interface
		eth, err := net.InterfaceByName(intf.Attrs().Name)
		if err != nil {
			slog.Error("interface lookup failed",
				"name", intf.Attrs().Name, "error", err)
			os.Exit(0)
		}

		slog.Debug("setup multicast", "interface", eth.Name)
		group := net.ParseIP(consts.GroupAddr)
		packetConnection := ipv6.NewPacketConn(listener)
		if err := packetConnection.JoinGroup(eth, &net.UDPAddr{IP: group}); err != nil {
			slog.Error("Failed joining group", "addr", consts.GroupAddr, "error", err)
			os.Exit(1)
		}

		alist, err := eth.Addrs()
		if err != nil {
			slog.Error("error getting int addresses", "error", err)
			continue
		}

		pe.mutex.Lock()
		entry := NewConnectionCtx(eth, alist, packetConnection)
		pe.connections = append(pe.connections, entry)
		pe.localAddrs = append(pe.localAddrs, alist...)
		pe.mutex.Unlock()
	}
}

/*
* Launch service threads.  1 thread per interface for now.
 */
func (pe *ProtocolEngine) listen() {

	for _, c := range pe.connections {
		go pe.listenOnConnection(c)
	}
}

/*
* Read from the interface and dispatch request.
 */
func (pe *ProtocolEngine) listenOnConnection(entry ConnectionCtx) {

	mgroup := net.ParseIP(consts.GroupAddr)
	buffer := make([]byte, consts.MaxDatagramSize)

	for {

		n, cm, addr, err := entry.PktConn.ReadFrom(buffer)
		if err != nil {
			slog.Error("readfrom failed", "error", err)
			continue
		}

		if cm != nil && cm.Dst != nil && !cm.Dst.Equal(mgroup) {
			slog.Warn("errant packet")
			continue
		}

		if pe.IsLocalAddr(addr) {
			//slog.Debug("my packet, dropping")
			continue
		}

		slog.Debug("recv", "addr", addr.String(), "bytes", n)
		packet := link_proto.Packet{}
		if err = proto.Unmarshal(buffer[:n], &packet); err != nil {
			slog.Error("Failed unmarshalling", "error", err)
		}

		switch pp := packet.Pkttype.(type) {

		case *link_proto.Packet_Helo:
			pe.HeloHandler(pp.Helo)

		case *link_proto.Packet_Announce:
			pe.AnnounceHandler(pp.Announce)
		}
	}
}

/*
* Is this address assigned to one of my interfaces?
 */
func (pe *ProtocolEngine) IsLocalAddr(addr net.Addr) bool {

	pe.mutex.Lock()
	defer pe.mutex.Unlock()

	left := inet.AddrToIP(addr)

	for _, a := range pe.localAddrs {

		right := inet.AddrToIP(a)
		//slog.Debug("address", "left", left, "right", right)
		if left.Equal(right) {
			return true
		}
	}
	return false
}

/*
* Send a helo on all interfaces.
 */
func (pe *ProtocolEngine) SendHelo() {

	mgroup := net.ParseIP(consts.GroupAddr)
	dst := &net.UDPAddr{IP: mgroup, Port: consts.ListenPort}

	pe.mutex.Lock()
	for _, c := range pe.connections {

		myAddr := c.GetIPv6Addr()
		if myAddr == nil {
			slog.Debug("no IPv6 Address available, trying IPv4")
			myAddr = c.GetIPv4Addr()
		}

		helo := link_proto.Helo{
			Ipaddr: myAddr.String(),
			Domain: pe.domain,
		}
		pph := link_proto.Packet_Helo{Helo: &helo}
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
	pe.mutex.Unlock()
}
