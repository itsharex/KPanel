package ai

import (
	"bytes"
	"testing"
)

func TestValidateAttachmentsSecurityBoundary(t *testing.T) {
	items, err := validateAttachments([]Attachment{{Name: "nginx.conf", Data: []byte("server { listen 80; }")}})
	if err != nil || len(items) != 1 || items[0].Kind != "text" || items[0].MimeType != "text/plain" {
		t.Fatalf("text attachment=%#v err=%v", items, err)
	}
	png := append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0}, 16)...)
	items, err = validateAttachments([]Attachment{{Name: "screen.png", Data: png}})
	if err != nil || len(items) != 1 || items[0].Kind != "image" || items[0].MimeType != "image/png" {
		t.Fatalf("image attachment=%#v err=%v", items, err)
	}
	for _, item := range []Attachment{
		{Name: "payload.sh", Data: []byte{0, 1, 2, 3}},
		{Name: "vector.svg", Data: []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`)},
		{Name: "huge.log", Data: bytes.Repeat([]byte("x"), (512<<10)+1)},
	} {
		if _, err := validateAttachments([]Attachment{item}); err == nil {
			t.Fatalf("unsafe attachment accepted: %s", item.Name)
		}
	}
}
