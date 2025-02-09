package inet

/*
* Track interfaces and tunnels.
 */
import (
	"log/slog"
	"net"
	"strings"
	"sync"

	"github.com/code-ointment/link-share/internal/consts"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

type InterfaceManager struct {
	mutex      sync.Mutex
	interfaces []netlink.Link
	tunnels    []netlink.Link
}

func NewInterfaceManager() *InterfaceManager {

	ifm := InterfaceManager{}
	var err error
	var interfaces []netlink.Link

	interfaces, err = netlink.LinkList()
	if err != nil {
		slog.Error("error getting interfaces", "err", err)
		return nil
	}

	for _, l := range interfaces {
		ifm.classify(l)
	}

	return &ifm
}

/*
* Launch the link monitor thread.
 */
func (ifm *InterfaceManager) Start() {
	go ifm.linkMonitor()
}

// Replace with iterator later.
func (ifm *InterfaceManager) GetInterfaces() []netlink.Link {
	return ifm.interfaces
}

/*
* Check for interfaces we are not using for our purpose.
 */
func (ifm *InterfaceManager) classify(l netlink.Link) consts.LinkClass {

	notThese := []string{"vmnet", "docker", "vibr"}
	lattrs := l.Attrs()

	for _, n := range notThese {
		if strings.HasPrefix(lattrs.Name, n) {
			return consts.UNUSED
		}
	}

	ifm.mutex.Lock()
	defer ifm.mutex.Unlock()

	/* Interface we're advertising.*/
	if lattrs.RawFlags&unix.IFF_POINTOPOINT == unix.IFF_POINTOPOINT {
		ifm.tunnels = append(ifm.tunnels, l)
		return consts.TUNNEL
	} else {
		/* Connected local interfaces */
		if lattrs.RawFlags&unix.IFF_LOOPBACK != unix.IFF_LOOPBACK {
			ifm.interfaces = append(ifm.interfaces, l)
			return consts.STANDARD
		}
	}

	slog.Debug("unclassifed interface", "name", lattrs.Name,
		"flags", lattrs.RawFlags)
	return consts.UNUSED
}

/*
* Look for index checking both tunnels and interface list.
 */
func (ifm *InterfaceManager) GetLinkByIndex(index int) netlink.Link {

	ifm.mutex.Lock()
	defer ifm.mutex.Unlock()

	for _, lnk := range ifm.interfaces {
		if lnk.Attrs().Index == index {
			return lnk
		}
	}

	for _, lnk := range ifm.tunnels {
		if lnk.Attrs().Index == index {
			return lnk
		}
	}

	return nil
}

/*
* Test for upper and lower halfs being up.
 */
func (ifm *InterfaceManager) IsUp(l netlink.Link) bool {

	if l.Attrs().RawFlags&unix.IFF_UP == unix.IFF_UP &&
		l.Attrs().RawFlags&unix.IFF_LOWER_UP == unix.IFF_LOWER_UP {
		return true
	}
	return false
}

/*
* Listen for  kernel link updates.
 */
func (ifm *InterfaceManager) linkMonitor() {

	ch := make(chan netlink.LinkUpdate)
	done := make(chan struct{})
	defer close(done)

	err := netlink.LinkSubscribe(ch, done)
	if err != nil {
		slog.Error("error subscribing NETLINK", "err", err)
		return
	}

	for {
		update := <-ch
		attrs := update.Link.Attrs()

		l := ifm.GetLinkByIndex(attrs.Index)
		if l == nil {
			ifm.classify(update.Link)
		} else {
			st1 := ifm.IsUp(l)
			st2 := ifm.IsUp(update.Link)
			if st1 != st2 {
				slog.Info("state change", "new state", st2)
			}
			// TODO: revisit what's saved.
			l.Attrs().RawFlags = update.Link.Attrs().RawFlags
		}
	}
}

func (ifm *InterfaceManager) GetTunnelByIndex(linkIndex int) netlink.Link {

	ifm.mutex.Lock()
	defer ifm.mutex.Unlock()

	for _, tun := range ifm.tunnels {
		if tun.Attrs().Index == linkIndex {
			return tun
		}
	}
	return nil
}

/*
* Have we detected any tunnels?
 */
func (ifm *InterfaceManager) HasTunnels() bool {
	ifm.mutex.Lock()
	defer ifm.mutex.Unlock()
	v := len(ifm.tunnels) != 0
	return v
}

/*
* Return a copy of our current tunnel set
 */
func (ifm *InterfaceManager) GetTunnels() []netlink.Link {
	ifm.mutex.Lock()
	defer ifm.mutex.Unlock()

	t := []netlink.Link{}
	t = append(t, ifm.tunnels...)
	return t
}

func (ifm *InterfaceManager) GetDefaultLink() netlink.Link {

	_, g, _ := net.ParseCIDR("8.8.8.8/32")

	routes, err := netlink.RouteGet(g.IP)
	if err != nil {
		slog.Warn("get default link failed", "error", err)
		return nil
	}

	if len(routes) > 0 {
		for _, r := range routes {
			l := ifm.GetLinkByIndex(r.LinkIndex)
			return l
		}
	}
	return nil
}
