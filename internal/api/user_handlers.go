package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/M306/backend/internal/db/sqlc"
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

func (s *Server) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value(UserContextKey).(UserClaims)
	userID, _ := uuid.Parse(claims.UserID)
	
	user, err := s.db.GetUserByID(r.Context(), userID)
	if err != nil {
		s.errorResponse(w, r, http.StatusNotFound, "User not found")
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         user.ID,
		"email":      user.Email,
		"full_name":  user.FullName.String,
		"street":     user.Street.String,
		"zip_code":   user.ZipCode.String,
		"city":       user.City.String,
		"role":       user.Role,
		"created_at": user.CreatedAt,
	})
}

func (s *Server) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value(UserContextKey).(UserClaims)
	userID, _ := uuid.Parse(claims.UserID)
	
	var req struct {
		FullName string `json:"full_name"`
		Street   string `json:"street"`
		ZipCode  string `json:"zip_code"`
		City     string `json:"city"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "Invalid input")
		return
	}
	
	arg := db.UpdateUserProfileParams{
		ID:       userID,
		FullName: sql.NullString{String: req.FullName, Valid: req.FullName != ""},
		Street:   sql.NullString{String: req.Street, Valid: req.Street != ""},
		ZipCode:  sql.NullString{String: req.ZipCode, Valid: req.ZipCode != ""},
		City:     sql.NullString{String: req.City, Valid: req.City != ""},
	}
	
	user, err := s.db.UpdateUserProfile(r.Context(), arg)
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to update profile")
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func (s *Server) handleGetUserLoyalty(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value(UserContextKey).(UserClaims)
	userID, _ := uuid.Parse(claims.UserID)
	
	user, err := s.db.GetUserByID(r.Context(), userID)
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to fetch user")
		return
	}

	// Recalculate and update user loyalty tier based on number of orders placed in the last 365 days
	userOrders, err := s.db.ListOrdersByUser(r.Context(), userID)
	if err == nil {
		orderCount := 0
		oneYearAgo := time.Now().AddDate(-1, 0, 0)
		for _, o := range userOrders {
			if o.CreatedAt.Valid && o.CreatedAt.Time.After(oneYearAgo) {
				orderCount++
			}
		}
		var newTier string
		if orderCount >= 20 {
			newTier = "Harvest Elite"
		} else if orderCount >= 10 {
			newTier = "Harvester"
		} else if orderCount >= 3 {
			newTier = "Sprout"
		} else {
			newTier = "Seedling"
		}

		if user.LoyaltyTier.String != newTier {
			user, _ = s.db.UpdateUserLoyaltyTier(r.Context(), db.UpdateUserLoyaltyTierParams{
				ID:          userID,
				LoyaltyTier: sql.NullString{String: newTier, Valid: true},
			})
		}
	}

	discount, err := s.db.GetUserDiscount(r.Context(), userID)
	if err != nil {
		if err == sql.ErrNoRows {
			discount = "0.00"
		} else {
			s.errorResponse(w, r, http.StatusInternalServerError, "Failed to fetch loyalty info")
			return
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tier":             user.LoyaltyTier.String,
		"discount_percent": discount,
	})
}
