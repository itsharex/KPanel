package cluster

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func testStoreV2(t *testing.T) (*storeV2, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), clusterStateV2FileName)
	store, err := openStoreV2(path)
	if err != nil {
		t.Fatalf("openStoreV2() error = %v", err)
	}
	if err := store.EnsureNodeID(strings.Repeat("a", 32)); err != nil {
		t.Fatalf("EnsureNodeID() error = %v", err)
	}
	return store, path
}

func testHostRecordV2(seed byte, now time.Time) hostRecordV2 {
	publicKey := bytes.Repeat([]byte{seed}, 32)
	identifier := fmt.Sprintf("%032x", seed)
	origin := fmt.Sprintf("https://node-%d.example", seed)
	record := hostRecordV2{
		ID:                 identifier,
		Name:               fmt.Sprintf("node-%d", seed),
		Origin:             origin,
		TransportSecurity:  v2TransportSecurity(origin),
		RemoteNodeID:       fmt.Sprintf("%032x", seed+50),
		ControllerID:       fmt.Sprintf("%032x", seed+100),
		State:              hostStateV2Active,
		TransactionID:      fmt.Sprintf("%032x", seed+150),
		CredentialFile:     "host-" + identifier + ".v2key",
		TargetPublicKey:    base64.RawURLEncoding.EncodeToString(publicKey),
		PeerFingerprint:    fingerprintV2(publicKey),
		FederationProtocol: FederationProtocolV2,
		CreatedAt:          now.UTC(),
		UpdatedAt:          now.UTC(),
	}
	record.ResourceVersion = hostResourceVersionV2(record)
	return record
}

func testPairingRecordV2(seed byte, now time.Time) pairingCodeRecordV2 {
	id := fmt.Sprintf("%016x", seed)
	return pairingCodeRecordV2{
		ID:             id,
		State:          pairingStateV2Issued,
		CredentialFile: "pair-" + id + ".v2key",
		ExpiresAt:      now.UTC().Add(5 * time.Minute),
	}
}

func testControllerRecordV2(seed byte, transactionID string, now time.Time) controllerRecordV2 {
	publicKey := bytes.Repeat([]byte{seed}, 32)
	return controllerRecordV2{
		ID:            fmt.Sprintf("%032x", seed),
		Name:          fmt.Sprintf("controller-%d", seed),
		PublicKey:     base64.RawURLEncoding.EncodeToString(publicKey),
		Fingerprint:   fingerprintV2(publicKey),
		Scope:         SummaryScope,
		State:         controllerStateV2Provisional,
		TransactionID: transactionID,
		CreatedAt:     now.UTC(),
		UpdatedAt:     now.UTC(),
	}
}

func TestStoreV2UsesDedicatedStrictPrivateState(t *testing.T) {
	store, path := testStoreV2(t)
	now := time.Now().UTC()
	if err := store.AddHost(testHostRecordV2(1, now)); err != nil {
		t.Fatalf("AddHost() error = %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != clusterStateV2FileName ||
		!bytes.Contains(content, []byte(`"schemaVersion": 2`)) {
		t.Fatalf("unexpected v2 state path or schema: %s\n%s", path, content)
	}
	if bytes.Contains(content, []byte(`"pairingKey"`)) ||
		bytes.Contains(content, []byte(`"controllerPrivate"`)) {
		t.Fatalf("secret material fields leaked into state JSON: %s", content)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Fatalf("state permissions = %o, want 600", got)
		}
	}

	if err := os.WriteFile(path, append(content, []byte(`{"extra":true}`)...), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := openStoreV2(path); err == nil ||
		!strings.Contains(err.Error(), "multiple JSON values") {
		t.Fatalf("multiple JSON values error = %v", err)
	}
}

func TestStoreV2RejectsUnknownCorruptAndOversizedState(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
	}{
		{
			name: "unknown field",
			content: []byte(
				`{"schemaVersion":2,"hosts":[],"controllers":[],"pairingCodes":[],"secret":"bad"}`,
			),
		},
		{
			name: "duplicate field",
			content: []byte(
				`{"schemaVersion":2,"schemaVersion":2,"hosts":[],"controllers":[],"pairingCodes":[]}`,
			),
		},
		{name: "truncated", content: []byte(`{"schemaVersion":2`)},
		{name: "empty", content: nil},
		{name: "oversized", content: bytes.Repeat([]byte("x"), int(maxClusterStoreV2Bytes+1))},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), clusterStateV2FileName)
			if err := os.WriteFile(path, test.content, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := openStoreV2(path); err == nil {
				t.Fatal("openStoreV2() unexpectedly accepted corrupt state")
			}
		})
	}
}

