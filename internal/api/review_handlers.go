package api

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/M306/backend/internal/db/sqlc"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (s *Server) handleCreateReview(w http.ResponseWriter, r *http.Request) {
	productID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "Invalid product ID")
		return
	}

	claims := r.Context().Value(UserContextKey).(UserClaims)
	userID, _ := uuid.Parse(claims.UserID)

	var req struct {
		Rating  int32  `json:"rating"`
		Comment string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "Invalid input")
		return
	}

	review, err := s.db.CreateReview(r.Context(), db.CreateReviewParams{
		ProductID: productID,
		UserID:    userID,
		Rating:    req.Rating,
		Comment:   sql.NullString{String: req.Comment, Valid: req.Comment != ""},
	})
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to create review")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(review)
}

func (s *Server) handleListReviews(w http.ResponseWriter, r *http.Request) {
	productID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "Invalid product ID")
		return
	}

	reviews, err := s.db.ListReviewsByProduct(r.Context(), productID)
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to list reviews")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reviews)
}

func (s *Server) handleGetReviewStats(w http.ResponseWriter, r *http.Request) {
	productID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "Invalid product ID")
		return
	}

	stats, err := s.db.GetAverageRating(r.Context(), productID)
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to get review stats")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
