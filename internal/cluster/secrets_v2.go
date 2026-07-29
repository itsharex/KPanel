package cluster

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/crypto/curve25519"
)

const (
	clusterSecretsV2DirectoryName = "cluster-secrets-v2"
	nodeIdentityV2FileName        = "node-identity.v2key"
	maxSecretV2Bytes              = int64(4 << 10)
)

type nodeIdentityV2 struct {
	PrivateKey []byte
	PublicKey  []byte
}

type v2Credential struct {
	ControllerPrivate []byte
	ControllerPublic  []byte
	TargetPublic      []byte
	PairingKey        []byte
}

type secretEnvelopeV2 struct {
	SchemaVersion int    `json:"schemaVersion"`
	Kind          string `json:"kind"`

	PrivateKey string `json:"privateKey,omitempty"`
	PublicKey  string `json:"publicKey,omitempty"`

	ControllerPrivate string `json:"controllerPrivate,omitempty"`
	ControllerPublic  string `json:"controllerPublic,omitempty"`
	TargetPublic      string `json:"targetPublic,omitempty"`
	PairingKey        string `json:"pairingKey,omitempty"`
}

type secretStoreV2 struct {
	mu        sync.RWMutex
	directory string
	ops       atomicFileOpsV2
}

func openSecretStoreV2(directory string) (*secretStoreV2, error) {
	if strings.TrimSpace(directory) == "" || !filepath.IsAbs(directory) ||
		filepath.Base(filepath.Clean(directory)) != clusterSecretsV2DirectoryName {
		return nil, fmt.Errorf(
			"cluster v2 secret directory must be an absolute %s path",
			clusterSecretsV2DirectoryName,
		)
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, fmt.Errorf("create cluster v2 secret directory: %w", err)
	}
	if err := protectDirectoryV2(directory); err != nil {
		return nil, err
	}
	store := &secretStoreV2{
		directory: directory,
		ops:       defaultAtomicFileOpsV2(),
	}
	if err := store.recoverPendingDeletes(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *secretStoreV2) WriteNodeIdentity(identity nodeIdentityV2) error {
	if err := validateNodeIdentityV2(identity); err != nil {
		return err
	}
	content, err := marshalSecretEnvelopeV2(secretEnvelopeV2{
		SchemaVersion: 2,
		Kind:          "node_identity",
		PrivateKey:    encodeSecretBytesV2(identity.PrivateKey),
		PublicKey:     encodeSecretBytesV2(identity.PublicKey),
	})
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	target := filepath.Join(s.directory, nodeIdentityV2FileName)
	if err := atomicWriteFileV2(target, content, 0o600, false, s.ops); err != nil {
		if errors.Is(err, os.ErrExist) {
			return errors.New("cluster v2 node identity already exists")
		}
		return fmt.Errorf("write cluster v2 node identity: %w", err)
	}
	return nil
}

func (s *secretStoreV2) ReadNodeIdentity() (nodeIdentityV2, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	content, err := readRegularFileV2(
		filepath.Join(s.directory, nodeIdentityV2FileName),
		maxSecretV2Bytes,
		true,
	)
	if err != nil {
		return nodeIdentityV2{}, fmt.Errorf("read cluster v2 node identity: %w", err)
	}
	envelope, err := decodeSecretEnvelopeV2(content)
	if err != nil || envelope.Kind != "node_identity" ||
		envelope.ControllerPrivate != "" ||
		envelope.ControllerPublic != "" ||
		envelope.TargetPublic != "" ||
		envelope.PairingKey != "" {
		return nodeIdentityV2{}, errors.New("cluster v2 node identity is malformed")
	}
	identity := nodeIdentityV2{
		PrivateKey: decodeSecretBytesV2(envelope.PrivateKey),
		PublicKey:  decodeSecretBytesV2(envelope.PublicKey),
	}
	if err := validateNodeIdentityV2(identity); err != nil {
		return nodeIdentityV2{}, errors.New("cluster v2 node identity is malformed")
	}
	return cloneNodeIdentityV2(identity), nil
}

func (s *secretStoreV2) WriteHostCredential(
	hostID string,
	credential v2Credential,
) (string, error) {
	if !validID(hostID) {
		return "", errors.New("cluster v2 host credential ID is invalid")
	}
	if err := validateHostCredentialV2(credential); err != nil {
		return "", err
	}
	return s.writeCredential(
		"host-"+hostID+".v2key",
		"host_credential",
		credential,
	)
}

func (s *secretStoreV2) WritePairingCredential(
	codeID string,
	credential v2Credential,
) (string, error) {
	if len(codeID) != 16 || !validHex(codeID) {
		return "", errors.New("cluster v2 pairing credential ID is invalid")
	}
	if err := validatePairingCredentialV2(credential); err != nil {
		return "", err
	}
	return s.writeCredential(
		"pair-"+codeID+".v2key",
		"pairing_credential",
		credential,
	)
}

func (s *secretStoreV2) ReadCredential(name string) (v2Credential, error) {
	if !validCredentialNameV2(name) && !validPairingCredentialNameV2(name) {
		return v2Credential{}, errors.New("cluster v2 credential reference is invalid")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	content, err := readRegularFileV2(
		filepath.Join(s.directory, name),
		maxSecretV2Bytes,
		true,
	)
	if err != nil {
		return v2Credential{}, fmt.Errorf("read cluster v2 credential: %w", err)
	}
	envelope, err := decodeSecretEnvelopeV2(content)
	if err != nil || envelope.PrivateKey != "" || envelope.PublicKey != "" {
		return v2Credential{}, errors.New("cluster v2 credential is malformed")
	}
	credential := v2Credential{
		ControllerPrivate: decodeSecretBytesV2(envelope.ControllerPrivate),
		ControllerPublic:  decodeSecretBytesV2(envelope.ControllerPublic),
		TargetPublic:      decodeSecretBytesV2(envelope.TargetPublic),
		PairingKey:        decodeSecretBytesV2(envelope.PairingKey),
	}
	switch {
	case validCredentialNameV2(name) && envelope.Kind == "host_credential":
		err = validateHostCredentialV2(credential)
	case validPairingCredentialNameV2(name) && envelope.Kind == "pairing_credential":
		err = validatePairingCredentialV2(credential)
	default:
		err = errors.New("cluster v2 credential type is invalid")
	}
	if err != nil {
		return v2Credential{}, errors.New("cluster v2 credential is malformed")
	}
	return cloneCredentialV2(credential), nil
}

func (s *secretStoreV2) Delete(name string) error {
	if !validCredentialNameV2(name) && !validPairingCredentialNameV2(name) {
		return errors.New("cluster v2 credential reference is invalid")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.deleteLocked(name)
}

func (s *secretStoreV2) deleteLocked(name string) error {
	target := filepath.Join(s.directory, name)
	info, err := os.Lstat(target)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect cluster v2 credential: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("cluster v2 credential must be a regular file")
	}
	tombstone := filepath.Join(s.directory, "."+name+".deleted")
	if _, err := os.Lstat(tombstone); err == nil {
		return errors.New("cluster v2 credential deletion is already pending")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := s.ops.rename(target, tombstone); err != nil {
		return fmt.Errorf("stage cluster v2 credential deletion: %w", err)
	}
	if err := s.ops.syncDir(s.directory); err != nil {
		_ = s.ops.rename(tombstone, target)
		_ = syncDirectoryV2(s.directory)
		return fmt.Errorf("commit cluster v2 credential deletion: %w", err)
	}
	_ = s.ops.remove(tombstone)
	_ = s.ops.syncDir(s.directory)
	return nil
}

func (s *secretStoreV2) RemoveOrphans(referenced map[string]struct{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := os.ReadDir(s.directory)
	if err != nil {
		return fmt.Errorf("enumerate cluster v2 credentials: %w", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if name == nodeIdentityV2FileName ||
			(!validCredentialNameV2(name) && !validPairingCredentialNameV2(name)) {
			continue
		}
		if _, ok := referenced[name]; ok {
			continue
		}
		if err := s.deleteLocked(name); err != nil {
			return fmt.Errorf("remove orphan cluster v2 credential: %w", err)
		}
	}
	return nil
}

func (s *secretStoreV2) writeCredential(
	name string,
	kind string,
	credential v2Credential,
) (string, error) {
	content, err := marshalSecretEnvelopeV2(secretEnvelopeV2{
		SchemaVersion:     2,
		Kind:              kind,
		ControllerPrivate: encodeSecretBytesV2(credential.ControllerPrivate),
		ControllerPublic:  encodeSecretBytesV2(credential.ControllerPublic),
		TargetPublic:      encodeSecretBytesV2(credential.TargetPublic),
		PairingKey:        encodeSecretBytesV2(credential.PairingKey),
	})
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := atomicWriteFileV2(
		filepath.Join(s.directory, name),
		content,
		0o600,
		false,
		s.ops,
	); err != nil {
		if errors.Is(err, os.ErrExist) {
			return "", errors.New("cluster v2 credential already exists")
		}
		return "", fmt.Errorf("write cluster v2 credential: %w", err)
	}
	return name, nil
}

func (s *secretStoreV2) recoverPendingDeletes() error {
	entries, err := os.ReadDir(s.directory)
	if err != nil {
		return fmt.Errorf("enumerate cluster v2 secret recovery files: %w", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, ".") || !strings.HasSuffix(name, ".deleted") {
			continue
		}
		originalName := strings.TrimSuffix(strings.TrimPrefix(name, "."), ".deleted")
		if !validCredentialNameV2(originalName) &&
			!validPairingCredentialNameV2(originalName) {
			continue
		}
		tombstone := filepath.Join(s.directory, name)
		info, err := os.Lstat(tombstone)
		if err != nil || !info.Mode().IsRegular() ||
			info.Mode()&os.ModeSymlink != 0 {
			return errors.New("cluster v2 deletion recovery file is unsafe")
		}
		target := filepath.Join(s.directory, originalName)
		if _, err := os.Lstat(target); err == nil {
			if err := s.ops.remove(tombstone); err != nil {
				return fmt.Errorf("discard cluster v2 deletion recovery file: %w", err)
			}
		} else if errors.Is(err, os.ErrNotExist) {
			if err := s.ops.rename(tombstone, target); err != nil {
				return fmt.Errorf("restore cluster v2 deletion recovery file: %w", err)
			}
		} else {
			return err
		}
		if err := s.ops.syncDir(s.directory); err != nil {
			return fmt.Errorf("sync cluster v2 deletion recovery: %w", err)
		}
	}
	return nil
}

func validateNodeIdentityV2(identity nodeIdentityV2) error {
	if len(identity.PrivateKey) != 32 || len(identity.PublicKey) != 32 ||
		bytes.Equal(identity.PrivateKey, make([]byte, 32)) ||
		bytes.Equal(identity.PublicKey, make([]byte, 32)) {
		return errors.New("cluster v2 node identity is invalid")
	}
	derived, err := curve25519.X25519(identity.PrivateKey, curve25519.Basepoint)
	if err != nil || !bytes.Equal(derived, identity.PublicKey) {
		return errors.New("cluster v2 node identity keypair does not match")
	}
	return nil
}

func validateHostCredentialV2(credential v2Credential) error {
	if len(credential.ControllerPrivate) != 32 ||
		len(credential.ControllerPublic) != 32 ||
		len(credential.TargetPublic) != 32 ||
		len(credential.PairingKey) != 0 ||
		bytes.Equal(credential.ControllerPrivate, make([]byte, 32)) ||
		bytes.Equal(credential.ControllerPublic, make([]byte, 32)) ||
		bytes.Equal(credential.TargetPublic, make([]byte, 32)) {
		return errors.New("cluster v2 host credential is invalid")
	}
	derived, err := curve25519.X25519(
		credential.ControllerPrivate,
		curve25519.Basepoint,
	)
	if err != nil || !bytes.Equal(derived, credential.ControllerPublic) {
		return errors.New("cluster v2 host credential keypair does not match")
	}
	return nil
}

func validatePairingCredentialV2(credential v2Credential) error {
	controllerPrivateValid := len(credential.ControllerPrivate) == 0 ||
		len(credential.ControllerPrivate) == 32
	controllerPublicValid := len(credential.ControllerPublic) == 0 ||
		len(credential.ControllerPublic) == 32
	controllerPairMatches :=
		(len(credential.ControllerPrivate) == 0) ==
			(len(credential.ControllerPublic) == 0)
	if !controllerPrivateValid || !controllerPublicValid || !controllerPairMatches ||
		len(credential.TargetPublic) != 32 ||
		len(credential.PairingKey) != 32 ||
		bytes.Equal(credential.TargetPublic, make([]byte, 32)) ||
		bytes.Equal(credential.PairingKey, make([]byte, 32)) {
		return errors.New("cluster v2 pairing credential is invalid")
	}
	if len(credential.ControllerPrivate) == 32 &&
		(bytes.Equal(credential.ControllerPrivate, make([]byte, 32)) ||
			bytes.Equal(credential.ControllerPublic, make([]byte, 32))) {
		return errors.New("cluster v2 pairing credential is invalid")
	}
	if len(credential.ControllerPrivate) == 32 {
		derived, err := curve25519.X25519(
			credential.ControllerPrivate,
			curve25519.Basepoint,
		)
		if err != nil || !bytes.Equal(derived, credential.ControllerPublic) {
			return errors.New("cluster v2 pairing credential keypair does not match")
		}
	}
	return nil
}

func validCredentialNameV2(name string) bool {
	const prefix = "host-"
	const suffix = ".v2key"
	if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, suffix) ||
		filepath.Base(name) != name {
		return false
	}
	id := strings.TrimSuffix(strings.TrimPrefix(name, prefix), suffix)
	return validID(id)
}

func validPairingCredentialNameV2(name string) bool {
	const prefix = "pair-"
	const suffix = ".v2key"
	if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, suffix) ||
		filepath.Base(name) != name {
		return false
	}
	id := strings.TrimSuffix(strings.TrimPrefix(name, prefix), suffix)
	return len(id) == 16 && validHex(id)
}

func marshalSecretEnvelopeV2(envelope secretEnvelopeV2) ([]byte, error) {
	content, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("encode cluster v2 secret: %w", err)
	}
	content = append(content, '\n')
	if int64(len(content)) > maxSecretV2Bytes {
		return nil, errors.New("cluster v2 secret is too large")
	}
	return content, nil
}

func decodeSecretEnvelopeV2(content []byte) (secretEnvelopeV2, error) {
	var envelope secretEnvelopeV2
	if err := decodeStrictJSONV2(content, &envelope); err != nil {
		return secretEnvelopeV2{}, err
	}
	if envelope.SchemaVersion != 2 {
		return secretEnvelopeV2{}, errors.New("cluster v2 secret schema is unsupported")
	}
	return envelope, nil
}

func encodeSecretBytesV2(value []byte) string {
	if len(value) == 0 {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(value)
}

func decodeSecretBytesV2(value string) []byte {
	if value == "" {
		return nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil
	}
	return decoded
}

func cloneNodeIdentityV2(source nodeIdentityV2) nodeIdentityV2 {
	return nodeIdentityV2{
		PrivateKey: append([]byte(nil), source.PrivateKey...),
		PublicKey:  append([]byte(nil), source.PublicKey...),
	}
}

func cloneCredentialV2(source v2Credential) v2Credential {
	return v2Credential{
		ControllerPrivate: append([]byte(nil), source.ControllerPrivate...),
		ControllerPublic:  append([]byte(nil), source.ControllerPublic...),
		TargetPublic:      append([]byte(nil), source.TargetPublic...),
		PairingKey:        append([]byte(nil), source.PairingKey...),
	}
}
