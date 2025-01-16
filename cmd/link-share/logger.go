package main

import (
	"log/slog"
	"path/filepath"

	logfile "github.com/code-ointment/log-writer/logfile"
)

/*
* Configure log at module load time.
 */
func init() {

	args := GetArgs()

	opts := slog.HandlerOptions{
		AddSource: true,
		Level:     args.LogLevel,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.SourceKey {
				src, _ := a.Value.Any().(*slog.Source)
				if src != nil {
					src.File = filepath.Base(src.File)
				}
			}
			return a
		}}

	lw := logfile.NewLogFileWriter(
		"/var/log/code-ointment/link-share/link-share.log",
		5, 1024*1024)

	//logger := slog.New(slog.NewJSONHandler(os.Stdout, &opts))
	logger := slog.New(slog.NewTextHandler(lw, &opts))
	slog.SetDefault(logger)
}
