// Froxy - HTTP over SSH proxy
//
// Copyright (C) 2019 and up by Alexander Pevzner (pzz@apevzner.com)
// See LICENSE for license terms and conditions
//
// System-dependent paths -- UNIX version
//
// +build darwin dragonfly freebsd linux nacl netbsd openbsd solaris

package sysdep

import (
	"os/user"
	"path/filepath"
	"strings"
)

var (
	userHomeDir string
)

//
// Initialize stuff
//
func init() {
	user, err := user.Current()
	if err != nil {
		panic(err.Error())
	}
	userHomeDir = user.HomeDir
}

//
// Get system configuration directory for the program
//
func SysConfDir(program string) string {
	return filepath.Join("/etc/", strings.ToLower(program))
}

//
// Get user home directory
//
func UserHomeDir() string {
	return userHomeDir
}

//
// Get user configuration directory for the program
//
func UserConfDir(program string) string {
	return filepath.Join(userHomeDir, "."+strings.ToLower(program))
}

//
// Get user desktop directory
//
func UserDesktopDir() string {
	return filepath.Join(userHomeDir, "Desktop")
}

//
// Get user startup (autostart) directory
//
func UserStartupDir() string {
	return filepath.Join(userHomeDir, ".config/autostart")
}

//
// Get user desktop file for the program
//
func UserDesktopFile(program, iconname string) string {
	return filepath.Join(UserDesktopDir(), strings.ToLower(program)+".desktop")
}

//
// Get user startup file for the program
//
func UserStartupFile(program, iconname string) string {
	return filepath.Join(UserStartupDir(), strings.ToLower(program)+".desktop")
}

//
// Get icon file extension
//
func IconExt() string {
	return "png"
}
