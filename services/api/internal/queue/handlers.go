package queue

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type Handler struct {
	client *Client
}

func NewHandler(client *Client) *Handler {
	return &Handler{client: client}
}

func (h *Handler) CreateJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	defer r.Body.Close()

	var request struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "invalid JSON request", http.StatusBadRequest)
		return
	}

	if request.Type == "" || len(request.Payload) == 0 {
		http.Error(w, "type and payload are required", http.StatusBadRequest)
		return
	}

	job, err := h.client.Enqueue(r.Context(), request.Type, request.Payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeQueueJSON(w, http.StatusAccepted, map[string]interface{}{
		"status": "accepted",
		"job":    job,
	})
}

func (h *Handler) GetJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		http.Error(w, "job ID is required", http.StatusBadRequest)
		return
	}

	job, err := h.client.GetJob(r.Context(), id)
	if errors.Is(err, redis.Nil) {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "failed to retrieve job", http.StatusInternalServerError)
		return
	}

	writeQueueJSON(w, http.StatusOK, job)
}

func writeQueueJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func readinessTimeout() time.Duration {
	return 3 * time.Second
}