func TestStoreV2ConcurrentMutationAndResourceVersion(t *testing.T) {
	store, _ := testStoreV2(t)
	now := time.Now().UTC()
	const workers = 8
	var wait sync.WaitGroup
	errorsChannel := make(chan error, workers)
	for index := 0; index < workers; index++ {
		wait.Add(1)
		go func(seed byte) {
			defer wait.Done()
			errorsChannel <- store.AddHost(testHostRecordV2(seed, now))
		}(byte(index + 1))
	}
	wait.Wait()
	close(errorsChannel)
	for err := range errorsChannel {
		if err != nil {
			t.Fatalf("concurrent AddHost() error = %v", err)
		}
	}
	if got := len(store.Hosts()); got != workers {
		t.Fatalf("Hosts() count = %d, want %d", got, workers)
	}
	record, err := store.Host(fmt.Sprintf("%032x", byte(1)))
	if err != nil {
		t.Fatal(err)
	}
	stale := record.ResourceVersion
	record.Name = "renamed"
	record.UpdatedAt = now.Add(time.Second)
	updated, err := store.UpdateHost(record, stale)
	if err != nil {
		t.Fatalf("UpdateHost() error = %v", err)
	}
	record.UpdatedAt = now.Add(2 * time.Second)
	if _, err := store.UpdateHost(record, stale); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale UpdateHost() error = %v, want ErrConflict", err)
	}
	if updated.ResourceVersion == stale {
		t.Fatal("resource version did not change")
	}
}

func TestStoreV2PersistenceFailureRollsBackMemoryAndDisk(t *testing.T) {
	store, path := testStoreV2(t)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	store.ops.syncDir = func(string) error { return errors.New("injected directory sync failure") }
	if err := store.AddHost(testHostRecordV2(1, time.Now().UTC())); err == nil {
		t.Fatal("AddHost() unexpectedly succeeded")
	}
	if got := len(store.Hosts()); got != 0 {
		t.Fatalf("memory state contains %d hosts after rollback", got)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("disk state changed after rollback\nbefore=%s\nafter=%s", before, after)
	}
}

func TestStoreV2RenameFailureRestoresPreviousFile(t *testing.T) {
	store, path := testStoreV2(t)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	realRename := os.Rename
	call := 0
	store.ops.rename = func(oldPath, newPath string) error {
		call++
		if call == 2 {
			return errors.New("injected install rename failure")
		}
		return realRename(oldPath, newPath)
	}
	if err := store.AddHost(testHostRecordV2(1, time.Now().UTC())); err == nil {
		t.Fatal("AddHost() unexpectedly succeeded")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("previous file was not restored\nbefore=%s\nafter=%s", before, after)
	}
}

