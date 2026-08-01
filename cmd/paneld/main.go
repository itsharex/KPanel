package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/kejilion/kejilion-panel/internal/agentclient"
	"github.com/kejilion/kejilion-panel/internal/auth"
	"github.com/kejilion/kejilion-panel/internal/contract"
	"github.com/kejilion/kejilion-panel/internal/panel"
	"github.com/kejilion/kejilion-panel/internal/store"
	"github.com/kejilion/kejilion-panel/internal/version"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		slog.Error("paneld stopped", "error", err)
		os.Exit(1)
	}
}

func run(arguments []string) error {
	if len(arguments) > 0 && arguments[0] == "healthcheck" {
		return runHealthcheck(arguments[1:])
	}
	if len(arguments) > 0 && arguments[0] == "agent-healthcheck" {
		return runAgentHealthcheck(arguments[1:])
	}

	flags := flag.NewFlagSet("paneld", flag.ContinueOnError)
	configPath := flags.String("config", "", "path to the JSON configuration file")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	config, err := panel.LoadConfig(*configPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(config.DataDir, 0o700); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}

	storage, err := store.Open(config.StorePath)
	if err != nil {
		return err
	}
	defer storage.Close()
	hasher, err := auth.NewArgon2idHasher(auth.DefaultArgon2idParams())
	if err != nil {
		return err
	}
	authService, err := auth.NewService(storage, hasher, auth.Config{
		BootstrapTokenPath: config.BootstrapTokenPath,
		TOTPKeyPath:        config.TOTPKeyPath,
		SessionTTL:         config.SessionTTL,
		LoginWindow:        config.LoginWindow,
		MaxLoginFailures:   config.MaxLoginFailures,
	})
	if err != nil {
		return err
	}
	if err := authService.EnsureBootstrapToken(); err != nil {
		return err
	}

	agent := panel.NewAgentClient(config.AgentSocket, config.AgentTokenFile, config.MaxAgentBytes)
	handler, err := panel.NewServer(config, authService, storage, agent)
	if err != nil {
		return err
	}
	defer handler.Close()
	server := &http.Server{
		Addr:              config.Listen,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       90 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	shutdownSignal, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	handler.StartBackground(shutdownSignal)
	go func() {
		<-shutdownSignal.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	slog.Info("starting paneld",
		"version", version.Version,
		"protocol", version.ProtocolVersion,
		"listen", config.Listen,
		"initialized", authService.IsInitialized(),
	)
	err = server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func runHealthcheck(arguments []string) error {
	flags := flag.NewFlagSet("paneld healthcheck", flag.ContinueOnError)
	configPath := flags.String("config", "", "path to the JSON configuration file")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	config, err := panel.LoadConfig(*configPath)
	if err != nil {
		return err
	}
	address, err := healthcheckAddress(config.Listen)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 4 * time.Second}
	response, err := client.Get("http://" + address + "/api/v1/health")
	if err != nil {
		return fmt.Errorf("healthcheck request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("healthcheck returned HTTP %d", response.StatusCode)
	}
	return nil
}

func runAgentHealthcheck(arguments []string) error {
	flags := flag.NewFlagSet("paneld agent-healthcheck", flag.ContinueOnError)
	configPath := flags.String("config", "", "path to the JSON configuration file")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected agent-healthcheck argument: %s", flags.Arg(0))
	}
	config, err := panel.LoadConfig(*configPath)
	if err != nil {
		return err
	}
	client, err := agentclient.New(config.AgentSocket, config.AgentTokenFile)
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
	if !health.CoreReady() {
		return fmt.Errorf("Agent is not ready: status %q, reasons %v", health.Status, health.Reasons)
	}
	if health.ReadOnly {
		return errors.New("Agent is unexpectedly read-only")
	}
	return nil
}

func healthcheckAddress(listen string) (string, error) {
	if strings.HasPrefix(listen, ":") {
		return "127.0.0.1" + listen, nil
	}
	host, port, err := net.SplitHostPort(listen)
	if err != nil {
		return "", fmt.Errorf("parse listen address: %w", err)
	}
	switch host {
	case "", "0.0.0.0", "::":
		host = "127.0.0.1"
	}
	return net.JoinHostPort(host, port), nil
}
