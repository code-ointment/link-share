package inet

import (
	"bufio"
	"encoding/json"
	"log/slog"
	"os"
	"strings"

	"github.com/code-ointment/link-share/internal/linux"
)

/*
* Note that domain and search were different a long time ago.  New school
* is to collapse these items together.
 */
type Resolvectl struct {
	NameServers string
	Domains     string
}

func NewResolvectl() *Resolvectl {

	rc := Resolvectl{}
	rc.ReadConfig()

	rc.initTmp()
	return &rc
}

const (
	backupDir      string = "/var/tmp/link-share"
	backupJsonFile string = "/var/tmp/link-share/backup.json"
)

func (rc *Resolvectl) initTmp() {

	if _, err := os.Stat(backupDir); err != nil {
		os.MkdirAll(backupDir, 0700)
	}
}

func (rc *Resolvectl) ReadConfig() {

	result := linux.Run([]string{"resolvectl", "status"})
	scanner := bufio.NewScanner(strings.NewReader(result.Stdout))

	for scanner.Scan() {

		line := scanner.Text()
		if strings.Contains(line, "DNS Servers") {
			rc.NameServers = rc.getValue(line)
			continue
		}

		if strings.Contains(line, "DNS Domain") {
			rc.Domains = rc.getValue(line)
		}
	}
}

func (rc *Resolvectl) getValue(line string) string {

	fields := strings.Split(line, ":")
	return strings.TrimSpace(fields[1])
}

// DnsConfig implementation.

// Space separated of servers.  Limit is 3.  See man resolv.conf
func (rc *Resolvectl) SetNameServers(servers string) {
	rc.NameServers = servers
}

func (rc *Resolvectl) GetNameServers() string {
	return rc.NameServers
}

// Space separated list of search domains
func (rc *Resolvectl) SetDomains(domains string) {
	rc.Domains = domains
}

func (rc *Resolvectl) GetDomains() string {
	return rc.Domains
}

// Backup the current configuration
func (rc *Resolvectl) BackupConfig() {

	if len(rc.NameServers) == 0 && len(rc.Domains) == 0 {
		slog.Warn("resolvectl ReadConfig not invoked, nothing to back up")
		return
	}

	rc.initTmp()
	fd, err := os.OpenFile(backupJsonFile, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0600)
	if err != nil {
		slog.Warn("error opening backup", "error", err)
		return
	}
	defer fd.Close()

	b, err := json.Marshal(rc)
	if err != nil {
		slog.Warn("error marshalling", "error", err)
		return
	}
	fd.Write(b)

}

// Restore previously backed up configuration
func (rc *Resolvectl) RestoreConfig() {

	if _, err := os.Stat(backupJsonFile); err != nil {
		slog.Debug("no backup available")
		return
	}

	b, err := os.ReadFile(backupJsonFile)
	if err != nil {
		slog.Warn("error reading backup", "error", err)
	}

	err = json.Unmarshal(b, rc)
	if err != nil {
		slog.Warn("error marshalling", "error", err)
		return
	}
}

// Commit changes
func (rc *Resolvectl) Commit() bool {

	ifm := NewInterfaceManager()
	l := ifm.GetDefaultLink()
	domains := strings.Split(rc.Domains, " ")
	vec := []string{"resolvectl", "domain", l.Attrs().Name}
	vec = append(vec, domains...)

	result := linux.Run(vec)
	if result.Err != nil || result.ExitCode != 0 {
		slog.Warn("set domain failed",
			"error", result.Err, "exit code", result.ExitCode)
		return false
	}

	nameservers := strings.Split(rc.NameServers, " ")
	vec = []string{"resolvectl", "dns", l.Attrs().Name}
	vec = append(vec, nameservers...)

	result = linux.Run(vec)
	if result.Err != nil || result.ExitCode != 0 {
		slog.Warn("set nameserver failed",
			"error", result.Err, "exit code", result.ExitCode)
		return false
	}
	return true
}
