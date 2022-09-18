// Froxy - HTTP over SSH proxy
//
// Copyright (C) 2019 and up by Alexander Pevzner (pzz@apevzner.com)
// See LICENSE for license terms and conditions
//
// SSH Keys management

package keys

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strings"
	"unicode"

	"github.com/mikesmitty/edkey"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/ssh"
)

//
// Key type
//
type KeyType int

const (
	KeyRsa2048 = KeyType(iota)
	KeyRsa4096
	KeyEcdsa256
	KeyEcdsa384
	KeyEcdsa521
	KeyEd25519
)

//
// KeyType->string
//
func (t KeyType) String() string {
	switch t {
	case KeyRsa2048:
		return "rsa-2048"
	case KeyRsa4096:
		return "rsa-4096"
	case KeyEcdsa256:
		return "ecdsa-256"
	case KeyEcdsa384:
		return "ecdsa-384"
	case KeyEcdsa521:
		return "ecdsa-521"
	case KeyEd25519:
		return "ed25519"
	}

	return fmt.Sprintf("unknown(%d)", t)
}

//
// Marshal KeyType to JSON
//
func (t KeyType) MarshalJSON() ([]byte, error) {
	return []byte(`"` + t.String() + `"`), nil
}

//
// Unmarshal KeyType from JSON
//
func (t *KeyType) UnmarshalJSON(data []byte) error {
	switch s := string(data); s {
	case `"rsa-2048"`:
		*t = KeyRsa2048
	case `"rsa-4096"`:
		*t = KeyRsa4096
	case `"ecdsa-256"`:
		*t = KeyEcdsa256
	case `"ecdsa-384"`:
		*t = KeyEcdsa384
	case `"ecdsa-521"`:
		*t = KeyEcdsa521
	case `"ed25519"`:
		*t = KeyEd25519
	default:
		return fmt.Errorf("Invalid key type %s", s)
	}
	return nil
}

//
// Key structure
//
type Key struct {
	Type    KeyType     // Key type
	Comment string      // Key comment
	signer  ssh.Signer  // ssh.Signer precomputed
	priv    interface{} // Algo-specific private keys
}

//
// JSON representation of the key
//
type keyJson struct {
	Type    string `json:"type"`
	Priv    []byte `json:"priv,omitempty"`
	Comment string `json:"comment,omitempty"`
}

var _ = json.Marshaler(&Key{})
var _ = json.Unmarshaler(&Key{})

//
// Panic on error
//
func check(err error) {
	if err != nil {
		panic(err)
	}
}

//
// Generate a key
//
func KeyGen(t KeyType) *Key {
	var err error
	key := &Key{Type: t}

	switch t {
	case KeyRsa2048, KeyRsa4096:
		var bits int

		switch t {
		case KeyRsa2048:
			bits = 2048
		case KeyRsa4096:
			bits = 4096
		default:
			panic("internal error")
		}

		key.priv, err = rsa.GenerateKey(rand.Reader, bits)
		check(err)

	case KeyEcdsa256, KeyEcdsa384, KeyEcdsa521:
		var c elliptic.Curve
		switch t {
		case KeyEcdsa256:
			c = elliptic.P256()
		case KeyEcdsa384:
			c = elliptic.P384()
		case KeyEcdsa521:
			c = elliptic.P521()
		default:
			panic("internal error")
		}

		key.priv, err = ecdsa.GenerateKey(c, rand.Reader)
		check(err)

	case KeyEd25519:
		_, key.priv, err = ed25519.GenerateKey(rand.Reader)
		check(err)

	default:
		panic("internal error")
	}

	key.signer, err = ssh.NewSignerFromKey(key.priv)
	check(err)

	return key
}

//
// Marshal key to JSON
//
func (key *Key) MarshalJSON() ([]byte, error) {
	jsn := keyJson{Comment: key.Comment}

	switch key.Type {
	case KeyRsa2048, KeyRsa4096:
		jsn.Type = "ssh-rsa"
		jsn.Priv = x509.MarshalPKCS1PrivateKey(key.priv.(*rsa.PrivateKey))

	case KeyEcdsa256, KeyEcdsa384, KeyEcdsa521:
		jsn.Type = "ssh-ecdsa"
		var err error
		jsn.Priv, err = x509.MarshalECPrivateKey(key.priv.(*ecdsa.PrivateKey))
		check(err)

	case KeyEd25519:
		jsn.Type = "ssh-ed25519"
		jsn.Priv = []byte(key.priv.(ed25519.PrivateKey))
	}

	return json.Marshal(jsn)
}

