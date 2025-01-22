package inet

/*
* Define an interface used to control DNS config.
* Configure new config BackupConfig(), Set*,Commit()
* Restore old config - RestoreConfig(), Commit()
 */
type DnsConfig interface {
	// Space separated of servers.  Limit is 3.  See man resolv.conf
	SetNameServers(intf string, servers string)
	GetNameServers(intf string) string

	// Space separated list of search domains
	SetDomains(intf string, domains string)
	GetDomains(intf string) string
	// Fetch the current DNS configuration
	ReadConfig()
	// Backup the current configuration
	BackupConfig() bool
	// Restore previously backed up configuration
	RestoreConfig()
	// Commit changes
	Commit() bool
}
