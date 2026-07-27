package main

import (
	"strings"
	"testing"

	"github.com/kejilion/kejilion-panel/internal/contract"
	"github.com/kejilion/kejilion-panel/internal/version"
)

func TestValidateAgentHealthAcceptsReadyAgent(t *testing.T) {
	err := validateAgentHealth(contract.AgentHealth{
		Status:          "ok",
		Version:         version.Version,
		ProtocolVersion: version.ProtocolVersion,
	})
	if err != nil {
		t.Fatalf("validateAgentHealth() error = %v", err)
	}
}

func TestValidateAgentHealthAcceptsMissingOptionalWebRoot(t *testing.T) {
	err := validateAgentHealth(contract.AgentHealth{
		Status:          "degraded",
		Version:         version.Version,
		ProtocolVersion: version.ProtocolVersion,
		Reasons:         []string{"web_root_unavailable"},
	})
	if err != nil {
		t.Fatalf("validateAgentHealth() error = %v", err)
	}
}

func TestValidateAgentHealthRejectsDegradedAgent(t *testing.T) {
	err := validateAgentHealth(contract.AgentHealth{
		Status:          "degraded",
		Version:         version.Version,
		ProtocolVersion: version.ProtocolVersion,
		Reasons:         []string{"docker_unavailable"},
	})
	if err == nil || !strings.Contains(err.Error(), "docker_unavailable") {
		t.Fatalf("validateAgentHealth() error = %v, want degraded reason", err)
	}
}

func TestValidateAgentHealthRejectsReadOnlyAgent(t *testing.T) {
	err := validateAgentHealth(contract.AgentHealth{
		Status:          "ok",
		Version:         version.Version,
		ProtocolVersion: version.ProtocolVersion,
		ReadOnly:        true,
	})
	if err == nil {
		t.Fatal("validateAgentHealth() accepted a read-only Agent")
	}
}
