package config

import (
	"encoding/json"
	"net/http"
)

// Handler wires HTTP routes to ConfigService.
//
// Routes:
//
//	GET  /api/config/remote     — agent polling endpoint (flat JSON)
//	GET  /api/config            — dashboard: get current config
//	PUT  /api/config            — dashboard: replace entire config
//	PATCH /api/config/{key}     — dashboard: update single field (?value=...)
//
// Examples:
//
//	curl http://localhost:18888/api/config/remote
//	curl -X PATCH "http://localhost:18888/api/config/emergencyOff?value=true"
//	curl -X PATCH "http://localhost:18888/api/config/headSampleRate?value=0.5"
//	curl -X PUT http://localhost:18888/api/config \
//	     -H "Content-Type: application/json" \
//	     -d '{"headSampleRate":0.5,"emergencyOff":false,"logInjection":true}'
type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/config/remote", h.getForAgent)
	mux.HandleFunc("GET /api/config", h.get)
	mux.HandleFunc("PUT /api/config", h.put)
	mux.HandleFunc("PATCH /api/config/{key}", h.patch)
}

func (h *Handler) getForAgent(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.service.Get())
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.service.Get())
}

func (h *Handler) put(w http.ResponseWriter, r *http.Request) {
	var cfg RemoteConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	h.service.Replace(cfg)
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *Handler) patch(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	value := r.URL.Query().Get("value")
	if err := h.service.Patch(key, value); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "patched", "key": key, "value": value})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}
