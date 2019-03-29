//
// The main module
//

package main

import (
	"flag"
	"fmt"
	"os"
)

//----- Program options -----
var (
	opt_port      = flag.Int("p", HTTP_SERVER_PORT, "Server port")
	opt_install   = flag.Bool("i", false, "Install and start the TProxy")
	opt_uninstall = flag.Bool("u", false, "Kill and uninstall the TProxy")
	opt_kill      = flag.Bool("k", false, "Kill running TProxy")
	opt_run       = flag.Bool("r", false, "Run TProxy in background")
	opt_detach    = flag.Bool("detach", false, "Close stdin/stdout/stderr after initialization")
	opt_open      = flag.Bool("open", false, "Open TProxy configuration in browser window")
	opt_used      = make(map[string]struct{})
)

//
// Print usage and exit
//
func Usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Options are:\n")
	flag.PrintDefaults()
}

//
// TProxy administration (install/uninstall etc)
//
func TproxyAdm(env *Env) {
	adm := Adm{Port: *opt_port, Env: env}
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
	case *opt_open:
		err = adm.Open()
	default:
		panic("Internal error")
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	os.Exit(0)
}

func main() {
	// Create environment
	env := NewEnv()

	// Parse command-line options
	flag.Usage = Usage
	flag.Lookup("p").DefValue = fmt.Sprintf("%d", env.GetPort())

	flag.Parse()
	if flag.NArg() != 0 {
		flag.Usage()
		os.Exit(2)
	}

	// Obtain list of actually used flags
	flag.Visit(func(f *flag.Flag) {
		opt_used[f.Name] = struct{}{}
	})

	// Validate arguments -- check administrative options
	admcnt := 0
	for _, f := range []bool{*opt_install, *opt_uninstall, *opt_kill, *opt_run, *opt_open} {
		if f {
			admcnt++
		}
	}

	if admcnt > 1 {
		fmt.Fprintf(os.Stderr, "Options -i, -u, -k, -r and -open are mutually exclusive\n")
		os.Exit(2)
	}

	// Check port
	if _, ok := opt_used["p"]; ok {
		if *opt_port < 1 || *opt_port > 0xffff {
			env.Exit("Port number %d out of range", *opt_port)
		}

		if *opt_uninstall || *opt_kill || *opt_open {
			env.Exit("Option -p is not compatible with -u, -k and -open")
		}
	} else {
		*opt_port = env.GetPort()
	}

	// Perform administration actions, if required
	if admcnt != 0 {
		TproxyAdm(env)
	}

	// Acquire tproxy.lock
	err := env.TproxyLockAcquire()
	if err != nil {
		env.Exit("%s", err)
	}

	// Create tproxy
	proxy, err := NewTproxy(env, *opt_port)
	if err != nil {
		env.Exit("%s", err)
	}

	// Detach stdin/stdout/stderr
	if *opt_detach {
		err = env.Detach()
		if err != nil {
			env.Exit("%s", err)
		}
	}

	// Run tproxy
	proxy.Run()
}
