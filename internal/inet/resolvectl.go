package inet

import (
	"bufio"
	"encoding/json"
	"log/slog"
	"os"
	"strings"

	"github.com/code-ointment/link-share/internal/linux"
)

type Resolvectl struct {
	GlobalProtocols string
	ResolvConfMode  string

	Links []*ResolvectlEntry
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

	rc.Links = nil
	result := linux.Run([]string{"resolvectl", "status"})
	scanner := bufio.NewScanner(strings.NewReader(result.Stdout))

	for scanner.Scan() {

		line := scanner.Text()
		if strings.Contains(line, "Global") {
			rc.parseGlobal(scanner)
			continue
		}

		if strings.Contains(line, "Link") {
			entry := NewResolvectlEntry(line, scanner)
			rc.Links = append(rc.Links, entry)
		}
	}
}

/*
* Parse global entry
 */
func (rc *Resolvectl) parseGlobal(scanner *bufio.Scanner) {

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "Protocols") {
			rc.GlobalProtocols = rc.getValue(line)
			continue
		}

		if strings.Contains(line, "resolv.conf mode") {
			rc.ResolvConfMode = rc.getValue(line)
			continue
		}
		return
	}
}

func (rc *Resolvectl) getValue(line string) string {

	fields := strings.Split(line, ":")
	return strings.TrimSpace(fields[1])
}

/*
* Lookup the entry assocaited with the interface, otherwise return nil
 */
func (rc *Resolvectl) findEntryByIntf(intf string) *ResolvectlEntry {

	for _, entry := range rc.Links {
		if strings.EqualFold(intf, entry.LinkName) {
			return entry
		}
	}
	return nil
}

func (rc *Resolvectl) addResolvectlEntry(intf string) *ResolvectlEntry {

	// Forcing a default, fix later. Not sure we need LinkIndex for our
	// purpose
	re := ResolvectlEntry{
		Scope:     "DNS",
		Protocols: "+DefaultRoute -LLMNR -mDNS -DNSOverTLS DNSSEC=no/unsupported",
		LinkName:  intf,
	}
	rc.Links = append(rc.Links, &re)
	return &re
}

// DnsConfig implementation.

// Space separated of servers.  Limit is 3.  See man resolv.conf
func (rc *Resolvectl) SetNameServers(intf string, servers string) {

	entry := rc.findEntryByIntf(intf)
	if entry == nil {
		entry = rc.addResolvectlEntry(intf)
		entry.DnsServers = servers
		entry.CurrentDnsServer = servers
	} else {
		entry.DnsServers = servers
		entry.CurrentDnsServer = servers
	}
}

func (rc *Resolvectl) GetNameServers(intf string) string {
	entry := rc.findEntryByIntf(intf)
	slog.Info("resolvectl", "entry", entry)
	if entry != nil {
		return entry.DnsServers
	}
	return ""
}

// Space separated list of search domains
func (rc *Resolvectl) SetDomains(intf string, domains string) {
	entry := rc.findEntryByIntf(intf)
	if entry == nil {
		entry = rc.addResolvectlEntry(intf)
		entry.DnsDomains = domains
	} else {
		entry.DnsDomains = domains
	}
}

func (rc *Resolvectl) GetDomains(intf string) string {
	entry := rc.findEntryByIntf(intf)
	if entry != nil {
		return entry.DnsDomains
	}
	return ""
}

// Backup the current configuration
func (rc *Resolvectl) BackupConfig() {

	if len(rc.Links) == 0 {
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

	vstr := rc.GetDomains(l.Attrs().Name)
	domains := strings.Split(vstr, " ")
	vec := []string{"resolvectl", "domain", l.Attrs().Name}
	vec = append(vec, domains...)

	result := linux.Run(vec)
	if result.Err != nil || result.ExitCode != 0 {
		slog.Warn("set domain failed",
			"error", result.Err, "exit code", result.ExitCode)
		return false
	}

	vstr = rc.GetNameServers(l.Attrs().Name)
	nameservers := strings.Split(vstr, " ")
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
