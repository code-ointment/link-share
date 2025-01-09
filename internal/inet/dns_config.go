package inet

/*
* Define an interface used to control DNS config.
* Configure new config BackupConfig(), Set*,Commit()
* Restore old config - RestoreConfig(), Commit()
 */
type DnsConfig interface {
	// Space separated of servers.  Limit is 3.  See man resolv.conf
	SetNameServers(servers string)
	GetNameServers() string

	// Space separated list of search domains
	SetDomains(domains string)
	GetDomains() string
	// Backup the current configuration
	BackupConfig()
	// Restore previously backed up configuration
	RestoreConfig()
	// Commit changes
	Commit() bool
}
