package browsercore

import "testing"

func TestNormalizeRuntimeMode(t *testing.T) {
	for input, expected := range map[string]string{
		"disabled": RuntimeModeDisabled,
		" reader ": RuntimeModeReader,
		"READER":   RuntimeModeReader,
		" beta ":   RuntimeModeBeta,
		"BETA":     RuntimeModeBeta,
	} {
		actual, err := NormalizeRuntimeMode(input)
		if err != nil || actual != expected {
			t.Fatalf("NormalizeRuntimeMode(%q) = %q, %v", input, actual, err)
		}
	}
	for _, input := range []string{"", "enabled", "stable", "alpha"} {
		if _, err := NormalizeRuntimeMode(input); err == nil {
			t.Errorf("NormalizeRuntimeMode(%q) accepted an unsupported mode", input)
		}
	}
}

func TestRuntimeModeEnabledOnlyForBeta(t *testing.T) {
	if RuntimeModeEnabled(RuntimeModeDisabled) || RuntimeModeEnabled(RuntimeModeReader) || RuntimeModeEnabled("enabled") ||
		!RuntimeModeEnabled(RuntimeModeBeta) || !RuntimeModeEnabled(" BETA ") {
		t.Fatal("runtime mode gate does not fail closed")
	}
}

func TestRuntimeModeUsesRelayForReaderAndBeta(t *testing.T) {
	if RuntimeModeUsesRelay(RuntimeModeDisabled) || RuntimeModeUsesRelay("enabled") ||
		!RuntimeModeUsesRelay(RuntimeModeReader) || !RuntimeModeUsesRelay(RuntimeModeBeta) {
		t.Fatal("relay mode gate does not fail closed")
	}
}
