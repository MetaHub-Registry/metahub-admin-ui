package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/metahub-registry/metahub-admin-ui/internal/inventory"
)

// entryRequest is the body of POST /api/v1/entries — exactly one of ip/cidr,
// non-empty profile, optional display hostname. Validation matches the YAML
// sources (shared validateEntry), so error messages name the field.
type entryRequest struct {
	IP       string `json:"ip"`
	CIDR     string `json:"cidr"`
	Profile  string `json:"profile"`
	Hostname string `json:"hostname"`
}

// handleAddEntry registers one runtime entry (ephemeral overlay — wiped on
// restart). 201 added / 400 invalid (named field) / 409 duplicate / 503
// engine missing.
func (s *Server) handleAddEntry(w http.ResponseWriter, r *http.Request) {
	if s.opts.Engine == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "inventory engine not initialized"})
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 4<<10))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read body: " + err.Error()})
		return
	}
	var req entryRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "malformed JSON body"})
		return
	}

	entry := inventory.Entry{IP: req.IP, CIDR: req.CIDR, Profile: req.Profile, Hostname: req.Hostname}
	if err := s.opts.Engine.AddEntry(entry); err != nil {
		if errors.Is(err, inventory.ErrDuplicate) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	log.WithFields(log.Fields{
		"ip": entry.IP, "cidr": entry.CIDR, "profile": entry.Profile,
	}).Info("runtime entry registered")
	writeJSON(w, http.StatusCreated, map[string]any{
		"status": "registered",
		"note":   "runtime entries are ephemeral — wiped on service restart",
	})
}

// handleDeleteEntry deregisters a runtime entry (?ip= or ?cidr=).
// 204 removed / 404 unknown (with a hint for static YAML entries) / 400
// bad query.
func (s *Server) handleDeleteEntry(w http.ResponseWriter, r *http.Request) {
	if s.opts.Engine == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "inventory engine not initialized"})
		return
	}
	ip, cidr := r.URL.Query().Get("ip"), r.URL.Query().Get("cidr")
	if (ip == "" && cidr == "") || (ip != "" && cidr != "") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "pass exactly one of ?ip= or ?cidr="})
		return
	}
	key := ip
	if cidr != "" {
		key = cidr
	}

	found, staticHit := s.opts.Engine.RemoveEntry(key)
	switch {
	case found:
		log.WithField("key", key).Info("runtime entry removed")
		w.WriteHeader(http.StatusNoContent)
	case staticHit:
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": key + " is a static YAML entry — only runtime entries can be removed",
		})
	default:
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no runtime entry for " + key})
	}
}
