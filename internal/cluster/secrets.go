package cluster

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type secretStore struct {
	directory string
	remove    func(string) error
}

func openSecretStore(directory string) (*secretStore, error) {
	if strings.TrimSpace(directory) == "" || !filepath.IsAbs(directory) {
		return nil, errors.New("cluster secret directory must be absolute")
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, fmt.Errorf("create cluster secret directory: %w", err)
	}
	info, err := os.Lstat(directory)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("cluster secret directory must be a real directory")
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return nil, fmt.Errorf("protect cluster secret directory: %w", err)
	}
	return &secretStore{directory: directory, remove: os.Remove}, nil
}

func (s *secretStore) Write(hostID string, key ed25519.PrivateKey) (string, error) {
	if !validID(hostID) || len(key) != ed25519.PrivateKeySize {
		return "", errors.New("cluster credential is invalid")
	}
	name := hostID + ".ed25519"
	target := filepath.Join(s.directory, name)
	if _, err := os.Lstat(target); err == nil {
		return "", errors.New("cluster credential already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("inspect cluster credential: %w", err)
	}
	temp, err := os.CreateTemp(s.directory, ".cluster-key-*")
	if err != nil {
		return "", fmt.Errorf("create cluster credential temporary file: %w", err)
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if err := temp.Chmod(0o600); err != nil {
		temp.Close()
		return "", err
	}
	encoded := base64.RawStdEncoding.EncodeToString(key) + "\n"
	if _, err := temp.WriteString(encoded); err != nil {
		temp.Close()
		return "", err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return "", err
	}
	if err := temp.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(tempName, target); err != nil {
		return "", fmt.Errorf("install cluster credential: %w", err)
	}
	return name, nil
}

func (s *secretStore) Read(name string) (ed25519.PrivateKey, error) {
	if !validCredentialName(name) {
		return nil, errors.New("cluster credential reference is invalid")
	}
	path := filepath.Join(s.directory, name)
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("inspect cluster credential: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
		return nil, errors.New("cluster credential permissions are unsafe")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read cluster credential: %w", err)
	}
	if len(content) > 256 {
		return nil, errors.New("cluster credential is too large")
	}
	decoded, err := base64.RawStdEncoding.DecodeString(strings.TrimSpace(string(content)))
	if err != nil || len(decoded) != ed25519.PrivateKeySize {
		return nil, errors.New("cluster credential is malformed")
	}
	return ed25519.PrivateKey(append([]byte(nil), decoded...)), nil
}

func (s *secretStore) Delete(name string) error {
	if !validCredentialName(name) {
		return errors.New("cluster credential reference is invalid")
	}
	err := s.remove(filepath.Join(s.directory, name))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func (s *secretStore) RemoveOrphans(referenced map[string]struct{}) error {
	entries, err := os.ReadDir(s.directory)
	if err != nil {
		return fmt.Errorf("enumerate cluster credentials: %w", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if !validCredentialName(name) {
			continue
		}
		if _, ok := referenced[name]; ok {
			continue
		}
		info, err := os.Lstat(filepath.Join(s.directory, name))
		if err != nil {
			return fmt.Errorf("inspect orphan cluster credential: %w", err)
		}
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("orphan cluster credential must be a regular file")
		}
		if err := s.Delete(name); err != nil {
			return fmt.Errorf("remove orphan cluster credential: %w", err)
		}
	}
	return nil
}

func validCredentialName(name string) bool {
	return len(name) == 32+len(".ed25519") && strings.HasSuffix(name, ".ed25519") &&
		validID(strings.TrimSuffix(name, ".ed25519")) &&
		filepath.Base(name) == name
}
