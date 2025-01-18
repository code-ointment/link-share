package consts

const (
	ListenAddr        string = "[::]:10210"
	GroupAddr         string = "ff02::210"
	LinkLocalPrefix6  string = "fe80"
	MulticastPrefix6  string = "ff00"
	LinkLocalPrexfix4 string = "169.254.0.0/16"
	MulticastPrefix4  string = "224.0.0.0/24"
)

const (
	MaxDatagramSize int = 9000  // Jumbo frame size.  Get from Interface maybe?
	ListenPort      int = 10210 // Any non-priviledged port.
	UP              int = 1
	DOWN            int = 2
)

type LinkClass int

const (
	TUNNEL LinkClass = iota + 1
	STANDARD
	UNUSED
)

const (
	POLL_INTERVAL int = 60 // seconds
)
