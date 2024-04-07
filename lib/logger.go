package lib

import (
	"os"

	"github.com/apsdehal/go-logger"
)

var (
	// Log global
	Log *logger.Logger
)

func init() {
	Log = initTWLogger()
}

func initTWLogger() *logger.Logger {
	var l *logger.Logger

	logLevel := logger.DebugLevel

	l, _ = logger.New("imap-scrub", 1, os.Stdout, logLevel)
	l.SetFormat("%{message}")

	return l
}
