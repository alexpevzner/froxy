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
	Server *ServerParams `json:"server,omitempty"` // Server parameters
	Sites  []StateSite   `json:"sites,omitempty"`  // List of forwarded sites
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
// Site parameters
//
type StateSite struct {
	Host string `json:"host,omitempty"` // Host name
	Rec  bool   `json:"rec,omitempty"`  // Recursive (with subdomains)
}

//
// Load state
//
func (state *State) Load(file string) error {
	// Read the state file
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	// Parse the state
	var newstate State
	err = json.Unmarshal(data, &newstate)
	if err == nil {
		*state = newstate
	}

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
