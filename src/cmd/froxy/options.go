//
// Froxy command-line options
//

package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

//
// Froxy command
//
type OptCmd int

const (
	OptCmdNone OptCmd = iota
	OptCmdDebug
	OptCmdRunFg
	OptCmdHelp
	OptCmdInstall
	OptCmdKill
	OptCmdOpen
	OptCmdRunBg
	OptCmdUninstall
)

//
// Command flags
//
type OptFlags int

const (
	OptFlgNoRun OptFlags = 1 << iota
	OptFlgNoAutostart
	OptFlgNoShortcut
)

//
// Parsed options
//
type Options struct {
	Cmd   OptCmd   // Froxy command
	Flags OptFlags // Command flags
	Port  int      // TCP port
}

//
// Parse options
//
func (opt *Options) Parse(env *Env, args []string) error {
	// No arguments is a special case
	if len(args) == 0 {
		*opt = Options{Cmd: OptCmdHelp}
		return nil
	}

	// Setup parser
	flagset := flag.NewFlagSet("", flag.ContinueOnError)
	debug := flag.Bool("noautostart", false, "")
	fg := flag.Bool("fg", false, "")
	help := flag.Bool("h", false, "")
	install := flag.Bool("i", false, "")
	kill := flag.Bool("k", false, "")
	open := flag.Bool("open", false, "")
	run := flag.Bool("r", false, "")
	uninstall := flag.Bool("u", false, "")

	norun := flag.Bool("norun", false, "")
	noautostart := flag.Bool("noautostart", false, "")
	port := flag.Int("p", env.GetPort(), "")

	// Parse arguments
	err := flagset.Parse(args)
	if err != nil {
		return err
	}

	if flagset.NArg() != 0 {
		return fmt.Errorf("Unrecognized argument %q", flagset.Arg(0))
	}

	// Decode command
	commands := []struct {
		flg bool
		cmd OptCmd
	}{
		{*debug, OptCmdDebug},
		{*fg, OptCmdRunFg},
		{*help, OptCmdHelp},
		{*install, OptCmdInstall},
		{*kill, OptCmdKill},
		{*open, OptCmdOpen},
		{*run, OptCmdRunBg},
		{*uninstall, OptCmdUninstall},
	}

	var cmd OptCmd
	for _, c := range commands {
		if c.flg {
			if cmd != OptCmdNone {
				cmd = c.cmd
			} else {
				return errors.New("Multiple commands not allowed")
			}
		}
	}

	if cmd == OptCmdNone {
		return errors.New("Missed command")
	}

	// Decode flags
	flags := []struct {
		flg bool
		bit OptFlags
	}{
		{*norun, OptFlgNoRun},
		{*noautostart, OptFlgNoAutostart},
	}

	var bits OptFlags
	for _, f := range flags {
		if f.flg {
			bits |= f.bit
		}
	}

	// Check port
	if *port < 1 || *port > 0xffff {
		return fmt.Errorf("Port number %d out of range", *port)
	}

	// Pack result
	opt.Cmd = cmd
	opt.Flags = bits
	opt.Port = *port

	return nil
}

//
// Print full usage
//
func (opt *Options) Usage(env *Env, full bool) {
	const short_usage = `Usage: froxy command [options]
Common commands:
  -i [-p port] [-norun] [-noshortcut] [-noautostart]
	Install and start the ${PROG}
  -u
	Uninstall the ${PROG}
  -h
        Print all commands and options
`

	const full_usage = `Usage: froxy command [options]
Commands:
  -debug        Run ${PROG} in debug mode
  -fg           Run ${PROG} in foreground
  -h            Print help page
  -i            Install and start the ${PROG}
  -k            Kill running ${PROG}
  -open         Open ${PROG} configuration in browser window
  -r            Run ${PROG} in background
  -u            Uninstall the ${PROG}

Options:
  -noautostart  Don't add ${PROG} to autostart
  -norun        Don't run after installation
  -noshortcut   Don't create desktop shortcut
  -p port       TCP port (default ${PORT})

Advanced options:
  -fg           Run in foreground
`

	usage := short_usage
	if full {
		usage = full_usage
	}

	usage = os.Expand(usage, func(name string) string {
		switch name {
		case "prog":
			return strings.ToLower(PROGRAM_NAME)

		case "PROG":
			return PROGRAM_NAME

		case "PORT":
			return strconv.Itoa(env.GetPort())
		}

		return ""
	})

	print(usage)
}

//
// Print error message
//
func (opt *Options) Error(err error) {
	println(err.Error())
	println("Try " + strings.ToLower(PROGRAM_NAME) + " -h for more information.")
}
