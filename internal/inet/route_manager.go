package inet

/*
* learnedUpdates are route updates caused by the VPN on the gateway host.
* learnedUpdates will typically be on the gateway host.
*
* selfRoutes are routes added by the route manager.  This sort route should
* exist on the 'client' hosts
 */
import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

type RouteManager struct {
	ifm            *InterfaceManager
	learnedUpdates []RouteUpdate   // Routes we learned from the kernel
	selfRoutes     []netlink.Route // Routes the manager was asked to add

	mutex sync.Mutex

	updateCount     int
	updateMutex     sync.Mutex
	updateCondition *sync.Cond

	routingEnabled int
	nfu            *NftUtil
	def6Net        *net.IPNet // Handy constants
	def4Net        *net.IPNet
}

type RouteUpdate struct {
	Op  uint16
	Dst net.IPNet
}

func NewRouteManager(manager *InterfaceManager) *RouteManager {

	rm := RouteManager{
		ifm: manager,
	}

	l := rm.GetDefaultLink()
	rm.nfu = NewNftUtil(l.Attrs().Name)

	// handy defaults
	_, rm.def6Net, _ = net.ParseCIDR("::/0")
	_, rm.def4Net, _ = net.ParseCIDR("0.0.0.0/0")

	go rm.routeMonitor()

	rm.updateCondition = sync.NewCond(&rm.updateMutex)

	rm.initLearnedUpdates()
	return &rm

}

/*
* Turn host routing on and off
 */
func (rm *RouteManager) EnableRouting() {

	if rm.routingEnabled == 1 {
		return
	}

	rm.routingEnabled = 1
	rm.setRouting(rm.routingEnabled)
	rm.nfu.EnableForwarding()
}

func (rm *RouteManager) DisableRouting() {

	if rm.routingEnabled == 0 {
		return
	}

	rm.routingEnabled = 0
	rm.setRouting(rm.routingEnabled)
	rm.nfu.DisableForwarding()
}

func (rm *RouteManager) setRouting(onoff int) {

	v := strconv.Itoa(onoff)

	ctl := []string{
		"/proc/sys/net/ipv4/ip_forward",
		"/proc/sys/net/ipv6/conf/all/forwarding"}

	for _, fname := range ctl {
		fd, err := os.OpenFile(fname, os.O_RDWR, 0644)
		if err != nil {
			slog.Warn("error opening", "fname", fname, "error", err)
			continue
		}
		fd.Write([]byte(v))
		fd.Close()
	}
}

/*
* TODO: Initialize our updates table with route table at start up.
 */
func (rm *RouteManager) routeMonitor() {

	ch := make(chan netlink.RouteUpdate)
	done := make(chan struct{})
	defer close(done)

	err := netlink.RouteSubscribe(ch, done)
	if err != nil {
		slog.Error("failed subscribing to route updates", "error", err)
		return
	}

	for {
		ru := <-ch

		// Don't advertise routes we inserted.
		if rm.findSelfRouteLocked(ru.Dst) != nil {
			slog.Debug("own route, not announced")
			continue
		}

		if rm.classifyUpdate(&ru) {
			rm.updateLearned(ru.Type, ru.Route.Dst)
			rm.routesReady()
		}
	}
}

/*
* Make sure interface name is a tunnel device.
* TODO: Do we need to do this or are RawFlags enough?
 */
func (rm *RouteManager) qualifyLinkName(link netlink.Link) bool {

	ifnames := []string{"gpd", "tun"}
	for _, n := range ifnames {
		if strings.Contains(link.Attrs().Name, n) {
			return true
		}
	}
	return false
}

/*
* Equality test for net.IPNet
 */
func (rm *RouteManager) netEqual(a *net.IPNet, b *net.IPNet) bool {

	sza, _ := a.Mask.Size()
	szb, _ := b.Mask.Size()
	if sza != szb {
		return false
	}

	if a.IP.Equal(b.IP) {
		return true
	}
	return false
}

/*
* Read routes and initLearnedUpdates ourselves.
 */
