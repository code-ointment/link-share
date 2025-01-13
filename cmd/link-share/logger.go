package main

import (
	"log/slog"
	"os"
	"path/filepath"
)

/*
* Configure log at module load time.
 */
func init() {

	opts := slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelInfo,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.SourceKey {
				src, _ := a.Value.Any().(*slog.Source)
				if src != nil {
					src.File = filepath.Base(src.File)
				}
			}
			return a
		}}
	//logger := slog.New(slog.NewJSONHandler(os.Stdout, &opts))
	logger := slog.New(slog.NewTextHandler(os.Stdout, &opts))
	slog.SetDefault(logger)
}
