package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kejilion/kejilion-panel/internal/browsercore"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	return runArgs(os.Args[1:])
}

func runArgs(args []string) error {
	if len(args) == 1 && args[0] == "healthcheck" {
		return relayHealthcheck(os.Getenv("KEJILION_BROWSER_RELAY_HEALTH_URL"))
	}

	flags := flag.NewFlagSet("kpanel-browser-relay", flag.ContinueOnError)
	listen := flags.String("listen", "127.0.0.1:8090", "HTTP listen address")
	allowedOrigin := flags.String("allowed-origin", "", "exact KPanel HTTP(S) origin")
	publicURL := flags.String("public-url", "", "public HTTP(S) origin of this relay")
	secretFile := flags.String("secret-file", "", "file containing at least 32 random bytes")
	maxGlobal := flags.Int("max-global", 24, "maximum concurrent upstream requests")
	maxSession := flags.Int("max-session", 6, "maximum concurrent requests per browser session")
	maxRequestBytes := flags.Int64("max-request-bytes", 16<<20, "maximum relayed request body")
	bodyIdleTimeout := flags.Duration("body-idle-timeout", 30*time.Second, "close upstream bodies that stop producing data")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}

	secret, err := browsercore.LoadSecretFile(*secretFile)
	if err != nil {
		return err
	}
	tokens, err := browsercore.NewTokenCodec(secret)
	if err != nil {
		return err
	}
	relay, err := browsercore.NewRelay(browsercore.RelayConfig{
		AllowedOrigin:   *allowedOrigin,
		RelayOrigin:     *publicURL,
		MaxRequestBytes: *maxRequestBytes,
		BodyIdleTimeout: *bodyIdleTimeout,
	}, tokens, browsercore.NewTargetPolicy(nil), browsercore.NewLimiter(*maxGlobal, *maxSession))
	if err != nil {
		return err
	}
	defer relay.CloseIdleConnections()

	server := &http.Server{
		Addr:              *listen,
		Handler:           relay.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    64 << 10,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errorsChannel := make(chan error, 1)
	go func() {
		log.Printf("kpanel browser relay listening on %s", *listen)
		errorsChannel <- server.ListenAndServe()
	}()

	select {
	case err := <-errorsChannel:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	}
}

func relayHealthcheck(rawURL string) error {
	if rawURL == "" {
		rawURL = "http://127.0.0.1:8090/healthz"
	}
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, rawURL, nil)
	if err != nil {
		return fmt.Errorf("create relay healthcheck: %w", err)
	}
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			Proxy: nil,
		},
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("relay healthcheck: %w", err)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4<<10))
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("relay healthcheck returned %s", response.Status)
	}
	return nil
}
