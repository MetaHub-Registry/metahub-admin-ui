// Package server wires the HTTP surface of metahub-admin-ui: the versioned
// JSON API under /api/v1/ and the embedded SPA for everything else.
package server

import (
	"encoding/json"
	"io/fs"
	"net"
	"net/http"
	"strconv"

	"github.com/chi-middleware/logrus-logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	log "github.com/sirupsen/logrus"
)

// Options configures the HTTP server.
type Options struct {
	Port    int
	Assets  fs.FS  // embedded SPA build output (may be nil in tests)
	Version string // reported by /api/v1/health and startup logs
}

// Server is the metahub-admin-ui HTTP service.
type Server struct {
	opts Options
}

// New returns a Server for the given options.
func New(opts Options) *Server {
	return &Server{opts: opts}
}

// Handler builds the full HTTP handler.
func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	reqLog := log.StandardLogger().WithFields(log.Fields{
		"package":    "server",
		"middleware": "logger",
	})
	r.Use(middleware.Heartbeat("/health"))
	r.Use(logger.Logger("server", reqLog))
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", s.handleHealth)
		r.Get("/resolve", handleResolve)
	})

	if s.opts.Assets != nil {
		r.Handle("/*", s.spaHandler())
	}
	return r
}

// ServeHTTP makes *Server directly usable as an http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.Handler().ServeHTTP(w, r)
}

// ListenAndServe starts the HTTP server on the configured port.
func (s *Server) ListenAndServe() error {
	addr := ":" + strconv.Itoa(s.opts.Port)
	log.Infof("metahub-admin-ui v%s listening on %s", s.opts.Version, addr)
	return http.ListenAndServe(addr, s.Handler())
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"version": s.opts.Version,
	})
}

// handleResolve implements GET /api/v1/resolve?ip=<ip>.
//
// Skeleton: the inventory is empty, so the spec-compliant no-match (404 {})
// is the correct answer for every well-formed request. The real resolution
// chain — exact-IP → hostname pattern (PTR, TTL-cached) → CIDR
// longest-prefix — lands with openspec task 2.1–2.3.
//
// Contract (specs/inventory-lookup): the resolve path never responds 5xx.
func handleResolve(w http.ResponseWriter, r *http.Request) {
	ip := r.URL.Query().Get("ip")
	if ip == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "missing ip query parameter",
		})
		return
	}
	if net.ParseIP(ip) == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid ip: " + ip,
		})
		return
	}
	writeNoMatch(w)
}

// writeNoMatch answers 404 with an empty JSON object — the documented
// no-match response consumers fail open on.
func writeNoMatch(w http.ResponseWriter) {
	writeJSON(w, http.StatusNotFound, struct{}{})
}

func writeJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.WithError(err).Debug("failed to encode response")
	}
}
