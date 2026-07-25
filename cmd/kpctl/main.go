package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/kejilion/kejilion-panel/internal/agentclient"
)

func main() {
	var socketPath string
	var tokenFile string
	flag.StringVar(&socketPath, "socket", envOr("KEJILION_AGENT_SOCKET", "/run/kejilion-panel/agent.sock"), "agent Unix socket")
	flag.StringVar(&tokenFile, "token-file", envOr("KEJILION_AGENT_TOKEN_FILE", "/etc/kejilion-panel/agent.token"), "agent token file")
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: kpctl [flags] health|system|sites|docker-containers")
		os.Exit(2)
	}

	path, ok := map[string]string{
		"health":            "/v1/health",
		"system":            "/v1/system/summary",
		"sites":             "/v1/sites",
		"docker-containers": "/v1/docker/containers",
	}[flag.Arg(0)]
	if !ok {
		fmt.Fprintln(os.Stderr, "unknown command:", flag.Arg(0))
		os.Exit(2)
	}

	client, err := agentclient.New(socketPath, tokenFile)
	if err != nil {
		fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	var result any
	if err := client.Get(ctx, path, &result); err != nil {
		fatal(err)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(result); err != nil {
		fatal(err)
	}
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "kpctl:", err)
	os.Exit(1)
}
