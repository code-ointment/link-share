package engine

/*
* Handles Annoucements.  Not a lot just yet.
 */
import (
	"log/slog"

	"github.com/code-ointment/link-share/link_proto"
	"golang.org/x/sys/unix"
)

func (pe *ProtocolEngine) AnnounceHandler(an *link_proto.Announce) {

	rts := an.GetRoutes()
	gw := an.GetGateway()
	domain := an.GetDomain()
	ns := an.GetNameservers()
	sd := an.GetSearchdomains()

	slog.Info("dns config",
		"nameservers", ns,
		"searchdomains", sd)
	pe.configured = true // switch to atomic variable

	for _, rt := range rts {

		if rt.Op == unix.RTM_NEWROUTE {
			slog.Info("add route ",
				"gw", gw, "dst", rt.Dest, "domain", domain)

			pe.rm.AddRoute(rt.Dest, gw)
		}

		if rt.Op == unix.RTM_DELROUTE {
			slog.Info("delete route ",
				"gw", gw, "dst", rt.Dest, "domain", domain)
			pe.rm.DeleteRoute(rt.Dest, gw)
		}
	}
}
