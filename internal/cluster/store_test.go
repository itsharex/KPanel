package cluster

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func openTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cluster-state.json")
	store, err := OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore() error: %v", err)
	}
	return store, path
}

func testController(t *testing.T, idByte byte, now time.Time) controllerRecord {
	t.Helper()
	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error: %v", err)
	}
	return controllerRecord{
		ID:          strings.Repeat(string(idByte), 32),
		Name:        "test controller",
		PublicKey:   encodePublicKey(publicKey),
		Fingerprint: fingerprint(publicKey),
		Scope:       SummaryScope,
		CreatedAt:   now.UTC(),
	}
}

func invalidPairingCodeWithSameID(t *testing.T, code string) string {
	t.Helper()
	id, _, ok := strings.Cut(code, ".")
	if !ok {
		t.Fatalf("pairing code %q has no separator", code)
	}
	return id + "." + strings.Repeat("0", 64)
}

func TestPairingCodeExpiryFailureLimitAndSingleUse(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 29, 10, 0, 0, 0, time.UTC)

	t.Run("four failures still permit correct single use", func(t *testing.T) {
		t.Parallel()
		store, path := openTestStore(t)
		code, err := store.CreatePairingCode(now)
		if err != nil {
			t.Fatalf("CreatePairingCode() error: %v", err)
		}
		if code.Scope != SummaryScope || !code.ExpiresAt.Equal(now.Add(5*time.Minute)) {
			t.Fatalf("unexpected pairing code metadata: %+v", code)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile() error: %v", err)
		}
		_, secret, _ := strings.Cut(code.Code, ".")
		if strings.Contains(string(content), code.Code) || strings.Contains(string(content), secret) {
			t.Fatal("raw pairing secret was persisted in cluster JSON")
		}

		wrong := invalidPairingCodeWithSameID(t, code.Code)
		for attempt := 0; attempt < 4; attempt++ {
			if err := store.ConsumePairingCode(wrong, testController(t, byte('a'+attempt), now), now); !errors.Is(err, ErrPairingCode) {
				t.Fatalf("wrong attempt %d error = %v, want ErrPairingCode", attempt+1, err)
			}
		}
		controller := testController(t, 'e', now)
		if err := store.ConsumePairingCode(code.Code, controller, now); err != nil {
			t.Fatalf("correct ConsumePairingCode() error after four failures: %v", err)
		}
		if _, err := store.Controller(controller.ID); err != nil {
			t.Fatalf("paired controller not stored: %v", err)
		}
		if err := store.ConsumePairingCode(code.Code, testController(t, 'f', now), now); !errors.Is(err, ErrPairingCode) {
			t.Fatalf("second consumption error = %v, want ErrPairingCode", err)
		}
	})

	t.Run("fifth failure invalidates code", func(t *testing.T) {
		t.Parallel()
		store, _ := openTestStore(t)
		code, err := store.CreatePairingCode(now)
		if err != nil {
			t.Fatalf("CreatePairingCode() error: %v", err)
		}
		wrong := invalidPairingCodeWithSameID(t, code.Code)
		for attempt := 0; attempt < 5; attempt++ {
			if err := store.ConsumePairingCode(wrong, testController(t, byte('a'+attempt), now), now); !errors.Is(err, ErrPairingCode) {
				t.Fatalf("wrong attempt %d error = %v, want ErrPairingCode", attempt+1, err)
			}
		}
		if err := store.ConsumePairingCode(code.Code, testController(t, 'f', now), now); !errors.Is(err, ErrPairingCode) {
			t.Fatalf("correct code after five failures error = %v, want ErrPairingCode", err)
		}
	})

	t.Run("expires at exact deadline", func(t *testing.T) {
		t.Parallel()
		store, _ := openTestStore(t)
		code, err := store.CreatePairingCode(now)
		if err != nil {
			t.Fatalf("CreatePairingCode() error: %v", err)
		}
		if err := store.ConsumePairingCode(code.Code, testController(t, 'a', now), code.ExpiresAt); !errors.Is(err, ErrPairingCode) {
			t.Fatalf("expired code error = %v, want ErrPairingCode", err)
		}
	})
}

func TestPairingCodeConcurrentConsumptionHasSingleWinner(t *testing.T) {
	t.Parallel()

	store, _ := openTestStore(t)
	now := time.Date(2026, 7, 29, 10, 0, 0, 0, time.UTC)
	code, err := store.CreatePairingCode(now)
	if err != nil {
		t.Fatalf("CreatePairingCode() error: %v", err)
	}
	controller := testController(t, 'a', now)

	var success atomic.Int32
	var unexpected atomic.Int32
	var wg sync.WaitGroup
	for index := 0; index < 16; index++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := store.ConsumePairingCode(code.Code, controller, now)
			switch {
			case err == nil:
				success.Add(1)
			case errors.Is(err, ErrPairingCode):
			default:
				unexpected.Add(1)
			}
		}()
	}
	wg.Wait()
	if success.Load() != 1 || unexpected.Load() != 0 {
		t.Fatalf("concurrent consumption success=%d unexpected=%d, want 1 and 0", success.Load(), unexpected.Load())
	}
}

func TestPrivateKeyIsKeptOutOfClusterJSON(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	storePath := filepath.Join(directory, "cluster-state.json")
	store, err := OpenStore(storePath)
	if err != nil {
		t.Fatalf("OpenStore() error: %v", err)
	}
	secrets, err := openSecretStore(filepath.Join(directory, "cluster-secrets"))
	if err != nil {
		t.Fatalf("openSecretStore() error: %v", err)
	}
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error: %v", err)
	}
	hostID := strings.Repeat("a", 32)
	credential, err := secrets.Write(hostID, privateKey)
	if err != nil {
		t.Fatalf("secretStore.Write() error: %v", err)
	}
	now := time.Date(2026, 7, 29, 10, 0, 0, 0, time.UTC)
	if err := store.AddHost(hostRecord{
		ID:                 hostID,
		Name:               "remote",
		Origin:             "https://panel.example.com",
		RemoteNodeID:       strings.Repeat("b", 32),
		ControllerID:       strings.Repeat("c", 32),
		CredentialFile:     credential,
		FederationProtocol: FederationProtocol,
		CreatedAt:          now,
		UpdatedAt:          now,
	}); err != nil {
		t.Fatalf("AddHost() error: %v", err)
	}

	content, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	encodedPrivateKey := base64.RawStdEncoding.EncodeToString(privateKey)
	encodedSeed := base64.RawStdEncoding.EncodeToString(privateKey.Seed())
	if strings.Contains(string(content), encodedPrivateKey) || strings.Contains(string(content), encodedSeed) {
		t.Fatal("private key material was persisted in cluster JSON")
	}
	if !strings.Contains(string(content), credential) {
		t.Fatal("cluster JSON did not retain the non-secret credential reference")
	}

	secretContent, err := os.ReadFile(filepath.Join(directory, "cluster-secrets", credential))
	if err != nil {
		t.Fatalf("read credential file: %v", err)
	}
	if strings.TrimSpace(string(secretContent)) != encodedPrivateKey {
		t.Fatal("credential file does not contain the expected private key")
	}
}
