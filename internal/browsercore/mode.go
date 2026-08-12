package browsercore

import (
	"fmt"
	"strings"
)

const (
	RuntimeModeDisabled = "disabled"
	RuntimeModeReader   = "reader"
	RuntimeModeBeta     = "beta"
)

func NormalizeRuntimeMode(value string) (string, error) {
	mode := strings.ToLower(strings.TrimSpace(value))
	switch mode {
	case RuntimeModeDisabled, RuntimeModeReader, RuntimeModeBeta:
		return mode, nil
	default:
		return "", fmt.Errorf("browser runtime mode must be %q, %q, or %q", RuntimeModeDisabled, RuntimeModeReader, RuntimeModeBeta)
	}
}

func RuntimeModeEnabled(value string) bool {
	mode, err := NormalizeRuntimeMode(value)
	return err == nil && mode == RuntimeModeBeta
}

func RuntimeModeUsesRelay(value string) bool {
	mode, err := NormalizeRuntimeMode(value)
	return err == nil && (mode == RuntimeModeReader || mode == RuntimeModeBeta)
}
