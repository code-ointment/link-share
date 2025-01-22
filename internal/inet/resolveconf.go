package inet

/*
* Handle updating a typical /etc/resolv.conf configuration
 */
import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

const (
	globalDnsDomain string = "~."
	resolv_conf     string = "/etc/resolv.conf"
	backupFile      string = "/var/tmp/link-share/backup.conf"
)

type ResolveConf struct {
	NameServers string
	Domains     string
}

func NewResolveConf() *ResolveConf {

	rc := ResolveConf{}
	rc.ReadConfig()
	rc.initTmp()
	return &rc
}

func (rc *ResolveConf) initTmp() {

	if _, err := os.Stat(backupDir); err != nil {
		os.MkdirAll(backupDir, 0700)
	}
}

/*
 */
func (rc *ResolveConf) getValue(line string) string {

	fields := strings.Split(line, " ")
	values := strings.Join(fields[1:], " ")
	return strings.TrimSpace(values)
}

func (rc *ResolveConf) ReadConfig() {

	fd, err := os.Open(resolv_conf)
	if err != nil {
		slog.Warn("failed opening /etc/resolv.conf", "error", err)
	}
	defer fd.Close()

	scanner := bufio.NewScanner(fd)
	for scanner.Scan() {

		line := scanner.Text()
		if strings.Contains(line, "nameserver") {
			rc.NameServers = rc.getValue(line)
			continue
		}

		if strings.Contains(line, "search") {
			rc.Domains = rc.getValue(line)
		}
	}
}

// DnsConfig implementation.

// Space separated of servers.  Limit is 3.  See man resolv.conf
func (rc *ResolveConf) SetNameServers(intf string, servers string) {
	rc.NameServers = servers
}

func (rc *ResolveConf) GetNameServers(intf string) string {
	return rc.NameServers
}

// Space separated list of search domains
func (rc *ResolveConf) SetDomains(intf string, domains string) {

	// Don't think resolv.conf can handle the global dns marker.
	if strings.Contains(domains, globalDnsDomain) {
		tmp := domains
		tmp = strings.Replace(tmp, globalDnsDomain, "", 1)
		rc.Domains = tmp
		return
	}
	rc.Domains = domains
}

func (rc *ResolveConf) GetDomains(intf string) string {
	return rc.Domains
}

func (rc *ResolveConf) Commit() bool {

	fields := strings.Split(rc.NameServers, " ")
	if len(fields) > 3 {
		slog.Error("invalid number of nameservers, 3 or less required")
		return false
	}

	fd, err := os.OpenFile(resolv_conf, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
	if err != nil {
		slog.Warn("failed opening "+resolv_conf, "error", err)
		return false
	}
	defer fd.Close()

	fmt.Fprintf(fd, "search %s\n", rc.Domains)

	for _, f := range fields {
		fmt.Fprintf(fd, "nameserver %s\n", f)
	}

	return false
}

func (rc *ResolveConf) BackupConfig() bool {

	if _, err := os.Stat(backupFile); err == nil {
		slog.Warn("dns config already backed up")
		return false
	}

	src, err := os.Open(resolv_conf)
	if err != nil {
		slog.Warn("failed opening "+resolv_conf, "error", err)
		return false
	}
	defer src.Close()

	dest, err := os.OpenFile(backupFile, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0600)
	if err != nil {
		slog.Warn("failed opening "+backupFile, "error", err)
		return false
	}
	defer dest.Close()

	io.Copy(dest, src)
	return true
}

/*
* Converse of backup.
 */
func (rc *ResolveConf) RestoreConfig() {

	src, err := os.Open(backupFile)
	if err != nil {
		slog.Warn("failed opening "+backupFile, "error", err)
		return
	}
	defer src.Close()

	dest, err := os.OpenFile(resolv_conf, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0600)
	if err != nil {
		slog.Warn("failed opening "+resolv_conf, "error", err)
		return
	}
	defer dest.Close()

	io.Copy(dest, src)
	os.Remove(backupFile)
}
