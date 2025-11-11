package http

import (
	"Code_executor/internal/domain"
	"Code_executor/internal/repository"
	"Code_executor/internal/service"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"net/http"
	"time"
)

var (
	ErrInvalidArgument = errors.New("invalid argument")
)

type ExecutionHandler struct {
	service service.ExecutionService
}

type createExecutionRequest struct {
	Language  string `json:"language"`
	Code      string `json:"code"`
	TimeoutMs int    `json:"timeout_ms"`
	Stdin     string `json:"stdin"`
	UserName  string `json:"user_name"`
}

type executionResponse struct {
	ID         string                 `json:"id"`
	Language   string                 `json:"language"`
	Status     domain.ExecutionStatus `json:"status"`
	Stdout     string                 `json:"stdout"`
	Stderr     string                 `json:"stderr"`
	ExitCode   *int                   `json:"exit_code"`
	TimeoutMs  int                    `json:"timeout_ms"`
	CreatedAt  time.Time              `json:"created_at"`
	StartedAt  *time.Time             `json:"started_at,omitempty"`
	FinishedAt *time.Time             `json:"finished_at,omitempty"`
	UserID     string                 `json:"user_id"`
}

func NewExecutionHandler(s service.ExecutionService) (*ExecutionHandler, error) {
	if s == nil {
		return nil, fmt.Errorf("%w: service is nil", ErrInvalidArgument)
	}

	return &ExecutionHandler{
		service: s,
	}, nil
}

func (h *ExecutionHandler) RegisterRoutes(r chi.Router) {
	r.Post("/executions", h.handleCreateExecution)
	r.Get("/executions/{executionID}", h.handleGetExecution)
}

func newExecutionResponse(exec *domain.Execution) executionResponse {
	if exec == nil {
		return executionResponse{}
	}

	return executionResponse{
		ID:         exec.ID,
		Language:   exec.Language,
		Status:     exec.Status,
		Stdout:     exec.Stdout,
		Stderr:     exec.Stderr,
		ExitCode:   exec.ExitCode,
		TimeoutMs:  exec.TimeoutMs,
		CreatedAt:  exec.CreatedAt.UTC(),
		StartedAt:  normalizeTimePtr(exec.StartedAt),
		FinishedAt: normalizeTimePtr(exec.FinishedAt),
		UserID:     exec.UserID,
	}
}

func normalizeTimePtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}

	ts := t.UTC()
	return &ts
}

func (h *ExecutionHandler) handleCreateExecution(w http.ResponseWriter, r *http.Request) {
	var req createExecutionRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	if err := validateCreateExecutionRequest(req); err != nil {
		writeServiceError(w, err)
		return
	}

	params := service.CreateExecutionParams{
		Language:  req.Language,
		Code:      req.Code,
		Stdin:     req.Stdin,
		TimeoutMs: req.TimeoutMs,
		UserID:    req.UserName,
	}

	exec, err := h.service.CreateExecutionAndEnqueue(r.Context(), params)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, newExecutionResponse(exec))
}

func (h *ExecutionHandler) handleGetExecution(w http.ResponseWriter, r *http.Request) {
	executionID := chi.URLParam(r, "executionID")
	if executionID == "" {
		writeServiceError(w, fmt.Errorf("%w: executionID is required", ErrInvalidArgument))
		return
	}

	exec, err := h.service.GetExecution(r.Context(), executionID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newExecutionResponse(exec))
}

func validateCreateExecutionRequest(req createExecutionRequest) error {
	if req.Language == "" {
		return fmt.Errorf("%w: language is required", ErrInvalidArgument)
	}

	if req.Code == "" {
		return fmt.Errorf("%w: code is required", ErrInvalidArgument)
	}

	if req.TimeoutMs <= 0 {
		return fmt.Errorf("%w: timeout_ms must be positive", ErrInvalidArgument)
	}

	if req.UserName == "" {
		return fmt.Errorf("%w: user_name is required", ErrInvalidArgument)
	}

	return nil
}

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

func writeServiceError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	message := "internal server error"

	switch {
	case errors.Is(err, ErrInvalidArgument):
		status = http.StatusBadRequest
		message = err.Error()
	case errors.Is(err, service.ErrInvalidServiceInput):
		status = http.StatusBadRequest
		message = err.Error()
	case errors.Is(err, domain.ErrInvalidExecution):
		status = http.StatusBadRequest
		message = err.Error()
	case errors.Is(err, repository.ErrExecutionNotFound):
		status = http.StatusNotFound
		message = err.Error()
	}

	writeError(w, status, message)
}
