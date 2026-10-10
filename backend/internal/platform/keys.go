package platform

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	jose "github.com/go-jose/go-jose/v3"
	"os"
	"path/filepath"
	"sync"
)

type KeyRing struct {
	mu     sync.RWMutex
	path   string
	Active string            `json:"active"`
	Keys   []jose.JSONWebKey `json:"keys"`
	Secret []byte            `json:"secret"`
}

func loadKeys(dir string) (*KeyRing, error) {
	dir = filepath.Join(dir, "keys")
	if e := os.MkdirAll(dir, 0700); e != nil {
		return nil, e
	}
	k := &KeyRing{path: filepath.Join(dir, "identity.json")}
	b, e := os.ReadFile(k.path)
	if e == nil {
		if e = json.Unmarshal(b, k); e != nil {
			return nil, e
		}
		return k, nil
	}
	if !os.IsNotExist(e) {
		return nil, e
	}
	k.Secret = make([]byte, 32)
	if _, e = rand.Read(k.Secret); e != nil {
		return nil, e
	}
	if e = k.Rotate(); e != nil {
		return nil, e
	}
	return k, nil
}
func (k *KeyRing) Rotate() error {
	k.mu.Lock()
	defer k.mu.Unlock()
	private, e := rsa.GenerateKey(rand.Reader, 3072)
	if e != nil {
		return e
	}
	key := jose.JSONWebKey{Key: private, KeyID: id(), Algorithm: "RS256", Use: "sig"}
	for i := range k.Keys {
		k.Keys[i] = k.Keys[i].Public()
	}
	k.Keys = append(k.Keys, key)
	k.Active = key.KeyID
	b, e := json.MarshalIndent(k, "", "  ")
	if e != nil {
		return e
	}
	tmp := k.path + ".tmp"
	if e = os.WriteFile(tmp, b, 0600); e != nil {
		return e
	}
	return os.Rename(tmp, k.path)
}
func (k *KeyRing) CurrentID() string { k.mu.RLock(); defer k.mu.RUnlock(); return k.Active }
func (k *KeyRing) Private() any {
	k.mu.RLock()
	defer k.mu.RUnlock()
	for _, v := range k.Keys {
		if v.KeyID == k.Active {
			return v.Key
		}
	}
	return nil
}
func (k *KeyRing) Public() jose.JSONWebKeySet {
	k.mu.RLock()
	defer k.mu.RUnlock()
	out := jose.JSONWebKeySet{}
	for _, v := range k.Keys {
		out.Keys = append(out.Keys, v.Public())
	}
	return out
}
