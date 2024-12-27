package engine

//
// Tracks participating systems.
//
import (
	"net"
	"time"

	"github.com/code-ointment/link-share/internal/consts"
)

type Host struct {
	State      int
	IP         net.IP
	UpdateTime int64
}

func NewHost(ip net.IP) *Host {
	h := Host{
		State:      consts.DOWN,
		IP:         ip,
		UpdateTime: time.Now().Unix(),
	}

	return &h
}
