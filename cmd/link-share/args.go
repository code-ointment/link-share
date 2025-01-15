package main

/*
* Holder for command line arguments.
 */
import (
	"flag"
	"log/slog"
	"strings"
	"sync"
)

/*
* Not a lot of options at present.
 */
type Args struct {
	LogLevel slog.Level
}

var cmdLineArgs *Args
var cmdLineLock sync.Mutex

func GetArgs() *Args {
	cmdLineLock.Lock()
	defer cmdLineLock.Unlock()

	if cmdLineArgs == nil {
		cmdLineArgs = &Args{}
		cmdLineArgs.init()
	}
	return cmdLineArgs
}

func (a *Args) init() {

	var levelStr string

	flag.StringVar(&levelStr, "log", "INFO", "logging level [ DEBUG,INFO,WARN ]")

	flag.Parse()
	a.LogLevel = a.parseLevel(levelStr)
}

func (a *Args) parseLevel(levelStr string) slog.Level {

	switch strings.ToUpper(levelStr) {
	case "INFO":
		return slog.LevelInfo
	case "DEBUG":
		return slog.LevelDebug
	case "WARN":
		return slog.LevelWarn
	}
	return slog.LevelInfo
}
