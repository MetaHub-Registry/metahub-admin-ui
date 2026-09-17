// Package server wires the HTTP surface of metahub-admin-ui: the versioned
// JSON API under /api/v1/ and the embedded SPA for everything else.
package server

import (
	"crypto/subtle"
	"encoding/json"
	"io/fs"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/chi-middleware/logrus-logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	log "github.com/sirupsen/logrus"

	"github.com/metahub-registry/metahub-admin-ui/internal/inventory"
)

// Options configures the HTTP server.
type Options struct {
	Port           int
	Assets         fs.FS             // embedded SPA build output (may be nil in tests)
	Version        string            // reported by /api/v1/health and startup logs
	Engine         *inventory.Engine // resolution engine (nil ⇒ always no-match)
	InventoryToken string            // optional bearer check on /api/v1/resolve
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
		// Health stays open — k8s probes and humans.
		r.Get("/health", s.handleHealth)
		// The resolve + entry-management surface optionally requires a bearer
		// token (spec: inventory-lookup). Unset INVENTORY_TOKEN ⇒ network
		// trust (v0).
		r.Group(func(r chi.Router) {
			if s.opts.InventoryToken != "" {
				r.Use(bearerAuth(s.opts.InventoryToken))
			}
			r.Get("/resolve", s.handleResolve)
			r.Get("/inventory", s.handleInventory)
			// Runtime entries (openspec add-runtime-entry-api): ephemeral,
			// in-memory overlay on top of the YAML sources.
			r.Post("/entries", s.handleAddEntry)
			r.Delete("/entries", s.handleDeleteEntry)
		})
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
// Evaluates the configured resolution chain (exact-IP → hostname pattern →
// CIDR; see internal/inventory). Contract (specs/inventory-lookup): 200 hit
// with {ip, profile, hostname?, source}, 404 {} on no-match — the resolve
// path never responds 5xx.
func (s *Server) handleResolve(w http.ResponseWriter, r *http.Request) {
	ip := r.URL.Query().Get("ip")
	if ip == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "missing ip query parameter",
		})
		return
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid ip: " + ip,
		})
		return
	}
	if s.opts.Engine == nil {
		// Defensive: a server built without an engine answers no-match —
		// never a 5xx.
		writeNoMatch(w)
		return
	}
	res, ok := s.opts.Engine.Resolve(r.Context(), parsed)
	if !ok {
		writeNoMatch(w)
		return
	}
	writeJSON(w, http.StatusOK, resolveResponse{
		IP:       parsed.String(),
		Profile:  res.Profile,
		Hostname: res.Hostname,
		Source:   res.Source,
	})
}

// handleInventory serves the configured sources (entries + hostname rules)
// for the overview UI.
func (s *Server) handleInventory(w http.ResponseWriter, _ *http.Request) {
	if s.opts.Engine == nil {
		writeJSON(w, http.StatusOK, inventory.EmptySnapshot())
		return
	}
	writeJSON(w, http.StatusOK, s.opts.Engine.Snapshot())
}

// resolveResponse is the documented v1 shape — additive-only evolution.
type resolveResponse struct {
	IP       string `json:"ip"`
	Profile  string `json:"profile"`
	Hostname string `json:"hostname,omitempty"`
	Source   string `json:"source"`
}

// bearerAuth enforces the optional INVENTORY_TOKEN check.
func bearerAuth(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			const prefix = "Bearer "
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, prefix) ||
				subtle.ConstantTimeCompare([]byte(strings.TrimPrefix(auth, prefix)), []byte(token)) != 1 {
				log.WithField("remote", r.RemoteAddr).Warn("resolve request rejected: missing/invalid bearer token")
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
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
