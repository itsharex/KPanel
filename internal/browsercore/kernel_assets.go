package browsercore

import (
	_ "embed"
	"html"
	"net/http"
	"strings"
)

//go:embed kernel.html
var kernelHTML string

//go:embed kernel.js
var kernelJS []byte

//go:embed kernel.css
var kernelCSS []byte

//go:embed kernel_sw.js
var kernelServiceWorker []byte

//go:embed kernel_transport.mjs
var kernelTransport []byte

//go:embed vendor/scramjet-v2/scramjet.js
var scramjetV2JS []byte

//go:embed vendor/scramjet-v2/scramjet.wasm
var scramjetV2WASM []byte

//go:embed vendor/controller/controller.api.js
var scramjetControllerAPI []byte

//go:embed vendor/controller/controller.inject.js
var scramjetControllerInject []byte

//go:embed vendor/controller/controller.sw.js
var scramjetControllerWorker []byte

func (r *Relay) registerKernel(mux *http.ServeMux) {
	mux.HandleFunc("GET /kernel/{$}", r.handleKernel)
	mux.HandleFunc("GET /kernel/kernel.js", func(w http.ResponseWriter, request *http.Request) {
		r.serveKernelAsset(w, request, "text/javascript; charset=utf-8", kernelJS)
	})
	mux.HandleFunc("GET /kernel/kernel.css", func(w http.ResponseWriter, request *http.Request) {
		r.serveKernelAsset(w, request, "text/css; charset=utf-8", kernelCSS)
	})
	mux.HandleFunc("GET /kernel/runtime/v3/sw.js", func(w http.ResponseWriter, _ *http.Request) {
		r.setKernelHeaders(w)
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("Service-Worker-Allowed", "/")
		_, _ = w.Write(kernelServiceWorker)
	})
	mux.HandleFunc("GET /kernel/runtime/v3/transport.mjs", func(w http.ResponseWriter, request *http.Request) {
		r.serveVersionedKernelAsset(w, request, "text/javascript; charset=utf-8", kernelTransport)
	})
	for path, asset := range map[string]struct {
		contentType string
		content     []byte
	}{
		"/scramjet/scramjet.js":            {"text/javascript; charset=utf-8", scramjetV2JS},
		"/scramjet/scramjet.wasm":          {"application/wasm", scramjetV2WASM},
		"/controller/controller.api.js":    {"text/javascript; charset=utf-8", scramjetControllerAPI},
		"/controller/controller.inject.js": {"text/javascript; charset=utf-8", scramjetControllerInject},
		"/controller/controller.sw.js":     {"text/javascript; charset=utf-8", scramjetControllerWorker},
	} {
		path, asset := path, asset
		mux.HandleFunc("GET "+path, func(w http.ResponseWriter, request *http.Request) {
			r.serveVersionedKernelAsset(w, request, asset.contentType, asset.content)
		})
	}
}

func (r *Relay) handleKernel(w http.ResponseWriter, request *http.Request) {
	r.setKernelHeaders(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	page := strings.ReplaceAll(kernelHTML, "{{PANEL_ORIGIN}}", html.EscapeString(r.config.AllowedOrigin))
	_, _ = w.Write([]byte(page))
}

func (r *Relay) serveKernelAsset(w http.ResponseWriter, _ *http.Request, contentType string, content []byte) {
	r.setKernelHeaders(w)
	w.Header().Set("Content-Type", contentType)
	_, _ = w.Write(content)
}

func (r *Relay) serveVersionedKernelAsset(w http.ResponseWriter, _ *http.Request, contentType string, content []byte) {
	r.setKernelHeaders(w)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
	_, _ = w.Write(content)
}

func (r *Relay) setKernelHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	// Scramjet v1 evaluates rewritten page scripts inside this isolated Relay
	// origin. Keep that capability out of the Panel origin and constrain every
	// other source to the vendored runtime served here.
	w.Header().Set("Content-Security-Policy", "default-src 'none'; base-uri 'none'; frame-ancestors "+r.config.AllowedOrigin+"; script-src 'self' 'unsafe-eval' 'wasm-unsafe-eval'; style-src 'self' 'unsafe-inline'; connect-src 'self'; worker-src 'self' blob:; frame-src blob: data: 'self'; img-src data: blob: 'self'; object-src 'none'; form-action 'none'")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cross-Origin-Resource-Policy", "cross-origin")
	w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
}
