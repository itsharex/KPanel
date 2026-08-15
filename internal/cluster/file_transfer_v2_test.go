package cluster

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"io"
	"testing"
	"time"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

func TestFederationFileStreamUsesHandshakeTransportCipherAndAuthenticatedEnd(t *testing.T) {
	controllerKey, err := v2NoiseSuite.GenerateKeypair(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	targetKey, err := v2NoiseSuite.GenerateKeypair(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	request := v2Envelope{
		Protocol: FederationProtocolV2, ControllerID: "aabbccddeeff00112233445566778899",
		TargetID: "11223344556677889900aabbccddeeff", Timestamp: time.Now().UTC().Unix(),
		RequestID: "00112233445566778899aabbccddeeff",
	}
	sealed, initiator, err := sealV2Request(
		"POST", v2FileOpenPath, request, controllerKey, targetKey.Public, nil,
		[]byte(`{"path":"/app","resourceVersion":"sha256:source"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	_, _, responder, err := openV2Request("POST", v2FileOpenPath, sealed, targetKey, nil)
	if err != nil {
		t.Fatal(err)
	}
	authorization := &FederationFileAuthorization{request: sealed, handshake: responder}
	response, encrypt, err := authorization.SealMetadata(contract.FileTransferMetadata{
		Name: "app", Kind: "directory", ResourceVersion: "sha256:source",
	})
	if err != nil {
		t.Fatal(err)
	}
	message, err := base64.RawURLEncoding.DecodeString(response.Message)
	if err != nil {
		t.Fatal(err)
	}
	_, _, decrypt, err := initiator.ReadMessage(nil, message)
	if err != nil || decrypt == nil {
		t.Fatalf("finish initiator handshake: decrypt=%v err=%v", decrypt, err)
	}

	content := bytes.Repeat([]byte("kpanel-transfer-"), 8_000)
	var stream bytes.Buffer
	writer := NewFederationFileWriter(&stream, encrypt)
	if _, err := writer.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Finish(nil); err != nil {
		t.Fatal(err)
	}
	reader := &federationFileReader{
		source: bufio.NewReader(bytes.NewReader(stream.Bytes())),
		body:   io.NopCloser(bytes.NewReader(nil)), cipher: decrypt,
	}
	plain, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(plain, content) {
		t.Fatalf("stream content mismatch: got=%d want=%d", len(plain), len(content))
	}
}

func TestFederationFileStreamRejectsTruncation(t *testing.T) {
	reader := &federationFileReader{
		source: bufio.NewReader(bytes.NewReader(nil)),
		body:   io.NopCloser(bytes.NewReader(nil)),
	}
	if _, err := reader.Read(make([]byte, 1)); err != io.ErrUnexpectedEOF {
		t.Fatalf("truncated stream error=%v", err)
	}
}

func TestFederationFileStreamClosesIdleSource(t *testing.T) {
	input, output := io.Pipe()
	reader := newIdleReadCloser(input, 20*time.Millisecond)
	t.Cleanup(func() {
		_ = reader.Close()
		_ = output.Close()
	})
	started := time.Now()
	_, err := reader.Read(make([]byte, 1))
	if err == nil {
		t.Fatal("idle read unexpectedly succeeded")
	}
	if elapsed := time.Since(started); elapsed < 15*time.Millisecond || elapsed > time.Second {
		t.Fatalf("idle read elapsed=%v", elapsed)
	}
}
