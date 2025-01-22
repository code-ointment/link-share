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
)

const (
	pidFile = "/var/tmp/link-share.pid"
)

// Using global so signal handlers can access.
var eng *engine.ProtocolEngine

/*
* record our PID in /var/tmp
 */
func recordPid() {

	fd, err := os.OpenFile(pidFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		slog.Error("can't write pid file", "error", err)
		os.Exit(1)
	}

	pid := os.Getpid()
	fmt.Fprintf(fd, "%d", pid)
	fd.Close()

}

/*
* Wait for an exit signal.
 */
func sigWaitHandler() {

	slog.Info("link-share waiting for exit signal")
	intChan := make(chan os.Signal, 1)
	signal.Notify(intChan, os.Interrupt, syscall.SIGTERM, syscall.SIGUSR1)
	<-intChan
	slog.Info("Exiting")
	// Wait for logwriter to finish archiving.
	logwriter.Flush()
	os.Remove(pidFile)
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
	os.Remove(pidFile)

	if eng != nil {
		eng.Shutdown()
	}
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

	go sigQuitHandler()

	eng = engine.NewProtocolEngine()
	eng.Start()

	go heloThread(eng)
	recordPid()

	sigWaitHandler()

	eng.Shutdown()
	os.Exit(0)
}
