package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/code-ointment/link-share/internal/engine"
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
	/*
	 * Call clean up and other process termination stuff.
	 */
	os.Exit(0)
}

/*
* Dump stack similarly to java when a  QUIT is recieved.
 */
func siqQuit() {

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGQUIT)
	buf := make([]byte, 1<<20)
	for {
		<-sigs
		stacklen := runtime.Stack(buf, true)
		fmt.Printf("=== received SIGQUIT ===\n*** goroutine dump...\n%s\n*** end\n", buf[:stacklen])
	}
}

/*
* Sends periodic helo.
 */
func heloThread(eng *engine.ProtocolEngine) {
	for {
		eng.SendHelo()
		time.Sleep(time.Minute * 1)
	}
}

func main() {

	//rtTest()
	go siqQuit()

	eng := engine.NewProtocolEngine()
	eng.Start()

	go heloThread(eng)
	sigWait()
}
