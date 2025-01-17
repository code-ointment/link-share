package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/code-ointment/link-share/internal/consts"
	"github.com/code-ointment/link-share/internal/engine"
	logwriter "github.com/code-ointment/log-writer"
	"github.com/vishvananda/netlink"
)

func rtTest() {

	l, _ := netlink.LinkByName("ens160")
	routes, err := netlink.RouteList(l, netlink.FAMILY_V4|netlink.FAMILY_V6)
	if err != nil {
		slog.Error("failed getting routes", "error", err)
		return
	}
	for _, rt := range routes {
		slog.Info("Route", "rt", rt)
	}
}

/*
* Wait for an exit signal.
 */
func sigWait() {

	intChan := make(chan os.Signal, 1)
	signal.Notify(intChan, os.Interrupt, syscall.SIGTERM, syscall.SIGUSR1)
	<-intChan
	slog.Info("Exiting")
	logwriter.Flush()
	/*
	 * Call clean up and other process termination stuff.
	 */
	os.Exit(0)
}

/*
* Dump stack similarly to java when a  QUIT is recieved.  Exit while we're
* at it.
 */
func sigQuitHandler() {

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGQUIT)

	<-sigs
	buf := make([]byte, 1048576) // 1MB

	stacklen := runtime.Stack(buf, true)
	fmt.Printf("*** goroutine dump...\n%s\n*** end\n", buf[:stacklen])
	logwriter.Flush()
	os.Exit(int(syscall.SIGQUIT))
}

/*
* Sends periodic helo.
 */
func heloThread(eng *engine.ProtocolEngine) {
	for {
		eng.SendHelo()
		eng.HostAccounting()
		time.Sleep(time.Duration(consts.POLL_INTERVAL) * time.Second)
	}
}

func main() {

	//rtTest()
	go sigQuitHandler()

	eng := engine.NewProtocolEngine()
	eng.Start()

	go heloThread(eng)
	sigWait()
}
