package cluster

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestV2PairingCodeRoundTripAndValidation(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 29, 15, 0, 0, 0, time.UTC)
	targetKey, err := v2NoiseSuite.GenerateKeypair(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKeypair() error = %v", err)
	}
	nodeID := strings.Repeat("a", 32)
	code, codeID, pairingKey, err := makeV2PairingCode(
		nodeID, targetKey.Public, now.Add(5*time.Minute),
	)
	if err != nil {
		t.Fatalf("makeV2PairingCode() error = %v", err)
	}
	if !strings.HasPrefix(code, v2PairingCodePrefix) || len(codeID) != 16 ||
		len(pairingKey) != 32 {
		t.Fatalf("unexpected pairing code metadata: prefix=%v id=%q key=%d", strings.HasPrefix(code, v2PairingCodePrefix), codeID, len(pairingKey))
	}

	descriptor, err := parseV2PairingCode(code, now)
	if err != nil {
		t.Fatalf("parseV2PairingCode() error = %v", err)
	}
	if descriptor.NodeID != nodeID || descriptor.CodeID != codeID ||
		!bytes.Equal(descriptor.TargetPublicKey, targetKey.Public) ||
		!bytes.Equal(descriptor.PairingKey, pairingKey) ||
		!descriptor.ExpiresAt.Equal(now.Add(5*time.Minute)) {
		t.Fatalf("unexpected descriptor: %#v", descriptor)
	}

	if _, err := parseV2PairingCode(code, descriptor.ExpiresAt); !errors.Is(err, ErrPairingCode) {
		t.Fatalf("expired parse error = %v, want ErrPairingCode", err)
	}
	for _, malformed := range []string{
		"",
		"kp2.",
		"kp2.not-base64",
		strings.Replace(code, "kp2.", "kp3.", 1),
		code + "extra",
	} {
		if _, err := parseV2PairingCode(malformed, now); !errors.Is(err, ErrPairingCode) {
			t.Fatalf("parseV2PairingCode(%q) error = %v, want ErrPairingCode", malformed, err)
		}
	}
}

func TestV2NoisePairAndDailyRequestProtectPayloads(t *testing.T) {
	t.Parallel()

	targetKey, err := v2NoiseSuite.GenerateKeypair(rand.Reader)
	if err != nil {
		t.Fatalf("target GenerateKeypair() error = %v", err)
	}
	controllerKey, err := v2NoiseSuite.GenerateKeypair(rand.Reader)
	if err != nil {
		t.Fatalf("controller GenerateKeypair() error = %v", err)
	}
	pairingKey := bytes.Repeat([]byte{0x42}, 32)
	request := v2Envelope{
		Protocol:     FederationProtocolV2,
		ControllerID: strings.Repeat("b", 32),
		TargetID:     strings.Repeat("a", 32),
		CodeID:       strings.Repeat("c", 16),
		Timestamp:    time.Date(2026, 7, 29, 15, 1, 0, 0, time.UTC).Unix(),
		RequestID:    strings.Repeat("d", 32),
	}
	pairPayload := []byte(`{"action":"pair.prepare","controllerName":"secret-controller"}`)

	sealed, initiator, err := sealV2Request(
		http.MethodPost, v2PairPath, request,
		controllerKey, targetKey.Public, pairingKey, pairPayload,
	)
	if err != nil {
		t.Fatalf("sealV2Request(pair) error = %v", err)
	}
	wire, err := base64.RawURLEncoding.DecodeString(sealed.Message)
	if err != nil {
		t.Fatalf("DecodeString(pair message) error = %v", err)
	}
	if bytes.Contains(wire, pairPayload) || bytes.Contains(wire, []byte("secret-controller")) {
		t.Fatal("pair payload appeared in plaintext on the wire")
	}
	opened, peerStatic, responder, err := openV2Request(
		http.MethodPost, v2PairPath, sealed, targetKey, pairingKey,
	)
	if err != nil {
		t.Fatalf("openV2Request(pair) error = %v", err)
	}
	if !bytes.Equal(opened, pairPayload) || !bytes.Equal(peerStatic, controllerKey.Public) {
		t.Fatal("pair payload or authenticated controller key did not round-trip")
	}
	responsePayload := []byte(`{"accepted":true,"hostname":"target"}`)
	response, err := sealV2Response(sealed, responder, responsePayload)
	if err != nil {
		t.Fatalf("sealV2Response(pair) error = %v", err)
	}
	decoded, err := openV2Response(sealed, response, initiator)
	if err != nil {
		t.Fatalf("openV2Response(pair) error = %v", err)
	}
	if !bytes.Equal(decoded, responsePayload) {
		t.Fatalf("pair response = %q, want %q", decoded, responsePayload)
	}

	dailyPayload := []byte(`{"action":"summary","requestId":"dddddddddddddddddddddddddddddddd"}`)
	daily, dailyInitiator, err := sealV2Request(
		http.MethodPost, v2SummaryPath, request,
		controllerKey, targetKey.Public, nil, dailyPayload,
	)
	if err != nil {
		t.Fatalf("sealV2Request(summary) error = %v", err)
	}
	dailyOpened, dailyPeer, dailyResponder, err := openV2Request(
		http.MethodPost, v2SummaryPath, daily, targetKey, nil,
	)
	if err != nil {
		t.Fatalf("openV2Request(summary) error = %v", err)
	}
	if !bytes.Equal(dailyOpened, dailyPayload) || !bytes.Equal(dailyPeer, controllerKey.Public) {
		t.Fatal("daily payload or authenticated controller key did not round-trip")
	}
	telemetry := []byte(`{"nodeId":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","telemetry":{"hostname":"target"}}`)
	dailyResponse, err := sealV2Response(daily, dailyResponder, telemetry)
	if err != nil {
		t.Fatalf("sealV2Response(summary) error = %v", err)
	}
	decodedTelemetry, err := openV2Response(daily, dailyResponse, dailyInitiator)
	if err != nil {
		t.Fatalf("openV2Response(summary) error = %v", err)
	}
	if !bytes.Equal(decodedTelemetry, telemetry) {
		t.Fatalf("summary response = %q, want %q", decodedTelemetry, telemetry)
	}
	responseWire, _ := base64.RawURLEncoding.DecodeString(dailyResponse.Message)
	if bytes.Contains(responseWire, telemetry) || bytes.Contains(responseWire, []byte("target")) {
		t.Fatal("telemetry appeared in plaintext on the wire")
	}
}

