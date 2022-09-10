package log

import (
	"io/ioutil"
	"log"
	"os"
)

var (
	// Info is the default logger for info-level messages.
	Info = log.Default()
	// Debug is the default logger for debug-level messages.
	Debug = log.Default()
	// Warn is the default logger for warn-level messages.
	Warn = log.Default()
	// Err is the default logger for err-level messages.
	Err = log.Default()
)

type LoggingConfig struct {
	EnableDebug bool
}

// Init initializes loggers in the package. If Init is not called, default
// loggers will be used for all levels of logging.
func Init(conf LoggingConfig) {
	flags := log.Ldate | log.Ltime
	debugOut := ioutil.Discard

	if conf.EnableDebug {
		debugOut = os.Stdout
		flags = log.Ldate | log.Ltime | log.Lshortfile
	}

	Info = log.New(os.Stdout, "I! ", flags)
	Debug = log.New(debugOut, "D! ", flags)
	Warn = log.New(os.Stdout, "W! ", flags)
	Err = log.New(os.Stderr, "E! ", flags)
}