func (rm *RouteManager) initLearnedUpdates() {

	netFamilies := []int{netlink.FAMILY_V4, netlink.FAMILY_V6}

	for _, family := range netFamilies {

		routes, err := netlink.RouteList(nil, family)
		if err != nil {
			slog.Error("failed getting routes", "error", err)
			return
		}

		for _, rt := range routes {

			ru := netlink.RouteUpdate{Type: unix.RTM_NEWROUTE,
				Route: rt,
			}
			if rm.classifyUpdate(&ru) {
				slog.Info("adding to learned updates", "rt", rt)
				rm.updateLearned(unix.RTM_NEWROUTE, ru.Dst)
			}
		}
	}
}

/*
* How many route have we learned?
 */
func (rm *RouteManager) LearnedCount() int {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	c := len(rm.learnedUpdates)
	return c
}

/*
* Determine if this is a route we're intereseted in.
* Looking for non-local, non-host routes.
 */
func (rm *RouteManager) classifyUpdate(ru *netlink.RouteUpdate) bool {

	l := rm.ifm.GetLinkByIndex(ru.LinkIndex)
	if l == nil {
		slog.Warn("no such interface", "index", ru.LinkIndex)
		return false
	}

	// Make sure name looks like a tunnel
	if !rm.qualifyLinkName(l) {
		return false
	}

	// No host routes
	bits, _ := ru.Dst.Mask.Size()
	if bits == 32 || bits == 128 {
		return false
	}

	// Don't goof with multicast or link locak
	if ru.Dst.IP.IsLinkLocalMulticast() ||
		ru.Dst.IP.IsLinkLocalUnicast() ||
		ru.Dst.IP.IsMulticast() {
		return false
	}

	// This is not a default route
	if rm.netEqual(ru.Dst, rm.def4Net) || rm.netEqual(ru.Dst, rm.def6Net) {
		return false
	}

	// If this is a route using a tunnel device...
	return rm.IsTunnelRoute(ru)
}

/*
* Check for route using a link that is a tunnel.
 */
func (rm *RouteManager) IsTunnelRoute(ru *netlink.RouteUpdate) bool {

	return rm.ifm.GetTunnelByIndex(ru.LinkIndex) != nil
}

/*
* Find matching routes.  If deleting, default routes are wild cards.
 */
func (rm *RouteManager) findLearnedUpdate(op uint16, dst *net.IPNet) []*RouteUpdate {

	matches := []*RouteUpdate{}
	for i, u := range rm.learnedUpdates {

		if op == unix.RTM_DELROUTE {
			if rm.netEqual(&u.Dst, dst) ||
				rm.netEqual(rm.def4Net, dst) ||
				rm.netEqual(rm.def6Net, dst) {
				matches = append(matches, &rm.learnedUpdates[i])
			}
		} else {
			if rm.netEqual(&u.Dst, dst) {
				matches = append(matches, &rm.learnedUpdates[i])
			}
		}
	}
	return matches
}

/*
* Find routes that match the update and set flags and/or add as needed.
 */
func (rm *RouteManager) updateLearned(op uint16, dst *net.IPNet) {

	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	ru := RouteUpdate{Op: op, Dst: *dst}
	matches := rm.findLearnedUpdate(op, dst)

	if len(matches) == 0 {
		if op == unix.RTM_NEWROUTE &&
			!rm.netEqual(rm.def4Net, dst) &&
			!rm.netEqual(rm.def6Net, dst) {
			rm.learnedUpdates = append(rm.learnedUpdates, ru)
			rm.EnableRouting()
		}
	} else {
		for _, m := range matches {
			m.Op = op
		}
		if op == unix.RTM_DELROUTE {
			rm.DisableRouting() // TODO: Support multiple gw interfaces.
		}
	}

	for _, u := range rm.learnedUpdates {

		op := fmt.Sprintf("0x%x", u.Op)
		d := IPNetToCidr(&u.Dst)

		slog.Debug("updates", "op", op, "dst", d)
	}
}

