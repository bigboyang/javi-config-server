package config

import (
	"encoding/json"
	"net/http"
)

// Handler wires HTTP routes to ConfigService.
//
// Routes:
//
//	GET  /api/config/remote              — agent polling (ETag, ?service, ?env)
//	GET  /api/config                     — dashboard: current global config + version
//	PUT  /api/config                     — dashboard: replace global config
//	PATCH /api/config/{key}              — dashboard: update single global field
//	GET  /api/config/services            — list all service overrides
//	GET  /api/config/service/{service}   — get service override (?env=...)
//	PUT  /api/config/service/{service}   — set service override (?env=...)
//	DELETE /api/config/service/{service} — remove service override (?env=...)
//
// Agent polling examples:
//
//	curl http://localhost:18888/api/config/remote
//	curl http://localhost:18888/api/config/remote?service=payment&env=prod
//	curl -H 'If-None-Match: "3"' http://localhost:18888/api/config/remote
//
// Dashboard examples:
//
//	curl -X PATCH "http://localhost:18888/api/config/emergencyOff?value=true"
//	curl -X PATCH "http://localhost:18888/api/config/headSampleRate?value=0.5"
//	curl -X PUT http://localhost:18888/api/config \
//	     -H "Content-Type: application/json" \
//	     -d '{"headSampleRate":0.5,"emergencyOff":false,"logInjection":true,...}'
//	curl -X PUT http://localhost:18888/api/config/service/payment?env=prod \
//	     -H "Content-Type: application/json" \
//	     -d '{"headSampleRate":0.1,...}'
type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Agent polling
	mux.HandleFunc("GET /api/config/remote", h.getForAgent)

	// Global config management
	mux.HandleFunc("GET /api/config", h.get)
	mux.HandleFunc("PUT /api/config", h.put)
	mux.HandleFunc("PATCH /api/config/{key}", h.patch)

	// Per-service override management (must be registered before /{key} catch-all)
	mux.HandleFunc("GET /api/config/services", h.listOverrides)
	mux.HandleFunc("GET /api/config/service/{service}", h.getOverride)
	mux.HandleFunc("PUT /api/config/service/{service}", h.putOverride)
	mux.HandleFunc("DELETE /api/config/service/{service}", h.deleteOverride)
}

// getForAgent handles agent polling with ETag-based conditional GET.
// Agents should send If-None-Match with the last ETag they received.
// Returns 304 Not Modified if config has not changed, avoiding full payload transfer.
func (h *Handler) getForAgent(w http.ResponseWriter, r *http.Request) {
	key := ServiceKey{
		Service: r.URL.Query().Get("service"),
		Env:     r.URL.Query().Get("env"),
	}

	etag := h.service.ETag()
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	cfg, version := h.service.GetEffective(key)
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "no-cache")
	writeJSON(w, http.StatusOK, AgentConfigResponse{
		RemoteConfig:  cfg,
		ConfigVersion: version,
	})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"config":  h.service.Get(),
		"version": h.service.Version(),
		"etag":    h.service.ETag(),
	})
}

func (h *Handler) put(w http.ResponseWriter, r *http.Request) {
	var cfg RemoteConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := cfg.Validate(); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	h.service.Replace(cfg)
	writeJSON(w, http.StatusOK, map[string]any{"status": "updated", "version": h.service.Version()})
}

func (h *Handler) patch(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	value := r.URL.Query().Get("value")
	if err := h.service.Patch(key, value); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "patched",
		"key":     key,
		"value":   value,
		"version": h.service.Version(),
	})
}

func (h *Handler) listOverrides(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.service.ListOverrides())
}

func (h *Handler) getOverride(w http.ResponseWriter, r *http.Request) {
	key := ServiceKey{
		Service: r.PathValue("service"),
		Env:     r.URL.Query().Get("env"),
	}
	cfg, ok := h.service.GetOverride(key)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no override for " + key.String()})
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (h *Handler) putOverride(w http.ResponseWriter, r *http.Request) {
	key := ServiceKey{
		Service: r.PathValue("service"),
		Env:     r.URL.Query().Get("env"),
	}
	var cfg RemoteConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := cfg.Validate(); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	h.service.SetOverride(key, cfg)
	writeJSON(w, http.StatusOK, map[string]any{"status": "set", "service": key.String(), "version": h.service.Version()})
}

func (h *Handler) deleteOverride(w http.ResponseWriter, r *http.Request) {
	key := ServiceKey{
		Service: r.PathValue("service"),
		Env:     r.URL.Query().Get("env"),
	}
	if !h.service.DeleteOverride(key) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no override for " + key.String()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}
