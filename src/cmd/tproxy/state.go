//
// TProxy persistent state
//

package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
)

//
// The persistent state
//
type State struct {
	Server *ServerParams `json:"server"` // Server parameters
	Sites  []SiteParams  `json:"sites"`  // List of forwarded sites
}

//
// Server parameters
//
type ServerParams struct {
	Addr     string `json:"addr,omitempty"`     // Server address
	Login    string `json:"login,omitempty"`    // Server login
	Password string `json:"password,omitempty"` // Server password
}

//
// Check if server is configured
//
func (p *ServerParams) Configured() bool {
	return p.Addr != "" && p.Login != "" && p.Password != ""
}

//
// Site parameters
//
type SiteParams struct {
	Host  string `json:"host,omitempty"`  // Host name
	Rec   bool   `json:"rec,omitempty"`   // Recursive (with subdomains)
	Block bool   `json:"block,omitempty"` // Block the site
}

//
// Load state
//
func (state *State) Load(file string) error {
	// Reset the state
	state.Server = &ServerParams{}
	state.Sites = []SiteParams{}

	// Read the state file
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	// Parse the state
	err = json.Unmarshal(data, &state)

	return err
}

//
// Save state
//
func (state *State) Save(file string) error {
	// Allocate buffer
	buf := &bytes.Buffer{}

	// Setup JSON encoder
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "    ")

	// Encode into the buffer
	err := enc.Encode(state)
	if err != nil {
		panic(err) // Should never happen
	}

	// Write to file
	return ioutil.WriteFile(file, buf.Bytes(), 0600)
}
