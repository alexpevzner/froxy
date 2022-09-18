//
// Set of SSH keys
//

package main

import (
	"crypto/md5"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/alexpevzner/froxy/internal/keys"
)

//
// Set of keys with disk persistence
//
type KeySet struct {
	env   *Env                 // Back link to environment
	lock  sync.Mutex           // Access lock
	keys  map[string]*keys.Key // Key id->Key
	infos map[string]*KeyInfo  // Key id->KeyInfo
}

//
// Create a KeySet
//
func NewKeySet(env *Env) *KeySet {
	set := &KeySet{
		env:   env,
		keys:  make(map[string]*keys.Key),
		infos: make(map[string]*KeyInfo),
	}

	set.load()
	return set
}

//
// Key information structure, for WebAPI
//
type KeyInfo struct {
	Id       string       `json:"id"`                  // Key id
	Type     keys.KeyType `json:"type"`                // Key type
	FpSHA256 string       `json:"fp_sha256,omitempty"` // SHA-256 fingerprint
	FpMD5    string       `json:"fp_md5,omitempty"`    // MD5 fingerprint
	Comment  string       `json:"comment,omitempty"`   // Key Comment
	Pubkey   string       `json:"pubkey,omitempty"`    // Public key
	Date     time.Time    `json:"date,omitempty"`      // Creation date
}

//
// Create KeyInfo from key
//
func NewKeyInfo(key *keys.Key) *KeyInfo {
	return &KeyInfo{
		Id:       key.Id(),
		Type:     key.Type,
		FpSHA256: key.FingerprintSHA256(),
		FpMD5:    key.FingerprintMD5(),
		Comment:  key.Comment,
		Pubkey:   key.AuthorizedKey(),
	}
}

//
// Get keys
//
func (set *KeySet) GetKeys() []KeyInfo {
	set.lock.Lock()
	defer set.lock.Unlock()

	infos := []KeyInfo{}
	for _, info := range set.infos {
		infos = append(infos, *info)
	}

	sort.Slice(infos, func(i, j int) bool {
		i1 := infos[i]
		i2 := infos[j]

		return i1.Date.Before(i2.Date) ||
			(i1.Date.Equal(i2.Date) && i1.Id < i2.Id)
	})

	return infos
}

//
// Get key by id
//
func (set *KeySet) KeyById(id string) *keys.Key {
	set.lock.Lock()
	defer set.lock.Unlock()

	return set.keys[id]
}

//
// Modify the key
//
func (set *KeySet) KeyMod(id string, newinfo *KeyInfo) error {
	// Acquire the lock
	set.lock.Lock()
	defer set.lock.Unlock()

	// Lookup the key
	key := set.keys[id]
	if key == nil {
		return ErrNoSuchKey
	}

	info := set.infos[id]
	if info == nil {
		panic("internal error")
	}

	// See what changed and update in-memory copy
	updateKey := false

	if key.Comment != newinfo.Comment {
		key.Comment = newinfo.Comment
		info.Comment = newinfo.Comment
		updateKey = true
	}

	// Update on-disk copy
	if updateKey {
		return set.updateKey(key, info)
	}

	return nil
}

//
// Del key from KeySet.
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
	delete(set.infos, id)

	return set.deleteKey(key)
}

//
// Generate new key
//
func (set *KeySet) KeyGen(info *KeyInfo) (*KeyInfo, error) {
	// Generate a key
	key := keys.KeyGen(info.Type)
	key.Comment = info.Comment

	// Update info
	info = NewKeyInfo(key)
	info.Date = time.Now()

	// Acquire the lock
	set.lock.Lock()
	defer set.lock.Unlock()

	// Save to disk
	err := set.updateKey(key, info)
	if err != nil {
		return nil, err
	}

	// Save the key to memory
	id := key.Id()
	set.keys[id] = key
	set.infos[id] = info

	return info, nil
}

// ----- On-disk key storage -----
//
// Get key's full path
//
func (set *KeySet) filePath(key *keys.Key) string {
	return filepath.Join(set.env.PathUserKeysDir, key.Id())
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
	// Acquire keys lock
	err := set.env.LockWait(EnvLockKeys)
	if err != nil {
		return
	}
	defer set.env.LockRelease(EnvLockKeys)

	// Read keys directory
	dir, err := ioutil.ReadDir(set.env.PathUserKeysDir)
	if err != nil {
		set.env.Warn("%s: %s", set.env.PathUserKeysDir, err)
		return
	}

	// Load all keys
	loadedKeys := make(map[string]*keys.Key)
	loadedInfos := make(map[string]*KeyInfo)

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
			ext = ext[1:]
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
			if err == nil && name != key.Id() {
				err = errors.New("file name doesn't match the key")
			}

			if err != nil {
				set.env.Warn("%s: %s", name, err)
				continue
			}

			loadedKeys[name] = key

			info := NewKeyInfo(key)
			info.Date = file.ModTime()
			loadedInfos[name] = info
		}
	}

	// Update keyset
	set.keys = loadedKeys
	set.infos = loadedInfos
}

//
// Update key at disk
//
func (set *KeySet) updateKey(key *keys.Key, info *KeyInfo) error {

	// Acquire keys lock
	err := set.env.LockWait(EnvLockKeys)
	if err != nil {
		return err
	}
	defer set.env.LockRelease(EnvLockKeys)

	// Update the key
	path := set.filePath(key)

	data := key.EncodePEM()
	err = ioutil.WriteFile(path, data, 0600)
	if err == nil {
		os.Chtimes(path, info.Date, info.Date)
	}

	return err
}

//
// Delete key from disk
//
func (set *KeySet) deleteKey(key *keys.Key) error {
	// Acquire keys lock
	err := set.env.LockWait(EnvLockKeys)
	if err != nil {
		return err
	}
	defer set.env.LockRelease(EnvLockKeys)

	// Delete the key
	path := set.filePath(key)
	os.Remove(path)

	return nil
}
