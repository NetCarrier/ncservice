package ncservice

import (
	"fmt"
	"log/slog"
	"os"
)

var lvls = []slog.Level{
	slog.LevelDebug,
	slog.LevelInfo,
	slog.LevelWarn,
	slog.LevelError,
}

func DecodeLogLevel(id string) slog.Level {
	for _, candidate := range lvls {
		if candidate.String() == id {
			return slog.Level(candidate)
		}
	}
	panic(fmt.Sprintf("%s is not a log level", id))
}

func LogFatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}
