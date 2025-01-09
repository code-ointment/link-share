package inet

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

type ResolveConf struct {
	NameServers string
	Domains     string
}

func NewResolveConf() *ResolveConf {

	rc := ResolveConf{}
	rc.ReadConfig()

	return &rc
}

const (
	resolv_conf string = "/etc/resolv.conf"
	backupFile  string = "/var/tmp/link-share/backup.conf"
)

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
		// domain ?  is anyone still using it?
	}
}

// DnsConfig implementation.

// Space separated of servers.  Limit is 3.  See man resolv.conf
func (rc *ResolveConf) SetNameServers(servers string) {
	rc.NameServers = servers
}

func (rc *ResolveConf) GetNameServers() string {
	return rc.NameServers
}

// Space separated list of search domains
func (rc *ResolveConf) SetDomains(domains string) {
	rc.Domains = domains
}

func (rc *ResolveConf) GetDomains() string {
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

func (rc *ResolveConf) BackupConfig() {

	src, err := os.Open(resolv_conf)
	if err != nil {
		slog.Warn("failed opening "+resolv_conf, "error", err)
		return
	}
	defer src.Close()

	dest, err := os.OpenFile(backupFile, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0600)
	if err != nil {
		slog.Warn("failed opening "+backupFile, "error", err)
		return
	}
	defer dest.Close()

	io.Copy(dest, src)
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
}
