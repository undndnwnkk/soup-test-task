package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/undndnwnkk/soup-test-task/internal/repository"
)

const (
	maxBodyBytes = 1 << 20
	maxKeyLen    = 255
)

type JobStore interface {
	CreateJob(ctx context.Context, key, hash string, payload json.RawMessage) (repository.Job, bool, error)
	GetJob(ctx context.Context, id int64) (repository.Job, error)
}

type Handler struct {
	store JobStore
}

func NewHandler(store JobStore) *Handler {
	return &Handler{store: store}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /jobs", h.createJob)
	mux.HandleFunc("GET /jobs/{id}", h.getJob)
	return mux
}

type createJobRequest struct {
	Payload json.RawMessage `json:"payload"`
}

func (h *Handler) createJob(w http.ResponseWriter, r *http.Request) {
	key := r.Header.Get("Idempotency-Key")
	if key == "" || len(key) > maxKeyLen {
		writeError(w, http.StatusBadRequest, "Idempotency-Key header is required (1..255 chars)")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	var req createJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if len(req.Payload) == 0 || bytes.Equal(req.Payload, []byte("null")) {
		writeError(w, http.StatusBadRequest, "payload is required")
		return
	}

	payload, err := canonicalJSON(req.Payload)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	sum := sha256.Sum256(payload)
	hash := hex.EncodeToString(sum[:])

	job, created, err := h.store.CreateJob(r.Context(), key, hash, payload)
	if err != nil {
		slog.Error("create job", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	switch {
	case created:
		writeJSON(w, http.StatusCreated, job)
	case job.RequestHash != hash:
		writeError(w, http.StatusConflict, "Idempotency-Key reused with a different payload")
	default:
		writeJSON(w, http.StatusOK, job)
	}
}

func (h *Handler) getJob(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	job, err := h.store.GetJob(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}
	if err != nil {
		slog.Error("get job", "error", err, "id", id)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, job)
}

// canonicalJSON re-encodes JSON so that semantically equal payloads produce
// identical bytes: object keys get sorted, whitespace dropped. UseNumber keeps
// large integers from being rounded through float64.
func canonicalJSON(raw json.RawMessage) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return json.Marshal(v)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("write response", "error", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
