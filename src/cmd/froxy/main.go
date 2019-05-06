//
// The main module
//

package main

import (
	"os"
)

//
// Froxy administration (install/uninstall etc)
//
func FroxyAdm(env *Env, opt *Options) {
	adm, err := NewAdm(env, opt.Port)
	if err != nil {
		adm.Fatal("%s", err)
	}

	switch opt.Cmd {
	case OptCmdInstall:
		err = adm.Install(opt.Flags)
	case OptCmdUninstall:
		err = adm.Uninstall()
	case OptCmdKill:
		err = adm.Kill()
	case OptCmdRunBg:
		err = adm.Run()
	case OptCmdOpen:
		err = adm.Open()
	default:
		panic("Internal error")
	}

	if err != nil {
		adm.Fatal("%s", err)
	}

	os.Exit(0)
}

func main() {
	// Create environment
	env := NewEnv()

	// Parse command-line options
	var opt Options
	err := opt.Parse(env)
	if err != nil {
		opt.Error(err)
	}

	// Perform administration actions, if required
	switch opt.Cmd {
	case OptCmdNone:
		opt.Usage(env, false)
	case OptCmdHelp:
		opt.Usage(env, true)
	case OptCmdRunFg, OptCmdDebug:
	default:
		FroxyAdm(env, &opt)
	}

	// Acquire froxy.lock
	err = env.FroxyLockAcquire()
	if err != nil {
		env.Fatal("%s", err)
	}

	// Create froxy
	proxy, err := NewFroxy(env, opt.Port)
	if err != nil {
		env.Fatal("%s", err)
	}

	// Detach stdin/stdout/stderr
	if opt.Cmd != OptCmdDebug {
		err = env.Detach()
		if err != nil {
			env.Fatal("%s", err)
		}
	}

	// Run froxy
	proxy.Run()
}