/*
* Make a copy of the current table under the lock. Table should not be
* very large so this should be efficient enough.
 */
func (rm *RouteManager) GetRouteUpdates() []RouteUpdate {

	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	r := []RouteUpdate{}
	r = append(r, rm.learnedUpdates...)
	return r
}

/*
* Alert anyone listening on  the update channel.
 */
func (rm *RouteManager) routesReady() {
	rm.updateMutex.Lock()
	rm.updateCount += 1
	rm.updateCondition.Signal()
	rm.updateMutex.Unlock()
}

/*
* Wait for update to show up on the update channel.
 */
func (rm *RouteManager) WaitForUpdate() {

	rm.updateMutex.Lock()

	for rm.updateCount == 0 {
		rm.updateCondition.Wait()
	}

	if rm.updateCount > 0 {
		rm.updateCount -= 1
	}
	rm.updateMutex.Unlock()
}

func (rm *RouteManager) GetDefaultLink() netlink.Link {

	_, g, _ := net.ParseCIDR("8.8.8.8/32")

	routes, err := netlink.RouteGet(g.IP)
	if err != nil {
		slog.Warn("get default link failed", "error", err)
		return nil
	}

	if len(routes) > 0 {
		for _, r := range routes {
			l := rm.ifm.GetLinkByIndex(r.LinkIndex)
			return l
		}
	}
	return nil
}

/*
* Look for the destination that were added via rm.AddRoute
 */
func (rm *RouteManager) findSelfRoute(dst *net.IPNet) *netlink.Route {

	for _, rt := range rm.selfRoutes {
		if rm.netEqual(dst, rt.Dst) {
			return &rt
		}
	}
	return nil
}

/*
* Find self route with mutex lock on.
 */
func (rm *RouteManager) findSelfRouteLocked(dst *net.IPNet) *netlink.Route {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	return rm.findSelfRoute(dst)
}

/*
* Add a route to the kernel.
 */
func (rm *RouteManager) AddRoute(dest string, gateway string) {

	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	_, dst, err := net.ParseCIDR(dest)
	if err != nil {
		slog.Warn("Failed parsing", "dest", dest)
		return
	}

	if rm.findSelfRoute(dst) != nil {
		slog.Info("route exists, skipping", "route", dest)
		return
	}

	// Look up the route to the gateway.
	// We will need the LinkIndex for this route in a moment.
	gw := net.ParseIP(gateway)
	gwrt, err := netlink.RouteGet(gw)
	if err != nil {
		slog.Warn("route lookup failure", "error", err)
		return
	}

	if len(gwrt) > 0 {
		rt := netlink.Route{LinkIndex: gwrt[0].LinkIndex, Dst: dst, Gw: gw}
		if err := netlink.RouteAdd(&rt); err != nil {
			slog.Warn("error adding route", "error", err)
			return
		}
		rm.selfRoutes = append(rm.selfRoutes, rt)
	} else {
		slog.Warn("No route found to gateway", "addr", gateway)
	}
}

/*
* Delete the route from our ownRoute table and the kernel.
 */
func (rm *RouteManager) DeleteRoute(dest string, gateway string) {

	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	_, dst, err := net.ParseCIDR(dest)
	if err != nil {
		slog.Warn("Failed parsing", "dest", dest)
		return
	}

	rt := rm.findSelfRoute(dst)
	if rt == nil {
		slog.Info("not a self route", "route", dest)
		return
	}

	if err := netlink.RouteDel(rt); err != nil {
		slog.Warn("error deleting route", "error", err)
		return
	}

	rm.delSelfRoute(dst)
}

func (rm *RouteManager) delSelfRoute(dst *net.IPNet) *netlink.Route {

	for i, rt := range rm.selfRoutes {
		if rm.netEqual(dst, rt.Dst) {
			rm.selfRoutes = append(rm.selfRoutes[:i], rm.selfRoutes[i+1:]...)
			return &rt
		}
	}
	return nil
}
