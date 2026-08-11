package browsercore

import (
	"bytes"
	"errors"
	"fmt"
	"os"
)

func LoadSecretFile(path string) ([]byte, error) {
	if path == "" {
		return nil, errors.New("browser relay secret file is required")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read browser relay secret: %w", err)
	}
	if len(content) > 4<<10 {
		return nil, errors.New("browser relay secret file exceeds 4 KiB")
	}
	secret := bytes.TrimSpace(content)
	if len(secret) < 32 {
		return nil, errors.New("browser relay secret must contain at least 32 bytes")
	}
	return append([]byte(nil), secret...), nil
}
