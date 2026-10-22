// Package credentials stores the master key outside PocketBase's database/data
// directory. Back up the key together with the associated data directory.
package credentials

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	d "podux/internal/domain/tunnel"
	"strings"
)

type Vault struct{ aead cipher.AEAD }

// Open never replaces an existing key. encrypted is true if ANY stored Token exists.
func Open(dataDir, keyPath string, encrypted bool) (*Vault, error) {
	data, e := filepath.Abs(dataDir)
	if e != nil {
		return nil, d.ErrCredentials
	}
	if keyPath == "" {
		keyPath = data + ".cloudflared-secrets/master.key"
	}
	keyPath, e = filepath.Abs(keyPath)
	if e != nil {
		return nil, d.ErrCredentials
	}
	// Resolve existing ancestors to reject paths that escape through symlinks.
	canonical := func(p string) string {
		tail := []string{}
		for {
			v, e := filepath.EvalSymlinks(p)
			if e == nil {
				for i := len(tail) - 1; i >= 0; i-- {
					v = filepath.Join(v, tail[i])
				}
				return v
			}
			parent := filepath.Dir(p)
			if parent == p {
				return p
			}
			tail = append(tail, filepath.Base(p))
			p = parent
		}
	}
	data = canonical(data)
	keyPath = canonical(keyPath)
	rel, e := filepath.Rel(data, keyPath)
	if e != nil || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))) {
		return nil, d.ErrCredentials
	}
	// Reject special files and insecure existing keys before opening/reading.
	info, statErr := os.Lstat(keyPath)
	if statErr == nil && (!info.Mode().IsRegular() || info.Size() != 32 || checkPermissions(keyPath, info) != nil) {
		return nil, d.ErrCredentials
	}
	if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return nil, d.ErrCredentials
	}
	key, e := os.ReadFile(keyPath)
	if errors.Is(e, os.ErrNotExist) {
		if encrypted {
			return nil, d.ErrCredentials
		}
		if e = os.MkdirAll(filepath.Dir(keyPath), 0700); e != nil {
			return nil, d.ErrCredentials
		}
		key = make([]byte, 32)
		if _, e = rand.Read(key); e != nil {
			return nil, d.ErrCredentials
		}
		f, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return nil, d.ErrCredentials
		}
		if e = protectFile(keyPath); e != nil {
			f.Close()
			return nil, d.ErrCredentials
		}
		_, e = f.Write(key)
		if e == nil {
			e = f.Sync()
		}
		ce := f.Close()
		if e != nil || ce != nil {
			return nil, d.ErrCredentials
		}
	} else if e != nil {
		return nil, d.ErrCredentials
	}
	info, e = os.Lstat(keyPath)
	if e != nil || !info.Mode().IsRegular() || len(key) != 32 {
		return nil, d.ErrCredentials
	}
	if e = checkPermissions(keyPath, info); e != nil {
		return nil, d.ErrCredentials
	}
	b, e := aes.NewCipher(key)
	clear(key)
	if e != nil {
		return nil, d.ErrCredentials
	}
	a, e := cipher.NewGCM(b)
	if e != nil {
		return nil, d.ErrCredentials
	}
	return &Vault{a}, nil
}
func (v *Vault) Encrypt(id, token string) (string, error) {
	nonce := make([]byte, v.aead.NonceSize())
	if _, e := rand.Read(nonce); e != nil {
		return "", d.ErrCredentials
	}
	b := v.aead.Seal(nonce, nonce, []byte(token), []byte("podux/tunnel/v1/"+id))
	return "v1:" + base64.RawStdEncoding.EncodeToString(b), nil
}
func (v *Vault) Decrypt(id, value string) (string, error) {
	if !strings.HasPrefix(value, "v1:") {
		return "", d.ErrCredentials
	}
	b, e := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(value, "v1:"))
	n := v.aead.NonceSize()
	if e != nil || len(b) < n {
		return "", d.ErrCredentials
	}
	p, e := v.aead.Open(nil, b[:n], b[n:], []byte("podux/tunnel/v1/"+id))
	if e != nil {
		return "", d.ErrCredentials
	}
	return string(p), nil
}
