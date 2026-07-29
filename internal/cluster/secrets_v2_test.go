package cluster

import (
	"bytes"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"golang.org/x/crypto/curve25519"
)

func testSecretStoreV2(t *testing.T) (*secretStoreV2, string) {
	t.Helper()
	directory := filepath.Join(t.TempDir(), clusterSecretsV2DirectoryName)
	store, err := openSecretStoreV2(directory)
	if err != nil {
		t.Fatalf("openSecretStoreV2() error = %v", err)
	}
	return store, directory
}

func testX25519KeypairV2(t *testing.T, seed byte) ([]byte, []byte) {
	t.Helper()
	privateKey := bytes.Repeat([]byte{seed}, 32)
	publicKey, err := curve25519.X25519(privateKey, curve25519.Basepoint)
	if err != nil {
		t.Fatal(err)
	}
	return privateKey, publicKey
}

func testHostCredentialV2(t *testing.T, seed byte) v2Credential {
	t.Helper()
	privateKey, publicKey := testX25519KeypairV2(t, seed)
	return v2Credential{
		ControllerPrivate: privateKey,
		ControllerPublic:  publicKey,
		TargetPublic:      bytes.Repeat([]byte{seed + 1}, 32),
	}
}

func testPairingCredentialV2(t *testing.T, seed byte) v2Credential {
	t.Helper()
	privateKey, publicKey := testX25519KeypairV2(t, seed)
	return v2Credential{
		ControllerPrivate: privateKey,
		ControllerPublic:  publicKey,
		TargetPublic:      bytes.Repeat([]byte{seed + 1}, 32),
		PairingKey:        bytes.Repeat([]byte{seed + 2}, 32),
	}
}

func TestSecretStoreV2RoundTripPermissionsAndIsolation(t *testing.T) {
	store, directory := testSecretStoreV2(t)
	privateKey, publicKey := testX25519KeypairV2(t, 1)
	identity := nodeIdentityV2{PrivateKey: privateKey, PublicKey: publicKey}
	if err := store.WriteNodeIdentity(identity); err != nil {
		t.Fatalf("WriteNodeIdentity() error = %v", err)
	}
	identity.PrivateKey[0] ^= 0xff
	readIdentity, err := store.ReadNodeIdentity()
	if err != nil {
		t.Fatalf("ReadNodeIdentity() error = %v", err)
	}
	if !bytes.Equal(readIdentity.PublicKey, publicKey) ||
		bytes.Equal(readIdentity.PrivateKey, identity.PrivateKey) {
		t.Fatal("node identity was not cloned or preserved")
	}

	hostID := strings.Repeat("a", 32)
	credential := testHostCredentialV2(t, 2)
	name, err := store.WriteHostCredential(hostID, credential)
	if err != nil {
		t.Fatalf("WriteHostCredential() error = %v", err)
	}
	readCredential, err := store.ReadCredential(name)
	if err != nil {
		t.Fatalf("ReadCredential() error = %v", err)
	}
	if !bytes.Equal(readCredential.ControllerPrivate, credential.ControllerPrivate) ||
		!bytes.Equal(readCredential.TargetPublic, credential.TargetPublic) {
		t.Fatal("credential round trip mismatch")
	}
	if runtime.GOOS != "windows" {
		for _, path := range []string{
			filepath.Join(directory, nodeIdentityV2FileName),
			filepath.Join(directory, name),
		} {
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if got := info.Mode().Perm(); got != 0o600 {
				t.Fatalf("%s permissions = %o, want 600", path, got)
			}
		}
		info, err := os.Stat(directory)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0o700 {
			t.Fatalf("secret directory permissions = %o, want 700", got)
		}
	}
}

