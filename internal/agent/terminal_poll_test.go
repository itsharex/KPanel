package agent

import (
	"context"
	"net/url"
	"sync/atomic"
	"testing"
	"time"
)

func TestParseTerminalReadQueryIsStrictAndBounded(t *testing.T) {
	query, err := parseTerminalReadQuery(url.Values{
		"offset":    {"42"},
		"wait":      {"1000"},
		"inputOpen": {"true"},
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	if query.Offset != 42 || query.Wait != time.Second ||
		!query.HasInputState || !query.InputOpen {
		t.Fatalf("terminal query = %#v", query)
	}

	for name, values := range map[string]url.Values{
		"missing offset":  {"wait": {"1000"}},
		"negative offset": {"offset": {"-1"}},
		"long wait":       {"offset": {"0"}, "wait": {"1501"}},
		"duplicate":       {"offset": {"0", "1"}},
		"unknown":         {"offset": {"0"}, "command": {"id"}},
		"invalid state":   {"offset": {"0"}, "inputOpen": {"maybe"}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseTerminalReadQuery(values, true); err == nil {
				t.Fatalf("query was accepted: %#v", values)
			}
		})
	}
}

func TestWaitForTerminalChunkReturnsWhenOutputArrives(t *testing.T) {
	var ready atomic.Bool
	var reads atomic.Int32
	go func() {
		time.Sleep(40 * time.Millisecond)
		ready.Store(true)
	}()

	started := time.Now()
	value, err := waitForTerminalChunk(
		context.Background(),
		time.Second,
		func() (string, error) {
			reads.Add(1)
			if ready.Load() {
				return "output", nil
			}
			return "", nil
		},
		func(value string) bool { return value != "" },
	)
	if err != nil {
		t.Fatal(err)
	}
	if value != "output" || reads.Load() < 2 {
		t.Fatalf("long poll result = %q reads=%d", value, reads.Load())
	}
	if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
		t.Fatalf("terminal output notification took %s", elapsed)
	}
}

func TestWaitForTerminalChunkHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := waitForTerminalChunk(
		ctx,
		time.Second,
		func() (string, error) { return "", nil },
		func(value string) bool { return value != "" },
	); err != context.Canceled {
		t.Fatalf("canceled long poll error = %v", err)
	}
}
