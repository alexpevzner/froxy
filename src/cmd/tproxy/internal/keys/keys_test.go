//
// SSH keys test
//

package keys

import (
	"bytes"
	"testing"
)

func keysEqual(k1, k2 *Key) bool {
	j1, _ := k1.MarshalJSON()
	j2, _ := k2.MarshalJSON()
	return bytes.Equal(j1, j2)
}

func TestKeys(tst *testing.T) {
	for t := KeyRsa2048; t <= KeyEd25519; t++ {
		// Generate a key
		key := KeyGen(t)
		key.Comment = "key used for testing"

		println(key.AuthorizedKey())

		// Test key->json->key transformation
		json, err := key.MarshalJSON()
		if err != nil {
			tst.Fatalf("%s: key.MarshalJSON: %s", key.Type, err)
		}
		var key2 Key
		err = key2.UnmarshalJSON(json)
		if err != nil {
			tst.Fatalf("%s: key.UnmarshalJSON: %s", key.Type, err)
		}

		if !keysEqual(key, &key2) {
			tst.Fatalf("%s: keys not equal after JSOM marshal/unmarshal", key.Type)
		}

		// Test key->PEM->key tranformation
		pem := key.EncodePEM()
		key2 = Key{}
		err = key2.DecodePEM(pem)
		if err != nil {
			tst.Fatalf("%s: key.DecodePEM: %s", key.Type, err)
		}

		if !keysEqual(key, &key2) {
			tst.Fatalf("%s: keys not equal after PEM encode/decode", key.Type)
		}
	}
}
