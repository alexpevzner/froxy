//
// The main module
//

package main

import (
	"flag"
	"fmt"
	"os"
	"syscall"
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

	// Create environment
	env := NewEnv()

	// Create tproxy
	proxy, err := NewTproxy(env, *opt_port)
	if err != nil {
		env.Exit("%s", err)
	}

	// Detach stdin/stdout/stderr
	if *opt_detach {
		var in, out int
		in, err = syscall.Open(os.DevNull, syscall.O_RDONLY, 0644)
		if err != nil {
			env.Exit("Open %q: %s", os.DevNull, err)
		}

		out, err = syscall.Open(env.pathUserLogFile,
			syscall.O_CREAT|syscall.O_WRONLY|syscall.O_APPEND, 0644)

		if err != nil {
			env.Exit("Open %q: %s", env.pathUserLogFile, err)
		}

		if err != nil {
			env.Exit("%s", err)
		}

		syscall.Dup2(in, syscall.Stdin)
		syscall.Dup2(out, syscall.Stdout)
		syscall.Dup2(out, syscall.Stderr)

		syscall.Close(in)
		syscall.Close(out)
	}

	// Run tproxy
	proxy.Run()
}
