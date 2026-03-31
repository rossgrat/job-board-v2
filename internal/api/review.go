package api

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rossgrat/job-board-v2/database/gen/db"
	"github.com/rossgrat/job-board-v2/internal/api/templates"
)

func (s *Server) handleReviewModal(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	queries := db.New(s.pool)
	ctx := r.Context()

	cj, err := queries.GetClassifiedJobByID(ctx, id)
	if err != nil {
		slog.Error("failed to get classified job", slog.String("err", err.Error()))
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	rj, err := queries.GetRawJobByID(ctx, cj.RawJobID)
	if err != nil {
		slog.Error("failed to get raw job", slog.String("err", err.Error()))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	s.renderReviewModal(w, r, cj, rj, queries)
}

func (s *Server) renderReviewModal(w http.ResponseWriter, r *http.Request, cj db.ClassifiedJob, rj db.RawJob, queries *db.Queries) {
	modal := templates.ReviewModal{
		ClassifiedJobID: uuidToString(cj.ID),
		RawJobID:        uuidToString(rj.ID),
		Title:           textOrEmpty(cj.Title),
		UserStatus:      textOrEmpty(rj.UserStatus),
		ModelCategory:   textOrEmpty(cj.Category),
		ModelRelevance:  textOrEmpty(cj.Relevance),
	}

	eval, err := queries.GetEvalEntryByRawJobID(r.Context(), rj.ID)
	if err == nil {
		modal.HasEval = true
		modal.EvalCategory = eval.ExpectedCategory
		modal.EvalRelevance = textOrEmpty(eval.ExpectedRelevance)
	} else {
		modal.EvalCategory = textOrEmpty(cj.Category)
		modal.EvalRelevance = textOrEmpty(cj.Relevance)
	}

	templates.ReviewModalContent(modal).Render(r.Context(), w)
}

func (s *Server) handleSetStatus(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	status := r.FormValue("status")

	queries := db.New(s.pool)
	ctx := r.Context()

	cj, err := queries.GetClassifiedJobByID(ctx, id)
	if err != nil {
		slog.Error("failed to get classified job", slog.String("err", err.Error()))
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	var userStatus pgtype.Text
	if status != "" {
		userStatus = pgtype.Text{String: status, Valid: true}
	}

	err = queries.SetUserStatus(ctx, db.SetUserStatusParams{
		ID:         cj.RawJobID,
		UserStatus: userStatus,
	})
	if err != nil {
		slog.Error("failed to set user status", slog.String("err", err.Error()))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Re-render the modal with updated status
	rj, err := queries.GetRawJobByID(ctx, cj.RawJobID)
	if err != nil {
		slog.Error("failed to get raw job", slog.String("err", err.Error()))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	s.renderReviewModal(w, r, cj, rj, queries)
}

func (s *Server) handleSetEval(w http.ResponseWriter, r *http.Request) {
	rawJobID, err := parseUUID(r.FormValue("raw_job_id"))
	if err != nil {
		http.Error(w, "Invalid raw_job_id", http.StatusBadRequest)
		return
	}

	expectedCategory := r.FormValue("expected_category")
	expectedRelevance := r.FormValue("expected_relevance")

	var relevance pgtype.Text
	if expectedRelevance != "" {
		relevance = pgtype.Text{String: expectedRelevance, Valid: true}
	}

	queries := db.New(s.pool)
	newID := uuid.New()

	err = queries.UpsertEvalEntry(r.Context(), db.UpsertEvalEntryParams{
		ID:                pgtype.UUID{Bytes: newID, Valid: true},
		RawJobID:          rawJobID,
		ExpectedCategory:  expectedCategory,
		ExpectedRelevance: relevance,
	})
	if err != nil {
		slog.Error("failed to upsert eval entry", slog.String("err", err.Error()))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
