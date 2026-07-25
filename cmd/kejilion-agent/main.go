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
	"github.com/kejilion/kejilion-panel/internal/dockerx"
	"github.com/kejilion/kejilion-panel/internal/sites"
	"github.com/kejilion/kejilion-panel/internal/systeminfo"
	"github.com/kejilion/kejilion-panel/internal/version"
)

func main() {
	if err := run(); err != nil {
		slog.Error("kejilion-agent stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	socketPath := flag.String("socket", env("KEJILION_AGENT_SOCKET", "/run/kejilion-panel/agent.sock"), "Unix Socket path")
	socketGroup := flag.String("socket-group", env("KEJILION_AGENT_SOCKET_GROUP", "kejilion-panel"), "Unix Socket group")
	tokenFile := flag.String("token-file", env("KEJILION_AGENT_TOKEN_FILE", "/etc/kejilion-panel/agent.token"), "shared token file")
	stateDir := flag.String("state-dir", env("KEJILION_AGENT_STATE_DIR", "/var/lib/kejilion-panel"), "Agent state directory")
	webRoot := flag.String("web-root", env("KEJILION_WEB_ROOT", "/home/web"), "Kejilion Web root")
	dockerSocket := flag.String("docker-socket", env("KEJILION_DOCKER_SOCKET", "/var/run/docker.sock"), "Docker Engine Unix Socket")
	dockerPIDFile := flag.String("docker-pid-file", env("KEJILION_DOCKER_PID_FILE", "/run/docker.pid"), "Docker daemon PID file")
	allowDockerSocketActivation := flag.Bool(
		"allow-docker-socket-activation",
		envBool("KEJILION_DOCKER_ALLOW_SOCKET_ACTIVATION", false),
		"allow connecting to Docker Socket when no running dockerd PID can be verified",
	)
	flag.Parse()

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
