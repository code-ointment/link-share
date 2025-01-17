package inet

import (
	"bufio"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
)

type ResolvectlEntry struct {
	LinkName         string
	LinkIndex        int
	Scope            string
	Protocols        string
	CurrentDnsServer string
	DnsServers       string
	DnsDomains       string
}

/*
* Parse the initial line and pass to balance of parsing.
* Link 2 (ens160)
 */
func NewResolvectlEntry(line string, scanner *bufio.Scanner) *ResolvectlEntry {

	re := ResolvectlEntry{}

	indexExp := regexp.MustCompile(` [0-9]+ `)
	matches := indexExp.FindAllString(line, -1)
	if len(matches) == 0 {
		slog.Warn("parse failure", "no ifindex in", line)
		return nil
	}
	re.LinkIndex, _ = strconv.Atoi(strings.TrimSpace(matches[0]))

	ifExp := regexp.MustCompile(` \([a-z0-9]+\)`)
	matches = ifExp.FindAllString(line, -1)
	if len(matches) == 0 {
		slog.Warn("parse failure", "ifname not found", line)
		return nil
	}

	re.LinkName = strings.Replace(matches[0], "(", "", 1)
	re.LinkName = strings.Replace(re.LinkName, ")", "", 1)
	re.LinkName = strings.TrimSpace(re.LinkName)
	re.Parse(scanner)
	return &re
}

// Look like cut-n-paste strikes again.
func (re *ResolvectlEntry) getValue(line string) string {

	fields := strings.Split(line, ":")
	return strings.TrimSpace(fields[1])
}

func (re *ResolvectlEntry) Parse(scanner *bufio.Scanner) {

	for scanner.Scan() {
		line := scanner.Text()

		if strings.Contains(line, "Current Scopes") {
			re.Scope = re.getValue(line)
			continue
		}

		if strings.Contains(line, "Protocols") {
			re.Protocols = re.getValue(line)
			continue
		}

		if strings.Contains(line, "Current DNS Server") {
			re.CurrentDnsServer = re.getValue(line)
			continue
		}

		if strings.Contains(line, "DNS Servers") {
			re.DnsServers = re.getValue(line)
			continue
		}

		if strings.Contains(line, "DNS Domain") {
			re.DnsDomains = re.getValue(line)
			continue
		}

		return
	}
}
