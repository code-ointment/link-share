package engine

/*
* Process a recieved helo packet.
 */
import (
	"log/slog"
	"net"
	"time"

	"github.com/code-ointment/link-share/internal/consts"
	"github.com/code-ointment/link-share/link_proto"
)

func (pe *ProtocolEngine) HeloHandler(hi *link_proto.Helo) {

	pe.mutex.Lock()

	ip := net.ParseIP(hi.Ipaddr)
	h := pe.findHost(ip)

	if h == nil {
		h = NewHost(ip)
		pe.hosts = append(pe.hosts, h)
		slog.Info("new host", "host", h.IP.String())

		// New guy on the block.  Send routes we have learned.
		pe.mutex.Unlock()

		pe.AdvertiseRoutes()
		return
		// Schedule Annoucement.
	}

	slog.Info("update host", "host", h.IP.String())
	h.State = consts.UP
	h.UpdateTime = time.Now().Unix()
	pe.mutex.Unlock()

}

/*
* Probably not a lot of hosts on the net.  Linear search for now.
 */
func (pe *ProtocolEngine) findHost(ipaddr net.IP) *Host {

	for _, h := range pe.hosts {
		if h.IP.Equal(ipaddr) {
			return h
		}
	}
	return nil
}