//
// Unmarshal key from JSON
//
func (key *Key) UnmarshalJSON(data []byte) error {
	var jsn keyJson

	err := json.Unmarshal(data, &jsn)
	if err != nil {
		return err
	}

	switch jsn.Type {
	case "ssh-rsa":
		priv, err := x509.ParsePKCS1PrivateKey(jsn.Priv)
		if err != nil {
			return err
		}

		t, err := keyTypeRSA(priv)
		if err != nil {
			return err
		}

		key.Type = t
		key.priv = priv
		key.Comment = jsn.Comment

	case "ssh-ecdsa":
		priv, err := x509.ParseECPrivateKey(jsn.Priv)
		if err != nil {
			return err
		}

		t, err := keyTypeECDSA(priv)
		if err != nil {
			return err
		}

		key.Type = t
		key.priv = priv
		key.Comment = jsn.Comment

	case "ssh-ed25519":
		key.Type = KeyEd25519
		key.priv = ed25519.PrivateKey(jsn.Priv)
		key.Comment = jsn.Comment

	default:
		return fmt.Errorf("unknown key type %q", jsn.Type)
	}

	key.signer, err = ssh.NewSignerFromKey(key.priv)
	check(err)

	return nil
}

//
// Encode key into PEM format
//
func (key *Key) EncodePEM() []byte {
	var blk pem.Block

	if key.Comment != "" {
		blk.Headers = make(map[string]string)
		blk.Headers["Comment"] = key.Comment
	}

	switch key.Type {
	case KeyRsa2048, KeyRsa4096:
		blk.Type = "RSA PRIVATE KEY"
		blk.Bytes = x509.MarshalPKCS1PrivateKey(key.priv.(*rsa.PrivateKey))

	case KeyEcdsa256, KeyEcdsa384, KeyEcdsa521:
		var err error
		blk.Type = "EC PRIVATE KEY"
		blk.Bytes, err = x509.MarshalECPrivateKey(key.priv.(*ecdsa.PrivateKey))
		check(err)

	case KeyEd25519:
		blk.Type = "OPENSSH PRIVATE KEY"
		blk.Bytes = edkey.MarshalED25519PrivateKey(key.priv.(ed25519.PrivateKey))

	default:
		panic("internal error")
	}

	return pem.EncodeToMemory(&blk)
}

//
// Decode key from PEM format
//
func (key *Key) DecodePEM(data []byte) error {
	priv, err := ssh.ParseRawPrivateKey(data)
	if err != nil {
		return err
	}

	var comment string
	blk, _ := pem.Decode(data)
	if blk != nil {
		comment = blk.Headers["Comment"]
	}

	var t KeyType

	switch p := priv.(type) {
	case *rsa.PrivateKey:
		t, err = keyTypeRSA(p)
	case *ecdsa.PrivateKey:
		t, err = keyTypeECDSA(p)
	case *ed25519.PrivateKey:
		t = KeyEd25519
		priv = *p
	default:
		err = fmt.Errorf("PEM: unsupported key type")
	}

	if err != nil {
		return err
	}

	key.Type = t
	key.Comment = comment
	key.priv = priv
	key.signer, err = ssh.NewSignerFromKey(key.priv)
	check(err)

	return nil
}

//
// Obtain ssh.Signer
//
func (key *Key) Signer() ssh.Signer {
	return key.signer
}

//
// Get key id
//
func (key *Key) Id() string {
	return fmt.Sprintf("%x", key.BinFingerprintMD5())
}

//
// Generate SHA256 fingerprint in OpenSSH text format
//
func (key *Key) FingerprintSHA256() string {
	return ssh.FingerprintSHA256(key.signer.PublicKey())
}

//
// Generate MD5 fingerprin in OpenSSH text formatt
//
func (key *Key) FingerprintMD5() string {
	return ssh.FingerprintLegacyMD5(key.signer.PublicKey())
}

//
// Generate SHA256 binary fingerprint
//
func (key *Key) BinFingerprintSHA256() []byte {
	sum := sha256.Sum256(key.signer.PublicKey().Marshal())
	return sum[:]
}

//
// Generate MD5 binary fingerprint
//
func (key *Key) BinFingerprintMD5() []byte {
	sum := md5.Sum(key.signer.PublicKey().Marshal())
	return sum[:]
}

//
// Generate public key string for inclusion into authorized_keys file
//
func (key *Key) AuthorizedKey() string {
	s := string(ssh.MarshalAuthorizedKey(key.signer.PublicKey()))
	comment := strings.TrimFunc(key.Comment, unicode.IsSpace)
	if comment != "" {
		s = strings.TrimRight(s, "\n")
		s += " " + comment + "\n"
	}
	return s
}

//
// Decode RSA key type
//
func keyTypeRSA(priv *rsa.PrivateKey) (KeyType, error) {
	switch bs := priv.N.BitLen(); bs {
	case 2048:
		return KeyRsa2048, nil
	case 4096:
		return KeyRsa4096, nil
	default:
		return 0, fmt.Errorf("Unsupported RSA key size %d", bs)
	}
}

//
// Decode ECDSA key type
//
func keyTypeECDSA(priv *ecdsa.PrivateKey) (KeyType, error) {
	switch bs := priv.Curve.Params().BitSize; bs {
	case 256:
		return KeyEcdsa256, nil
	case 384:
		return KeyEcdsa384, nil
	case 521:
		return KeyEcdsa521, nil
	default:
		return 0, fmt.Errorf("Unsupported ECDSA key size %d", bs)
	}
}
