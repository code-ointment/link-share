package inet

import (
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

type RouteManager struct {
	ifm          *InterfaceManager
	localUpdates []RouteUpdate   // Routes we learned from the kernel
	selfRoutes   []netlink.Route // Routes the manager was asked to add

	mutex   sync.Mutex
	updated chan struct{}

	def6Net *net.IPNet // Handy constants
	def4Net *net.IPNet
}

type RouteUpdate struct {
	Op  uint16
	Dst net.IPNet
}

func NewRouteManager(manager *InterfaceManager) *RouteManager {

	rm := RouteManager{
		ifm:     manager,
		updated: make(chan struct{}),
	}

	// handy defaults
	_, rm.def6Net, _ = net.ParseCIDR("::/0")
	_, rm.def4Net, _ = net.ParseCIDR("0.0.0.0/0")

	go rm.routeMonitor()

	return &rm

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
		if rm.findOwnRoute(ru.Dst) != nil {
			slog.Debug("own route, not announced")
			continue
		}

		if rm.classify(&ru) {
			rm.updateRoutes(ru.Type, ru.Route.Dst)
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
* Determine if this is a route we're intereseted in.
* Looking for non-local, non-host routes.
 */
func (rm *RouteManager) classify(ru *netlink.RouteUpdate) bool {

	l := rm.ifm.GetLinkByIndex(ru.LinkIndex)
	if l == nil {
		slog.Warn("no such interface", "index", ru.LinkIndex)
		return false
	}

	// Check interface name.
	if !rm.qualifyLinkName(l) {
		return false
	}

	// Host route check
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

	return true
}

/*
* Find matching routes.  If deleting, default routes are wild cards.
 */
func (rm *RouteManager) findRouteUpdate(op uint16, dst *net.IPNet) []*RouteUpdate {

	matches := []*RouteUpdate{}
	for i, u := range rm.localUpdates {

		if op == unix.RTM_DELROUTE {
			if rm.netEqual(&u.Dst, dst) ||
				rm.netEqual(rm.def4Net, dst) ||
				rm.netEqual(rm.def6Net, dst) {
				matches = append(matches, &rm.localUpdates[i])
			}
		} else {
			if rm.netEqual(&u.Dst, dst) {
				matches = append(matches, &rm.localUpdates[i])
			}
		}
	}
	return matches
}

/*
* Find routes that match the update and set flags and/or add as needed.
 */
func (rm *RouteManager) updateRoutes(op uint16, dst *net.IPNet) {

	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	ru := RouteUpdate{Op: op, Dst: *dst}
	matches := rm.findRouteUpdate(op, dst)

	if len(matches) == 0 {
		if op == unix.RTM_NEWROUTE &&
			!rm.netEqual(rm.def4Net, dst) &&
			!rm.netEqual(rm.def6Net, dst) {
			rm.localUpdates = append(rm.localUpdates, ru)
		}
	} else {
		for _, m := range matches {
			m.Op = op
		}
	}

	for _, u := range rm.localUpdates {

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
	r = append(r, rm.localUpdates...)
	return r
}

/*
* Alert anyone listening on  the update channel.
 */
func (rm *RouteManager) routesReady() {
	go func() {
		rm.updated <- struct{}{}
	}()
}

/*
* Wait for update to show up on the update channel.
 */
func (rm *RouteManager) WaitForUpdate() {
	<-rm.updated
}

func (rm *RouteManager) GetDefaultLink() {

	_, g, _ := net.ParseCIDR("8.8.8.8/32")

	routes, err := netlink.RouteGet(g.IP)
	if err != nil {
		slog.Warn("get default link failed", "error", err)
		return
	}

	if len(routes) > 0 {
		for _, r := range routes {
			slog.Info("def route", "index", r.LinkIndex)
		}
	}
}

/*
* Look for the destination that were added via rm.AddRoute
 */
func (rm *RouteManager) findOwnRoute(dst *net.IPNet) *netlink.Route {

	for _, rt := range rm.selfRoutes {
		if rm.netEqual(dst, rt.Dst) {
			return &rt
		}
	}
	return nil
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

	if rm.findOwnRoute(dst) != nil {
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

	rt := rm.findOwnRoute(dst)
	if rt == nil {
		slog.Info("not a self route", "route", dest)
		return
	}

	if err := netlink.RouteDel(rt); err != nil {
		slog.Warn("error deleting route", "error", err)
		return
	}

	rm.delOwnRoute(dst)
}

func (rm *RouteManager) delOwnRoute(dst *net.IPNet) *netlink.Route {

	for i, rt := range rm.selfRoutes {
		if rm.netEqual(dst, rt.Dst) {
			rm.selfRoutes = append(rm.selfRoutes[:i], rm.selfRoutes[i+1:]...)
			return &rt
		}
	}
	return nil
}
