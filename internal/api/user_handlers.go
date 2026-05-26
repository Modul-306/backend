package api

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.db.ListUsers(r.Context())
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to list users")
		return
	}
	type safeUser struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	result := make([]safeUser, 0)
	for _, u := range users {
		result = append(result, safeUser{
			ID:    u.ID.String(),
			Email: u.Email,
			Role:  u.Role,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleGetUserLoyalty(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value(UserContextKey).(UserClaims)
	userID, _ := uuid.Parse(claims.UserID)
	
	discount, err := s.db.GetUserDiscount(r.Context(), userID)
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to fetch loyalty info")
		return
	}
	
	user, _ := s.db.GetUserByID(r.Context(), userID)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tier": user.LoyaltyTier.String,
		"discount_percent": discount,
	})
}
