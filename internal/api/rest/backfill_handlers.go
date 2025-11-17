package rest

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/fortuna/minerva/internal/backfill"
)

// BackfillHandler proxies API calls to the backfill service.
type BackfillHandler struct {
	service *backfill.Service
}

// NewBackfillHandler wires the REST layer to the backfill service.
func NewBackfillHandler(service *backfill.Service) *BackfillHandler {
	return &BackfillHandler{service: service}
}

type apiBackfillRequest struct {
	Sport     string   `json:"sport"`
	SeasonID  string   `json:"season_id"`
	StartDate string   `json:"start_date"`
	EndDate   string   `json:"end_date"`
	GameID    string   `json:"game_id"`
	GameIDs   []string `json:"game_ids"`
	DryRun    bool     `json:"dry_run"`
}

// HandleBackfillRequest handles POST /api/v1/backfill
func (h *BackfillHandler) HandleBackfillRequest(w http.ResponseWriter, r *http.Request) {
	var req apiBackfillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	backfillReq := backfill.Request{
		Sport:    req.Sport,
		SeasonID: req.SeasonID,
		DryRun:   req.DryRun,
	}

	if len(req.GameIDs) > 0 {
		backfillReq.GameIDs = append(backfillReq.GameIDs, req.GameIDs...)
	}
	if req.GameID != "" {
		backfillReq.GameIDs = append(backfillReq.GameIDs, req.GameID)
	}

	if req.StartDate != "" {
		start, err := time.Parse("2006-01-02", req.StartDate)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid start_date format (YYYY-MM-DD)", err)
			return
		}
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
		backfillReq.StartDate = &start
	}

	if req.EndDate != "" {
		end, err := time.Parse("2006-01-02", req.EndDate)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid end_date format (YYYY-MM-DD)", err)
			return
		}
		end = time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, time.UTC)
		backfillReq.EndDate = &end
	}

	job, err := h.service.Enqueue(r.Context(), backfillReq)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Failed to enqueue backfill job", err)
		return
	}

	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"job": jobPayload(job),
	})
}

// HandleBackfillStatus handles GET /api/v1/backfill/status
func (h *BackfillHandler) HandleBackfillStatus(w http.ResponseWriter, r *http.Request) {
	summary, err := h.service.GetStatus(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch status", err)
		return
	}

	payload := buildStatusPayload(summary)
	respondJSON(w, http.StatusOK, payload)
}

func buildStatusPayload(summary *backfill.StatusSummary) map[string]interface{} {
	response := map[string]interface{}{
		"status":  "idle",
		"message": "No active jobs",
		"history": []map[string]interface{}{},
	}

	if summary.ActiveJob != nil {
		response["status"] = summary.ActiveJob.Status
		if summary.ActiveJob.StatusMessage.Valid {
			response["message"] = summary.ActiveJob.StatusMessage.String
		}
		response["active_job"] = jobPayload(summary.ActiveJob)
	}

	history := make([]map[string]interface{}, 0, len(summary.History))
	for _, job := range summary.History {
		history = append(history, jobPayload(job))
	}

	response["history"] = history
	return response
}

func jobPayload(job *backfill.Job) map[string]interface{} {
	if job == nil {
		return nil
	}

	payload := map[string]interface{}{
		"job_id":           job.JobID,
		"job_type":         job.JobType,
		"status":           job.Status,
		"progress_current": job.ProgressCurrent,
		"progress_total":   job.ProgressTotal,
		"created_at":       job.CreatedAt,
		"updated_at":       job.UpdatedAt,
	}

	if job.StatusMessage.Valid {
		payload["status_message"] = job.StatusMessage.String
	}
	if job.SeasonID.Valid {
		payload["season_id"] = job.SeasonID.String
	}
	if job.StartDate.Valid {
		payload["start_date"] = job.StartDate.Time.Format("2006-01-02")
	}
	if job.EndDate.Valid {
		payload["end_date"] = job.EndDate.Time.Format("2006-01-02")
	}
	if len(job.GameIDs) > 0 {
		payload["game_ids"] = job.GameIDs
	}
	if job.StartedAt.Valid {
		payload["started_at"] = job.StartedAt.Time
	}
	if job.CompletedAt.Valid {
		payload["completed_at"] = job.CompletedAt.Time
	}
	if job.LastError.Valid {
		payload["last_error"] = job.LastError.String
	}

	return payload
}
