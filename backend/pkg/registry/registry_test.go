package registry

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"testing"
	"time"
)

func TestSignVerifyAndValidate(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	ix := Index{GeneratedAt: time.Now(), Apps: []Manifest{{
		ID: "mn.test.inv", ShortID: "inv", Name: "Inv", Version: "1.2.3", GoModule: "example.com/inv",
		Permissions: []Permission{{Code: "inv.read", Name: "r"}}, Menus: []Menu{{ID: "inv.list", Label: "x", Path: "/inv"}},
	}}}
	raw, _ := json.Marshal(ix)
	sig := Sign(priv, raw)
	if err := Verify([]ed25519.PublicKey{pub}, raw, sig); err != nil {
		t.Fatal(err)
	}
	raw2 := append([]byte{}, raw...)
	raw2[len(raw2)-2] ^= 1
	if Verify([]ed25519.PublicKey{pub}, raw2, sig) == nil {
		t.Fatal("өөрчилсөн байтууд батлагдаж болохгүй")
	}
	if _, err := Parse(raw); err != nil {
		t.Fatal(err)
	}
	bad := ix
	bad.Apps[0].Permissions[0].Code = "other.read"
	if bad.Validate() == nil {
		t.Fatal("prefix зөрчил барих ёстой")
	}
}