func TestStoreV2RecoversValidBackupAfterInterruptedReplacement(t *testing.T) {
	store, path := testStoreV2(t)
	if err := store.AddHost(testHostRecordV2(1, time.Now().UTC())); err != nil {
		t.Fatal(err)
	}
	valid, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".previous", valid, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"broken":`), 0o600); err != nil {
		t.Fatal(err)
	}
	recovered, err := openStoreV2(path)
	if err != nil {
		t.Fatalf("openStoreV2() recovery error = %v", err)
	}
	if got := len(recovered.Hosts()); got != 1 {
		t.Fatalf("recovered Hosts() count = %d, want 1", got)
	}
	if _, err := os.Stat(path + ".previous"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("backup still exists after recovery: %v", err)
	}
}

func TestStoreV2PairingLifecycleAndRevocationTombstone(t *testing.T) {
	store, _ := testStoreV2(t)
	now := time.Now().UTC()
	pairing := testPairingRecordV2(1, now)
	if err := store.AddPairingCode(pairing, now); err != nil {
		t.Fatal(err)
	}
	transactionID := strings.Repeat("c", 32)
	controller := testControllerRecordV2(2, transactionID, now)
	bound, err := store.BindPairingCode(pairing.ID, transactionID, controller, now)
	if err != nil {
		t.Fatalf("BindPairingCode() error = %v", err)
	}
	if bound.State != pairingStateV2Bound {
		t.Fatalf("bound state = %q", bound.State)
	}
	if !bound.ExpiresAt.After(pairing.ExpiresAt) {
		t.Fatalf(
			"bound transaction deadline = %s, want after one-time code expiry %s",
			bound.ExpiresAt,
			pairing.ExpiresAt,
		)
	}
	for attempt := 0; attempt < 5; attempt++ {
		if _, err := store.RecordPairingFailure(
			pairing.ID,
			now.Add(time.Duration(attempt+1)*time.Second),
		); !errors.Is(err, ErrPairingCode) {
			t.Fatalf("bound pairing failure attempt %d error = %v", attempt+1, err)
		}
	}
	if remaining, err := store.PairingCode(pairing.ID); err != nil ||
		remaining.State != pairingStateV2Bound {
		t.Fatalf("authenticated bound transaction was invalidated: %#v, %v", remaining, err)
	}
	credentialFile, err := store.CommitPairingCode(
		pairing.ID,
		transactionID,
		pairing.ExpiresAt.Add(time.Minute),
	)
	if err != nil {
		t.Fatalf("CommitPairingCode() error = %v", err)
	}
	if credentialFile != pairing.CredentialFile {
		t.Fatalf("committed credential = %q, want %q", credentialFile, pairing.CredentialFile)
	}
	active, err := store.Controller(controller.ID)
	if err != nil || active.State != controllerStateV2Active {
		t.Fatalf("active Controller() = %#v, %v", active, err)
	}
	retainUntil := now.Add(10 * time.Minute)
	revoked, err := store.RevokeController(
		controller.ID,
		strings.Repeat("d", 32),
		now.Add(time.Second),
		retainUntil,
	)
	if err != nil || revoked.State != controllerStateV2Revoked {
		t.Fatalf("RevokeController() = %#v, %v", revoked, err)
	}
	if got := len(store.Controllers()); got != 0 {
		t.Fatalf("Controllers() exposed %d revoked tombstones", got)
	}
	if _, err := store.Controller(controller.ID); err != nil {
		t.Fatalf("Controller() must retain revoked tombstone: %v", err)
	}
	if removed, err := store.GCRevokedControllers(retainUntil); err != nil || removed != 1 {
		t.Fatalf("GCRevokedControllers() = %d, %v", removed, err)
	}
}

func TestStoreV2PairingFailureLimitAndExpiryGC(t *testing.T) {
	store, _ := testStoreV2(t)
	now := time.Now().UTC()
	pairing := testPairingRecordV2(5, now)
	if err := store.AddPairingCode(pairing, now); err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 4; attempt++ {
		credential, err := store.RecordPairingFailure(pairing.ID, now)
		if !errors.Is(err, ErrPairingCode) || credential != "" {
			t.Fatalf(
				"RecordPairingFailure(%d) = %q, %v",
				attempt+1,
				credential,
				err,
			)
		}
	}
	if current, err := store.PairingCode(pairing.ID); err != nil || current.Attempts != 4 {
		t.Fatalf("PairingCode() after failures = %#v, %v", current, err)
	}
	credential, err := store.RecordPairingFailure(pairing.ID, now)
	if !errors.Is(err, ErrPairingCode) || credential != pairing.CredentialFile {
		t.Fatalf("fifth failure = %q, %v", credential, err)
	}
	if _, err := store.PairingCode(pairing.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("PairingCode() after fifth failure error = %v", err)
	}

	expiring := testPairingRecordV2(6, now)
	if err := store.AddPairingCode(expiring, now); err != nil {
		t.Fatal(err)
	}
	removed, err := store.GCExpiredPairingCodes(expiring.ExpiresAt)
	if err != nil || len(removed) != 1 || removed[0] != expiring.CredentialFile {
		t.Fatalf("GCExpiredPairingCodes() = %#v, %v", removed, err)
	}
}

func TestStoreV2StateNeverContainsPairingSecret(t *testing.T) {
	store, path := testStoreV2(t)
	now := time.Now().UTC()
	secretDirectory := filepath.Join(filepath.Dir(path), clusterSecretsV2DirectoryName)
	secrets, err := openSecretStoreV2(secretDirectory)
	if err != nil {
		t.Fatal(err)
	}
	pairingCredential := testPairingCredentialV2(t, 7)
	credentialFile, err := secrets.WritePairingCredential(
		fmt.Sprintf("%016x", byte(7)),
		pairingCredential,
	)
	if err != nil {
		t.Fatal(err)
	}
	record := testPairingRecordV2(7, now)
	record.CredentialFile = credentialFile
	if err := store.AddPairingCode(record, now); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	pairingKey := base64.RawURLEncoding.EncodeToString(pairingCredential.PairingKey)
	for _, forbidden := range []string{"pairingKey", pairingKey, "controllerPrivate", "privateKey"} {
		if bytes.Contains(content, []byte(forbidden)) {
			t.Fatalf("state JSON contains forbidden secret marker %q: %s", forbidden, content)
		}
	}
	secretContent, err := os.ReadFile(filepath.Join(secretDirectory, credentialFile))
	if err != nil || !bytes.Contains(secretContent, []byte(pairingKey)) {
		t.Fatalf("dedicated secret file does not contain the pairing key: %v", err)
	}
}
