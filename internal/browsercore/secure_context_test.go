package browsercore

import "testing"

func TestSupportsServiceWorkerOrigin(t *testing.T) {
	for _, origin := range []string{
		"https://panel.example.com",
		"https://192.0.2.10:8443",
		"http://localhost",
		"http://dev.localhost:8080",
		"http://127.0.0.1",
		"http://127.42.0.9:8080",
		"http://[::1]:8080",
	} {
		if !SupportsServiceWorkerOrigin(origin) {
			t.Errorf("trusted origin rejected: %s", origin)
		}
	}
	for _, origin := range []string{
		"",
		"http://panel.example.com",
		"http://192.168.1.10:8080",
		"http://0.0.0.0:8080",
		"http://localhost.example.com",
		"ws://localhost",
		"https://panel.example.com/path",
	} {
		if SupportsServiceWorkerOrigin(origin) {
			t.Errorf("untrusted origin accepted: %s", origin)
		}
	}
}
