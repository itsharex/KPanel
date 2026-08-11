package main

import (
	"context"
	"errors"
	"flag"
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
	listen := flag.String("listen", "127.0.0.1:8090", "HTTP listen address")
	allowedOrigin := flag.String("allowed-origin", "", "exact KPanel HTTP(S) origin")
	publicURL := flag.String("public-url", "", "public HTTP(S) origin of this relay")
	secretFile := flag.String("secret-file", "", "file containing at least 32 random bytes")
	maxGlobal := flag.Int("max-global", 24, "maximum concurrent upstream requests")
	maxSession := flag.Int("max-session", 6, "maximum concurrent requests per browser session")
	maxRequestBytes := flag.Int64("max-request-bytes", 16<<20, "maximum relayed request body")
	bodyIdleTimeout := flag.Duration("body-idle-timeout", 30*time.Second, "close upstream bodies that stop producing data")
	flag.Parse()

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
