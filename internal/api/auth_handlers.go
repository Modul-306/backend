package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/M306/backend/internal/db/sqlc"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "Invalid input")
		return
	}

	if req.Email == "" || req.Password == "" {
		s.errorResponse(w, r, http.StatusBadRequest, "Email and password are required")
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to process password")
		return
	}

	arg := db.CreateUserParams{
		TenantID:     uuid.NullUUID{},
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Role:         "customer",
	}

	user, err := s.db.CreateUser(r.Context(), arg)
	if err != nil {
		if strings.Contains(err.Error(), "unique constraint") {
			s.errorResponse(w, r, http.StatusConflict, "Email already in use")
			return
		}
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to create user")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "Invalid input")
		return
	}

	user, err := s.db.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		s.errorResponse(w, r, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		s.errorResponse(w, r, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID.String(),
		"email":   user.Email,
		"tenant_id": func() string {
			if user.TenantID.Valid {
				return user.TenantID.UUID.String()
			}
			return ""
		}(),
		"role": user.Role,
		"exp":  time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token":   tokenString,
		"role":    user.Role,
		"user_id": user.ID.String(),
	})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(UserContextKey).(UserClaims)
	if !ok {
		s.errorResponse(w, r, http.StatusUnauthorized, "Unauthorized")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(claims)
}
