package main

import (
	"github.com/go-ini/ini"
	"net/url"
	"strings"
	"time"
)

const (
	CONNECT_TIMEOUT = 10 * time.Second

	DEFAULT_CLIENT_CFG = "client.cfg"
	DEFAULT_SERVER_CFG = "server.cfg"

	SSH_MAX_CONN_PER_CLIENT = 10
)

//
// Client configuration
//
type CfgClient struct {
	Port     int      // TCP port
	Server   *url.URL // Server URL
	Login    string   // User login
	Password string   // User password
	Sites    []string // List of tunneled sites
}

//
// Load Client configuration
//
func LoadCfgClient(path string) (*CfgClient, error) {
	// Load INI file
	ini, err := ini.LoadSources(
		ini.LoadOptions{
			UnparseableSections: []string{"sites"},
		},
		path,
	)

	if err != nil {
		return nil, err
	}

	cfg := &CfgClient{}

	// Get http configuration
	s, err := ini.GetSection("http")
	if err != nil {
		return nil, err
	}

	k, err := s.GetKey("port")
	if err != nil {
		return nil, err
	}

	cfg.Port, err = k.Int()
	if err != nil {
		return nil, err
	}

	// Get server configuration
	s, err = ini.GetSection("server")
	if err != nil {
		return nil, err
	}

	k, err = s.GetKey("url")
	if err != nil {
		return nil, err
	}

	cfg.Server, err = url.Parse(k.String())
	if err != nil {
		return nil, err
	}

	k, err = s.GetKey("login")
	if err != nil {
		return nil, err
	}

	cfg.Login = k.String()

	k, err = s.GetKey("password")
	if err != nil {
		return nil, err
	}

	cfg.Password = k.String()

	// Get sites
	s, err = ini.GetSection("sites")
	if err != nil {
		return nil, err
	}

	cfg.Sites = strings.Fields(s.Body())

	return cfg, err
}
