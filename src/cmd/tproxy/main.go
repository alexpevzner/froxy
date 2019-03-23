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
	opt_port      = flag.Int("p", HTTP_SERVER_PORT, "Server port")
	opt_install   = flag.Bool("i", false, "Install and start the TProxy")
	opt_uninstall = flag.Bool("u", false, "Kill and uninstall the TProxy")
	opt_kill      = flag.Bool("k", false, "Kill running TProxy")
	opt_run       = flag.Bool("r", false, "Run TProxy in background")
	opt_detach    = flag.Bool("detach", false, "Close stdin/stdout/stderr after initialization")
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

	// Perform administration actions, if required
	admcnt := 0
	for _, f := range []bool{*opt_install, *opt_uninstall, *opt_kill, *opt_run} {
		if f {
			admcnt++
		}
	}

	if admcnt > 1 {
		fmt.Fprintf(os.Stderr, "Options -i, -u, -k and -r are mutually exclusive\n")
		os.Exit(2)
	}

	if admcnt != 0 {
		adm := Adm{Port: *opt_port}
		var err error
		switch {
		case *opt_install:
			err = adm.Install()
		case *opt_uninstall:
			err = adm.Uninstall()
		case *opt_kill:
			err = adm.Kill()
		case *opt_run:
			err = adm.Run()
		default:
			panic("Internal error")
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}

		os.Exit(0)
	}

	// Run tproxy
	proxy, err := NewTproxy(*opt_port)
	if err != nil {
		log.Exit("%s", err)
	}

	if *opt_detach {
		// FIXME -- it is temporary, buggy, UNIX-only solution
		os.Stdin.Close()
		os.Stdout.Close()
		os.Stderr.Close()

		os.Stdin, _ = os.Open("/dev/null")
		os.Stdout, _ = os.OpenFile("log", os.O_RDWR|os.O_CREATE, 0644)
		os.Stderr, _ = os.OpenFile("log", os.O_RDWR|os.O_CREATE, 0644)
	}

	proxy.Run()
}
