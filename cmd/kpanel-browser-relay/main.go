package main

import (
	"context"
	"encoding/json"
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
	modeValue := flags.String("mode", browsercore.RuntimeModeDisabled, "browser runtime mode: disabled, reader, or beta")
	secretFile := flags.String("secret-file", "", "file containing at least 32 random bytes")
	maxGlobal := flags.Int("max-global", 64, "maximum concurrent upstream requests")
	maxSession := flags.Int("max-session", 16, "maximum concurrent requests per browser session")
	maxRequestBytes := flags.Int64("max-request-bytes", 16<<20, "maximum relayed request body")
	bodyIdleTimeout := flags.Duration("body-idle-timeout", 30*time.Second, "close upstream bodies that stop producing data")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}

	mode, err := browsercore.NormalizeRuntimeMode(*modeValue)
	if err != nil {
		return err
	}

	var handler http.Handler = disabledRuntimeHandler()
	if browsercore.RuntimeModeUsesRelay(mode) {
		secret, secretErr := browsercore.LoadSecretFile(*secretFile)
		if secretErr != nil {
			return secretErr
		}
		tokens, tokenErr := browsercore.NewTokenCodec(secret)
		if tokenErr != nil {
			return tokenErr
		}
		relay, relayErr := browsercore.NewRelay(browsercore.RelayConfig{
			RuntimeMode:     mode,
			AllowedOrigin:   *allowedOrigin,
			RelayOrigin:     *publicURL,
			MaxRequestBytes: *maxRequestBytes,
			BodyIdleTimeout: *bodyIdleTimeout,
		}, tokens, browsercore.NewTargetPolicy(nil), browsercore.NewLimiter(*maxGlobal, *maxSession))
		if relayErr != nil {
			return relayErr
		}
		defer relay.CloseIdleConnections()
		handler = relay.Handler()
	}

	server := &http.Server{
		Addr:              *listen,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    64 << 10,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errorsChannel := make(chan error, 1)
	go func() {
		log.Printf("kpanel browser relay listening on %s (mode=%s)", *listen, mode)
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

func disabledRuntimeHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if r.URL.Path == "/healthz" {
			switch r.Method {
			case http.MethodGet, http.MethodHead:
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				if r.Method == http.MethodGet {
					_, _ = io.WriteString(w, `{"ok":true,"engine":"kpanel-browser-core","version":1,"mode":"disabled"}`)
				}
			default:
				w.Header().Set("Allow", "GET, HEAD")
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
			return
		}
		http.Error(w, "Embedded browser is disabled", http.StatusServiceUnavailable)
	})
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
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4<<10))
		return fmt.Errorf("relay healthcheck returned %s", response.Status)
	}
	expected := os.Getenv("KEJILION_BROWSER_RELAY_EXPECT_MODE")
	if expected == "" {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4<<10))
		return nil
	}
	var payload struct {
		OK   bool   `json:"ok"`
		Mode string `json:"mode"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 4<<10)).Decode(&payload); err != nil || !payload.OK {
		return errors.New("relay healthcheck returned an invalid payload")
	}
	mode, err := browsercore.NormalizeRuntimeMode(expected)
	if err != nil || payload.Mode != mode {
		return fmt.Errorf("relay healthcheck mode mismatch: got %q, want %q", payload.Mode, expected)
	}
	return nil
}
