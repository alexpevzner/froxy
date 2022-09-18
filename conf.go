// Froxy - HTTP over SSH proxy
//
// Copyright (C) 2019 and up by Alexander Pevzner (pzz@apevzner.com)
// See LICENSE for license terms and conditions
//
// Default configuration

package main

import (
	"time"
)

const (
	// ----- Program parameters -----
	//
	// Name of this program
	//
	PROGRAM_NAME = "Froxy"

	//
	// Name as shown under desktop icon
	//
	PROGRAM_ICON_NAME = PROGRAM_NAME + " Proxy"

	// ----- TCP parameters -----
	//
	// TCP Keep-alive
	//
	TCP_KEEP_ALIVE = 20 * time.Second

	//
	// Enable TCP dual-stack (RFC 6555-compliant "Happy Eyeballs")
	//
	TCP_DUAL_STACK = true

	// ----- HTTP transport parameters -----
	//
	// Max number of idle connections accross all hoshs.
	//
	HTTP_MAX_IDLE_CONNS = 100

	//
	// Max amount of time an idle connection will remain idle
	// before closing
	//
	HTTP_IDLE_CONN_TIMEOUT = 90 * time.Second

	//
	// How long to wait for a server's first response headers after fully
	// writing the request headers if the request has an
	// "Expect: 100-continue" header.
	//
	HTTP_EXPECT_CONTINUE_TIMEOUT = 1 * time.Second

	// ----- Built-in HTTP server configuration -----
	//
	// TCP port to run server on
	//
	HTTP_SERVER_PORT = 8888

	// ----- SSH configuration -----
	//
	// Max connections per client session
	//
	SSH_MAX_CONN_PER_CLIENT = 10

	// ----- Logging configuration -----
	//
	// Max size of log file
	//
	LOG_MAX_FILE_SIZE = 100 * 1024

	//
	// Max count of backup log files
	//
	LOG_MAX_BACKUP_FILES = 3

	// ----- Cookie names used by Froxy -----
	//
	// Last visited Froxy configuration page
	//
	COOKIE_LAST_VISITED_PAGE = "froxy-lvp"
)
