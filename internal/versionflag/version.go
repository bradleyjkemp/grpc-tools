package versionflag

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"
)

func init() {
	version := "non-module"
	if info, ok := debug.ReadBuildInfo(); ok {
		version = info.Main.Version
	}

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s (%s):\n", os.Args[0], version)
		flag.PrintDefaults()
	}
}
