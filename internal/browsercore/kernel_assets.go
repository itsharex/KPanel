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

func (r *Relay) registerKernel(mux *http.ServeMux) {
	mux.HandleFunc("GET /kernel/{$}", r.handleKernel)
	mux.HandleFunc("GET /kernel/kernel.js", func(w http.ResponseWriter, request *http.Request) {
		r.serveKernelAsset(w, request, "text/javascript; charset=utf-8", kernelJS)
	})
	mux.HandleFunc("GET /kernel/kernel.css", func(w http.ResponseWriter, request *http.Request) {
		r.serveKernelAsset(w, request, "text/css; charset=utf-8", kernelCSS)
	})
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

func (r *Relay) setKernelHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; base-uri 'none'; frame-ancestors "+r.config.AllowedOrigin+"; script-src 'self'; style-src 'self' 'unsafe-inline'; connect-src 'self'; frame-src blob: 'self'; img-src data: blob: 'self'; object-src 'none'; form-action 'none'")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cross-Origin-Resource-Policy", "cross-origin")
	w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
}
