package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/kejilion/kejilion-panel/internal/agent"
	"github.com/kejilion/kejilion-panel/internal/agentclient"
	"github.com/kejilion/kejilion-panel/internal/contract"
	"github.com/kejilion/kejilion-panel/internal/dockerx"
	"github.com/kejilion/kejilion-panel/internal/sites"
	"github.com/kejilion/kejilion-panel/internal/systeminfo"
	"github.com/kejilion/kejilion-panel/internal/version"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		slog.Error("kejilion-agent stopped", "error", err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	if len(arguments) > 0 && arguments[0] == "version" {
		if len(arguments) != 1 {
			return errors.New("kejilion-agent version does not accept arguments")
		}
		fmt.Printf("%s %s\n", version.Version, version.ProtocolVersion)
		return nil
	}
	if len(arguments) > 0 && arguments[0] == "healthcheck" {
		return runHealthcheck(arguments[1:])
	}

	flags := flag.NewFlagSet("kejilion-agent", flag.ContinueOnError)
	socketPath := flags.String("socket", env("KEJILION_AGENT_SOCKET", "/run/kejilion-panel/agent.sock"), "Unix Socket path")
	socketGroup := flags.String("socket-group", env("KEJILION_AGENT_SOCKET_GROUP", "kejilion-panel"), "Unix Socket group")
	tokenFile := flags.String("token-file", env("KEJILION_AGENT_TOKEN_FILE", "/etc/kejilion-panel/agent.token"), "shared token file")
	stateDir := flags.String("state-dir", env("KEJILION_AGENT_STATE_DIR", "/var/lib/kejilion-panel"), "Agent state directory")
	webRoot := flags.String("web-root", env("KEJILION_WEB_ROOT", "/home/web"), "Kejilion Web root")
	dockerSocket := flags.String("docker-socket", env("KEJILION_DOCKER_SOCKET", "/var/run/docker.sock"), "Docker Engine Unix Socket")
	dockerPIDFile := flags.String("docker-pid-file", env("KEJILION_DOCKER_PID_FILE", "/run/docker.pid"), "Docker daemon PID file")
	allowDockerSocketActivation := flags.Bool(
		"allow-docker-socket-activation",
		envBool("KEJILION_DOCKER_ALLOW_SOCKET_ACTIVATION", false),
		"allow connecting to Docker Socket when no running dockerd PID can be verified",
	)
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected argument: %s", flags.Arg(0))
	}

	token, err := agent.PrepareTokenFile(*tokenFile, *socketGroup)
	if err != nil {
		return err
	}
	dockerClient := dockerx.New(*dockerSocket, *webRoot, *stateDir)
	dockerClient.ConfigureDaemonAccess(*dockerPIDFile, *allowDockerSocketActivation)
	handler, err := agent.NewServer(agent.Config{
		Token: token, Version: version.Version, ProtocolVersion: version.ProtocolVersion,
		WebRoot: *webRoot, System: systeminfo.NewCollector(),
		Sites: sites.NewDiscoverer(*webRoot), Docker: dockerClient,
	})
	clear(token)
	if err != nil {
		return err
	}
	listener, cleanup, err := agent.ListenUnix(*socketPath, *socketGroup)
	if err != nil {
		return err
	}
	defer cleanup()

	httpServer := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- httpServer.Serve(listener)
	}()
	slog.Info("kejilion-agent listening", "socket", *socketPath, "version", version.Version)
	select {
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve Agent API: %w", err)
		}
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownContext); err != nil {
			return fmt.Errorf("shutdown Agent API: %w", err)
		}
	}
	return nil
}

func runHealthcheck(arguments []string) error {
	flags := flag.NewFlagSet("kejilion-agent healthcheck", flag.ContinueOnError)
	socketPath := flags.String("socket", env("KEJILION_AGENT_SOCKET", "/run/kejilion-panel/agent.sock"), "Unix Socket path")
	tokenFile := flags.String("token-file", env("KEJILION_AGENT_TOKEN_FILE", "/etc/kejilion-panel/agent.token"), "shared token file")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected healthcheck argument: %s", flags.Arg(0))
	}

	client, err := agentclient.New(*socketPath, *tokenFile)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	var health contract.AgentHealth
	if err := client.Get(ctx, "/v1/health", &health); err != nil {
		return fmt.Errorf("Agent healthcheck request: %w", err)
	}
	return validateAgentHealth(health)
}

func validateAgentHealth(health contract.AgentHealth) error {
	if health.Version != version.Version {
		return fmt.Errorf("Agent version mismatch: running %q, expected %q", health.Version, version.Version)
	}
	if health.ProtocolVersion != version.ProtocolVersion {
		return fmt.Errorf(
			"Agent protocol mismatch: running %q, expected %q",
			health.ProtocolVersion,
			version.ProtocolVersion,
		)
	}
	if health.Status != "ok" {
		return fmt.Errorf("Agent is not ready: status %q, reasons %v", health.Status, health.Reasons)
	}
	if health.ReadOnly {
		return errors.New("Agent is unexpectedly read-only")
	}
	return nil
}

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func envBool(name string, fallback bool) bool {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func clear(value []byte) {
	for i := range value {
		value[i] = 0
	}
}