func TestSecretStoreV2RejectsMismatchedKeypairsAndStrictJSON(t *testing.T) {
	store, directory := testSecretStoreV2(t)
	privateKey, publicKey := testX25519KeypairV2(t, 3)
	wrongPrivate, _ := testX25519KeypairV2(t, 4)
	if err := store.WriteNodeIdentity(nodeIdentityV2{
		PrivateKey: wrongPrivate,
		PublicKey:  publicKey,
	}); err == nil {
		t.Fatal("WriteNodeIdentity() accepted a mismatched keypair")
	}
	credential := testHostCredentialV2(t, 5)
	credential.ControllerPrivate = privateKey
	if _, err := store.WriteHostCredential(strings.Repeat("b", 32), credential); err == nil {
		t.Fatal("WriteHostCredential() accepted a mismatched keypair")
	}

	name := "host-" + strings.Repeat("c", 32) + ".v2key"
	content := `{"schemaVersion":2,"kind":"host_credential",` +
		`"controllerPrivate":"` + base64.RawURLEncoding.EncodeToString(privateKey) + `",` +
		`"controllerPublic":"` + base64.RawURLEncoding.EncodeToString(publicKey) + `",` +
		`"targetPublic":"` + base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{9}, 32)) + `",` +
		`"unexpected":true}`
	if err := os.WriteFile(filepath.Join(directory, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReadCredential(name); err == nil {
		t.Fatal("ReadCredential() accepted an unknown JSON field")
	}
	validContent, err := marshalSecretEnvelopeV2(secretEnvelopeV2{
		SchemaVersion:     2,
		Kind:              "host_credential",
		ControllerPrivate: base64.RawURLEncoding.EncodeToString(privateKey),
		ControllerPublic:  base64.RawURLEncoding.EncodeToString(publicKey),
		TargetPublic: base64.RawURLEncoding.EncodeToString(
			bytes.Repeat([]byte{9}, 32),
		),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(directory, name),
		append(validContent, []byte(`{}`)...),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReadCredential(name); err == nil {
		t.Fatal("ReadCredential() accepted multiple JSON values")
	}
	duplicateKind := bytes.Replace(
		validContent,
		[]byte(`"kind":"host_credential"`),
		[]byte(`"kind":"host_credential","kind":"host_credential"`),
		1,
	)
	if err := os.WriteFile(
		filepath.Join(directory, name),
		duplicateKind,
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReadCredential(name); err == nil {
		t.Fatal("ReadCredential() accepted a duplicate JSON field")
	}
}

func TestSecretStoreV2PairingKeyIsOnlyInSecretFile(t *testing.T) {
	store, directory := testSecretStoreV2(t)
	credential := testPairingCredentialV2(t, 7)
	name, err := store.WritePairingCredential(strings.Repeat("d", 16), credential)
	if err != nil {
		t.Fatalf("WritePairingCredential() error = %v", err)
	}
	content, err := os.ReadFile(filepath.Join(directory, name))
	if err != nil {
		t.Fatal(err)
	}
	encodedKey := base64.RawURLEncoding.EncodeToString(credential.PairingKey)
	if !bytes.Contains(content, []byte(encodedKey)) ||
		!bytes.Contains(content, []byte(`"pairingKey"`)) {
		t.Fatal("pairing key was not stored in the dedicated secret file")
	}
	read, err := store.ReadCredential(name)
	if err != nil || !bytes.Equal(read.PairingKey, credential.PairingKey) {
		t.Fatalf("ReadCredential() pairing key mismatch: %v", err)
	}
}

func TestSecretStoreV2RejectsUnsafePermissionsAndOversizedFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not expose POSIX file mode semantics")
	}
	store, directory := testSecretStoreV2(t)
	hostID := strings.Repeat("9", 32)
	name, err := store.WriteHostCredential(hostID, testHostCredentialV2(t, 6))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, name)
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReadCredential(name); err == nil {
		t.Fatal("ReadCredential() accepted unsafe permissions")
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, bytes.Repeat([]byte("x"), int(maxSecretV2Bytes+1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReadCredential(name); err == nil {
		t.Fatal("ReadCredential() accepted an oversized secret")
	}
}

func TestSecretStoreV2ConcurrentDuplicateWriteHasOneWinner(t *testing.T) {
	store, _ := testSecretStoreV2(t)
	credential := testHostCredentialV2(t, 8)
	hostID := strings.Repeat("e", 32)
	const workers = 16
	var wait sync.WaitGroup
	var successes atomic.Int32
	for index := 0; index < workers; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			if _, err := store.WriteHostCredential(hostID, credential); err == nil {
				successes.Add(1)
			}
		}()
	}
	wait.Wait()
	if got := successes.Load(); got != 1 {
		t.Fatalf("successful concurrent writes = %d, want 1", got)
	}
}

func TestSecretStoreV2AtomicWriteAndDeleteRollback(t *testing.T) {
	store, directory := testSecretStoreV2(t)
	credential := testHostCredentialV2(t, 9)
	hostID := strings.Repeat("f", 32)

	store.ops.syncDir = func(string) error { return errors.New("injected sync failure") }
	if _, err := store.WriteHostCredential(hostID, credential); err == nil {
		t.Fatal("WriteHostCredential() unexpectedly succeeded")
	}
	name := "host-" + hostID + ".v2key"
	if _, err := os.Stat(filepath.Join(directory, name)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed write left target behind: %v", err)
	}

	store.ops = defaultAtomicFileOpsV2()
	if _, err := store.WriteHostCredential(hostID, credential); err != nil {
		t.Fatal(err)
	}
	store.ops.syncDir = func(string) error { return errors.New("injected delete sync failure") }
	if err := store.Delete(name); err == nil {
		t.Fatal("Delete() unexpectedly succeeded")
	}
	if _, err := store.ReadCredential(name); err != nil {
		t.Fatalf("Delete() rollback did not restore credential: %v", err)
	}
}

func TestSecretStoreV2RecoversInterruptedDelete(t *testing.T) {
	store, directory := testSecretStoreV2(t)
	hostID := strings.Repeat("1", 32)
	name, err := store.WriteHostCredential(hostID, testHostCredentialV2(t, 10))
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(directory, name)
	tombstone := filepath.Join(directory, "."+name+".deleted")
	if err := os.Rename(target, tombstone); err != nil {
		t.Fatal(err)
	}
	recovered, err := openSecretStoreV2(directory)
	if err != nil {
		t.Fatalf("openSecretStoreV2() recovery error = %v", err)
	}
	if _, err := recovered.ReadCredential(name); err != nil {
		t.Fatalf("recovered credential error = %v", err)
	}
	if _, err := os.Stat(tombstone); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("delete tombstone still exists: %v", err)
	}
}

func TestSecretStoreV2RemoveOrphansAndRejectSymlink(t *testing.T) {
	store, directory := testSecretStoreV2(t)
	referencedName, err := store.WriteHostCredential(
		strings.Repeat("2", 32),
		testHostCredentialV2(t, 11),
	)
	if err != nil {
		t.Fatal(err)
	}
	orphanName, err := store.WriteHostCredential(
		strings.Repeat("3", 32),
		testHostCredentialV2(t, 12),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.RemoveOrphans(map[string]struct{}{referencedName: {}}); err != nil {
		t.Fatalf("RemoveOrphans() error = %v", err)
	}
	if _, err := store.ReadCredential(referencedName); err != nil {
		t.Fatalf("referenced credential removed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(directory, orphanName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("orphan credential still exists: %v", err)
	}

	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks may require elevated Windows privileges")
	}
	symlinkName := "host-" + strings.Repeat("4", 32) + ".v2key"
	if err := os.Symlink(
		filepath.Join(directory, referencedName),
		filepath.Join(directory, symlinkName),
	); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReadCredential(symlinkName); err == nil {
		t.Fatal("ReadCredential() accepted a symlink")
	}
}
