package browsercore

import (
	"fmt"
	"strings"
)

const (
	RuntimeModeDisabled = "disabled"
	RuntimeModeBeta     = "beta"
)

func NormalizeRuntimeMode(value string) (string, error) {
	mode := strings.ToLower(strings.TrimSpace(value))
	switch mode {
	case RuntimeModeDisabled, RuntimeModeBeta:
		return mode, nil
	default:
		return "", fmt.Errorf("browser runtime mode must be %q or %q", RuntimeModeDisabled, RuntimeModeBeta)
	}
}

func RuntimeModeEnabled(value string) bool {
	mode, err := NormalizeRuntimeMode(value)
	return err == nil && mode == RuntimeModeBeta
}
