//
// Froxy persistent state
//

package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"

	"github.com/alexpevzner/froxy/internal/sysdep"
)

//
// The persistent state
//
type State struct {
	Port   int          `json:"port"`   // TCP port Froxy runs on
	Server ServerParams `json:"server"` // Server parameters
	Sites  []SiteParams `json:"sites"`  // List of forwarded sites
}

//
// Server parameters
//
type ServerParams struct {
	Addr     string `json:"addr,omitempty"`     // Server address
	Login    string `json:"login,omitempty"`    // Server login
	Password string `json:"password,omitempty"` // Server password
	Keyid    string `json:"keyid,omitempty"`    // Key ID
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
	state.Server = ServerParams{}
	state.Sites = []SiteParams{}

	// Read the state file
	f, err := os.Open(file)
	if err != nil {
		return err
	}

	defer f.Close()

	err = sysdep.FileLock(f, false, true)
	if err != nil {
		return err
	}

	defer sysdep.FileUnlock(f)

	data, err := ioutil.ReadAll(f)
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
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}

	err = sysdep.FileLock(f, true, true)
	if err != nil {
		f.Close()
		return err
	}

	err = f.Truncate(0)
	if err == nil {
		n, err2 := f.Write(buf.Bytes())
		if err2 == nil && n < buf.Len() {
			err2 = io.ErrShortWrite
		}
		err = err2
	}

	sysdep.FileUnlock(f)

	if err2 := f.Close(); err == nil {
		err = err2
	}

	return err
}
