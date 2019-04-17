//
// Default configuration
//

package main

const (
	// ----- Built-in HTTP server configuration -----
	//
	// Name of this program
	//
	PROGRAM_NAME = "TProxy"

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

	// ----- Cookie names used by TProxy -----
	//
	// Last visited TProxy configuration page
	//
	COOKIE_LAST_VISITED_PAGE = "tproxy-lvp"
)
