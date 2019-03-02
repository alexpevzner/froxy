//
// The main module
//

package main

import (
	"flag"
	"fmt"
	"os"
	"tproxy/log"
)

//----- Program options -----
var (
	opt_cfg = flag.String("f", "", "Path to configuration file")
)

//
// Print usage and exit
//
func Usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Options are:\n")
	flag.PrintDefaults()
}

func main() {
	// Parse command-line options
	flag.Usage = Usage

	flag.Parse()
	if flag.NArg() != 0 {
		flag.Usage()
		os.Exit(2)
	}

	// Run tproxy
	proxy, err := NewTproxy(*opt_cfg)
	if err == nil {
		err = proxy.Run()
	}

	if err != nil {
		log.Exit("%s", err)
	}
}
