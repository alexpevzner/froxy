//
// Set of SSH keys
//

package main

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"cmd/tproxy/internal/keys"
)

//
// Set of keys with disk persistence
//
type KeySet struct {
	env     *Env                 // Back link to environment
	lock    sync.Mutex           // Access lock
	keys    map[string]*keys.Key // Key id->key map
	enabled map[string]struct{}  // Enabled keys, by id
}

//
// Key information structure, for WebAPI
//
type KeyInfo struct {
	Type     keys.KeyType `json:"type"`                // Key type
	FpSHA256 string       `json:"fp_sha256",omitempty` // SHA-256 fingerprint
	FpMD5    string       `json:"fp_md5",omitempty`    // MD5 fingerprint
	Comment  string       `json:"comment",omitempty`   // Key Comment
	Enabled  bool         `json:"enabled"`             // Key is enabled
}

//
// Create a KeySet
//
func NewKeySet(env *Env) *KeySet {
	set := &KeySet{
		env:     env,
		keys:    make(map[string]*keys.Key),
		enabled: make(map[string]struct{}),
	}

	set.load()
	return set
}

//
// Get keys
//
func (set *KeySet) GetKeys() []KeyInfo {
	set.lock.Lock()
	defer set.lock.Unlock()

	keys := []KeyInfo{}
	for _, key := range set.keys {
		info := KeyInfo{
			Type:     key.Type,
			FpSHA256: key.FingerprintSHA256(),
			FpMD5:    key.FingerprintMD5(),
			Comment:  key.Comment,
		}

		_, info.Enabled = set.enabled[info.FpSHA256]

		keys = append(keys, info)
	}

	return keys
}

//
// Modify the key
//
func (set *KeySet) KeyMod(id string, info *KeyInfo) error {
	// Acquire the lock
	set.lock.Lock()
	defer set.lock.Unlock()

	// Lookup the key
	key := set.keys[id]
	if key == nil {
		return ErrNoSuchKey
	}

	// See what changed and update in-memory copy
	updateKey := false
	updateEnabled := false

	if key.Comment != info.Comment {
		key.Comment = info.Comment
		updateKey = true
	}

	if _, enabled := set.enabled[id]; enabled != info.Enabled {
		if info.Enabled {
			set.enabled[id] = struct{}{}
		} else {
			delete(set.enabled, id)
		}
		updateEnabled = true
	}

	// Update on-disk copy
	if updateKey || updateEnabled {
		return set.updateKey(key, updateKey, updateEnabled, info.Enabled)
	}

	return set.deleteKey(key)
}

//
// Del key from KeySet. id is the key's
// SHA256 fingerpring
//
func (set *KeySet) KeyDel(id string) error {
	// Acquire the lock
	set.lock.Lock()
	defer set.lock.Unlock()

	// Lookup the key
	key := set.keys[id]
	if key == nil {
		return nil
	}

	// Delete the key
	delete(set.keys, id)
	delete(set.enabled, id)

	return set.deleteKey(key)
}

//
// Generate new key
//
func (set *KeySet) KeyGen(info *KeyInfo) (*KeyInfo, error) {
	// Generate a key
	key := keys.KeyGen(info.Type)
	key.Comment = info.Comment

	// Acquire the lock
	set.lock.Lock()
	defer set.lock.Unlock()

	// Save to disk
	err := set.updateKey(key, true, true, info.Enabled)
	if err != nil {
		return nil, err
	}

	// Save the key to memory
	id := key.FingerprintSHA256()
	set.keys[id] = key
	if info.Enabled {
		set.enabled[id] = struct{}{}
	}

	// Update info
	newinfo := *info
	newinfo.FpSHA256 = id
	newinfo.FpMD5 = key.FingerprintMD5()

	return &newinfo, nil
}

// ----- On-disk key storage -----
const (
	pathExtEnabled = "enabled"
)

//
// Get key's file name (without directory)
//
func (set *KeySet) fileName(key *keys.Key) string {
	return fmt.Sprintf("%x", key.BinFingerprintMD5())
}

//
// Get key's full path
//
func (set *KeySet) filePath(key *keys.Key) string {
	return filepath.Join(set.env.PathUserKeysDir, set.fileName(key))
}

//
// Check that file name is a valid key's filename
//
func (set *KeySet) checkName(name string) bool {
	if len(name) != md5.Size*2 {
		return false
	}

	for _, c := range []byte(name) {
		switch {
		case '0' <= c && c <= '9':
		case 'a' <= c && c <= 'f':
		case 'A' <= c && c <= 'A':
		default:
			return false
		}
	}

	return true
}

//
// Load all keys from disk
//
func (set *KeySet) load() {
	dir, err := ioutil.ReadDir(set.env.PathUserKeysDir)
	if err != nil {
		set.env.Warn("%s: %s", set.env.PathUserKeysDir, err)
		return
	}

	loadedKeys := make(map[string]*keys.Key)
	enabled := make(map[string]struct{})

	for _, file := range dir {
		// Skip all non-regular files
		if !file.Mode().IsRegular() {
			continue
		}

		// Obtain file name and extension
		name := file.Name()
		ext := filepath.Ext(name)
		if ext != "" {
			name = name[:len(name)-len(ext)]
		}

		if !set.checkName(name) {
			set.env.Warn("%s: %s", name, "invalid key name")
			continue
		}

		// Load all files
		switch ext {
		case "":
			path := filepath.Join(set.env.PathUserKeysDir, name)
			data, err := ioutil.ReadFile(path)
			key := &keys.Key{}
			if err == nil {
				err = key.DecodePEM(data)
			}
			if err == nil && name != set.fileName(key) {
				err = errors.New("file name doesn't match the key")
			}

			if err != nil {
				set.env.Warn("%s: %s", name, err)
				continue
			}

			loadedKeys[key.FingerprintSHA256()] = key

		case pathExtEnabled:
			enabled[name] = struct{}{}
		}

		// Update keyset
		set.keys = loadedKeys
		set.enabled = make(map[string]struct{})
		for _, key := range loadedKeys {
			if _, ok := enabled[set.fileName(key)]; ok {
				set.enabled[key.FingerprintSHA256()] = struct{}{}
			}
		}
	}
}

//
// Update key at disk
//
func (set *KeySet) updateKey(key *keys.Key, updateKey, updateEnabled, enabled bool) error {
	path := set.filePath(key)
	var err error

	if updateKey {
		data := key.EncodePEM()
		err = ioutil.WriteFile(path, data, 0600)
	}

	if err == nil && updateEnabled {
		path += "." + pathExtEnabled
		if enabled {
			err = ioutil.WriteFile(path, []byte{}, 0600)
		} else {
			os.Remove(path) // Ignore an error, if any
		}
	}

	return err
}

//
// Delete key from disk
//
func (set *KeySet) deleteKey(key *keys.Key) error {
	path := filepath.Join(set.env.PathUserKeysDir)
	os.Remove(path)
	os.Remove(path + ".use")
	return nil
}