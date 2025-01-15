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
	defer pe.mutex.Unlock()

	ip := net.ParseIP(hi.Ipaddr)
	h := pe.findHost(ip)

	if h == nil {
		h = NewHost(ip)
		pe.hosts = append(pe.hosts, h)
		slog.Debug("new host", "host", h.IP.String())

		// New guy on the block.  Send routes we have learned.
		pe.AdvertiseRoutesUL()
		return
	}

	// Remote host is requesting an update.
	if hi.Request == link_proto.HeloRequest_INIT {
		pe.AdvertiseRoutesUL()
	}

	slog.Debug("update host", "host", h.IP.String())
	h.State = consts.UP
	h.UpdateTime = time.Now().Unix()
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

/*
* Eject hosts that we haven't heard from in 3 polling periods
 */
func (pe *ProtocolEngine) HostAccounting() {

	pe.mutex.Lock()
	defer pe.mutex.Unlock()

	now := time.Now().Unix()
	hosts := []*Host{}

	for _, h := range pe.hosts {

		delta := now - h.UpdateTime
		if delta > int64(3*consts.POLL_INTERVAL) {
			slog.Info("host timed out,removing", "host", h.IP.String())
		} else {
			hosts = append(hosts, h)
		}
	}
	pe.hosts = hosts
}
