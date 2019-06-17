package versionflag

import (
	"flag"
	"fmt"
	"os"
)

var (
	version string
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s (%s):\n", os.Args[0], version)
		flag.PrintDefaults()
	}
}
