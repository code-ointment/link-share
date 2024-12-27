package inet

import (
	"fmt"
	"net"

	"golang.org/x/sys/unix"
)

/*
* Extract the IP byte value from the net.Addr.
* Seems a bit whacky to me.
 */
func AddrToIP(addr net.Addr) net.IP {

	//t := fmt.Sprintf("%T", addr)
	//slog.Debug("ADDR", "type", t)

	uaddr, ok := addr.(*net.UDPAddr)
	if ok {
		return uaddr.IP
	}

	ipaddr, ok := addr.(*net.IPNet)
	if ok {
		return ipaddr.IP
	}

	taddr, ok := addr.(*net.TCPAddr)
	if ok {
		return taddr.IP
	}

	return net.IP{}
}

/*
* Return net as a cidr string.
 */
func IPNetToCidr(a *net.IPNet) string {

	bits, _ := a.Mask.Size()
	dstStr := fmt.Sprintf("%s/%d", a.IP.String(), bits)
	return dstStr
}

func NlFlagsToString(flgs uint16) string {

	xtab := []struct {
		v   uint16
		txt string
	}{
		{unix.NLM_F_REPLACE, "NLM_F_REPLACE"},
		{unix.NLM_F_EXCL, "NLM_F_EXCL"},
		{unix.NLM_F_CREATE, "NLM_F_CREATE"},
		{unix.NLM_F_APPEND, "NLM_F_APPEND"},
	}

	// NLM_F_BULK not meaningful in this context.
	stripped := flgs
	extra := ""
	if flgs&unix.NLM_F_BULK == unix.NLM_F_BULK {
		stripped &= ^uint16(unix.NLM_F_BULK)
		extra = " NLM_F_BULK"
	}

	for i := 0; i < len(xtab); i++ {
		if xtab[i].v == stripped {
			return xtab[i].txt + extra
		}
	}
	return fmt.Sprintf("0x%x", flgs)
}
