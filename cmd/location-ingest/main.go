package main

import (
	"flag"
	"fmt"

	"codeberg.org/romanzipp/location-ingest/internal/buildinfo"
)

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(buildinfo.Version)
		return
	}

	fmt.Println("location-ingest", buildinfo.Version)
}
