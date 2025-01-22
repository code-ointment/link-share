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

		slog.Info("Announce Update", "op", rt.Op)
		if rt.Op == unix.RTM_NEWROUTE {
			slog.Info("add route ",
				"gw", gw, "dst", rt.Dest, "domain", domain)

			// hmmm...
			if pe.rm.AddRoute(rt.Dest, gw) {

				intf := pe.ifm.GetDefaultLink()
				if pe.dnsConfig.BackupConfig() {
					pe.dnsConfig.SetNameServers(intf.Attrs().Name, ns)
					pe.dnsConfig.SetDomains(intf.Attrs().Name, sd)
					pe.dnsConfig.Commit()
				}
			}
		}

		if rt.Op == unix.RTM_DELROUTE {
			slog.Info("delete route ",
				"gw", gw, "dst", rt.Dest, "domain", domain)
			// hmmm...
			if pe.rm.DeleteRoute(rt.Dest, gw) {
				pe.dnsConfig.RestoreConfig()
			}
		}
	}
}
