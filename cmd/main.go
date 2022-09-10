package main

import (
	"flag"
	"fmt"
	"github.com/kevinoula/beach/collection"
	log "github.com/kevinoula/beach/log"
)

var flgVersion bool
var flgDebug bool

// Built at release time go build -ldflags="-X 'github.com/kevinoula/beach/main.Version=v0.1'"
var (
	Version = "dev"     // Version of the app.
	Commit  = "none"    // Commit hash.
	Date    = "unknown" // Date of the build.
	BuiltBy = "unknown" // The author or builder.
)

func main() {
	// Init
	flag.BoolVar(&flgVersion, "version", false, "print software version")
	flag.BoolVar(&flgVersion, "v", false, "print software version")
	flag.BoolVar(&flgDebug, "debug", false, "run in debug mode")
	flag.BoolVar(&flgDebug, "d", false, "run in debug mode")
	flag.Parse()

	if flgVersion {
		fmt.Println(Version)
		return
	}

	log.Init(log.LoggingConfig{EnableDebug: flgDebug})
	log.Info.Printf("Beach CLI version %s (%.8s) built on %s by %s", Version, Commit, Date, BuiltBy)

	// See past shells, connect to a new shell, or exit
	coll := collection.InitCollection()
	log.Debug.Printf("Collection of SSH: %v\n", coll)
	coll.DisplayShellAndOptions()
}
