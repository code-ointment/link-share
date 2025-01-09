package inet

/*
* Configure DNS name resolution.  Should be used by the client.
 */
import (
	"bufio"
	"log/slog"
	"os"
	"strings"

	"github.com/code-ointment/link-share/internal/linux"
)

type ResolverConfigFactory struct {
	useSystemdResolve bool
	useGenericResolv  bool
}

func NewResolverConfigFactory() *ResolverConfigFactory {

	rf := ResolverConfigFactory{}

	rf.useSystemdResolve = rf.isSystemdResolv()
	if rf.useSystemdResolve {
		slog.Debug("using systemd-resolve")
		return &rf
	}

	rf.useGenericResolv = true
	return &rf
}

/*
* Per /etc/vpnc/vpnc-script
 */
func (rf *ResolverConfigFactory) isSystemdResolv() bool {

	fd, err := os.Open("/etc/nsswitch.conf")
	if err != nil {
		slog.Warn("can't open /etc/nsswitch.conf", "error", err)
		return false
	}
	defer fd.Close()

	scanner := bufio.NewScanner(fd)
	for scanner.Scan() {
		line := scanner.Text()

		// systemd-resolve
		if strings.HasPrefix(line, "hosts") {
			if strings.Contains(line, "resolve") {
				return rf.checkSystemdResolveStatus()
			}

			// nss-dns with systemd-resolve
			if strings.Contains(line, "dns") {
				dest, err := os.Readlink("/etc/resolv.conf")
				if err != nil {
					slog.Warn("error reading /etc/resolv.conf", "error", err)
					return false
				}
				if strings.HasSuffix(dest, "stub-resolv.conf") {
					return rf.checkSystemdResolveStatus()
				}
			}
		}
	}
	return false
}

/*
* Make sure we can communicate with systemd-resolve
 */
func (rf *ResolverConfigFactory) checkSystemdResolveStatus() bool {

	result := linux.Run([]string{"resolvectl", "status"})
	if result.Err == nil && result.ExitCode == 0 {
		return true
	}

	result = linux.Run([]string{"systemd-resolve", "--status"})
	if result.Err == nil && result.ExitCode == 0 {
		return true
	}

	return false
}

func (rf *ResolverConfigFactory) GetDNSConfig() DnsConfig {

	if rf.useSystemdResolve {
		return NewResolvectl()
	}
	if rf.useGenericResolv {
		return NewResolveConf()
	}
	return nil
}