func TestV2NoiseRejectsTamperingWrongContextAndWrongCredentials(t *testing.T) {
	t.Parallel()

	targetKey, err := v2NoiseSuite.GenerateKeypair(rand.Reader)
	if err != nil {
		t.Fatalf("target GenerateKeypair() error = %v", err)
	}
	controllerKey, err := v2NoiseSuite.GenerateKeypair(rand.Reader)
	if err != nil {
		t.Fatalf("controller GenerateKeypair() error = %v", err)
	}
	otherTarget, err := v2NoiseSuite.GenerateKeypair(rand.Reader)
	if err != nil {
		t.Fatalf("other target GenerateKeypair() error = %v", err)
	}
	pairingKey := bytes.Repeat([]byte{0x24}, 32)
	request := v2Envelope{
		Protocol:     FederationProtocolV2,
		ControllerID: strings.Repeat("b", 32),
		TargetID:     strings.Repeat("a", 32),
		CodeID:       strings.Repeat("c", 16),
		Timestamp:    time.Date(2026, 7, 29, 15, 2, 0, 0, time.UTC).Unix(),
		RequestID:    strings.Repeat("d", 32),
	}
	sealed, _, err := sealV2Request(
		http.MethodPost, v2PairPath, request,
		controllerKey, targetKey.Public, pairingKey, []byte(`{"action":"pair.prepare"}`),
	)
	if err != nil {
		t.Fatalf("sealV2Request() error = %v", err)
	}

	tampered := sealed
	message, _ := base64.RawURLEncoding.DecodeString(tampered.Message)
	message[len(message)-1] ^= 0x01
	tampered.Message = base64.RawURLEncoding.EncodeToString(message)
	if _, _, _, err := openV2Request(http.MethodPost, v2PairPath, tampered, targetKey, pairingKey); !errors.Is(err, ErrAuthentication) {
		t.Fatalf("tampered message error = %v, want ErrAuthentication", err)
	}
	if _, _, _, err := openV2Request(http.MethodPost, v2SummaryPath, sealed, targetKey, pairingKey); !errors.Is(err, ErrAuthentication) {
		t.Fatalf("cross-path message error = %v, want ErrAuthentication", err)
	}
	wrongKey := bytes.Repeat([]byte{0x25}, 32)
	if _, _, _, err := openV2Request(http.MethodPost, v2PairPath, sealed, targetKey, wrongKey); !errors.Is(err, ErrAuthentication) {
		t.Fatalf("wrong PSK error = %v, want ErrAuthentication", err)
	}
	if _, _, _, err := openV2Request(http.MethodPost, v2PairPath, sealed, otherTarget, pairingKey); !errors.Is(err, ErrAuthentication) {
		t.Fatalf("wrong target identity error = %v, want ErrAuthentication", err)
	}
	crossNode := sealed
	crossNode.TargetID = strings.Repeat("e", 32)
	if _, _, _, err := openV2Request(http.MethodPost, v2PairPath, crossNode, targetKey, pairingKey); !errors.Is(err, ErrAuthentication) {
		t.Fatalf("cross-node message error = %v, want ErrAuthentication", err)
	}
}

func BenchmarkV2NoiseSummaryRoundTrip(b *testing.B) {
	targetKey, err := v2NoiseSuite.GenerateKeypair(rand.Reader)
	if err != nil {
		b.Fatal(err)
	}
	controllerKey, err := v2NoiseSuite.GenerateKeypair(rand.Reader)
	if err != nil {
		b.Fatal(err)
	}
	request := v2Envelope{
		Protocol:     FederationProtocolV2,
		ControllerID: strings.Repeat("b", 32),
		TargetID:     strings.Repeat("a", 32),
		Timestamp:    time.Date(2026, 7, 29, 15, 3, 0, 0, time.UTC).Unix(),
		RequestID:    strings.Repeat("d", 32),
	}
	input := []byte(`{"action":"summary"}`)
	output := []byte(`{"nodeId":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","telemetry":{"hostname":"target"}}`)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		sealed, initiator, err := sealV2Request(
			http.MethodPost,
			v2SummaryPath,
			request,
			controllerKey,
			targetKey.Public,
			nil,
			input,
		)
		if err != nil {
			b.Fatal(err)
		}
		_, _, responder, err := openV2Request(
			http.MethodPost,
			v2SummaryPath,
			sealed,
			targetKey,
			nil,
		)
		if err != nil {
			b.Fatal(err)
		}
		response, err := sealV2Response(sealed, responder, output)
		if err != nil {
			b.Fatal(err)
		}
		if _, err := openV2Response(sealed, response, initiator); err != nil {
			b.Fatal(err)
		}
	}
}
