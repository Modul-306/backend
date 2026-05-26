package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/M306/backend/internal/db/sqlc"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type contextKey string

const (
	UserContextKey   contextKey = "user"
	TenantContextKey contextKey = "tenant"
)

type UserClaims struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	TenantID string `json:"tenant_id"`
	Role     string `json:"role"`
}

func (s *Server) JWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(s.jwtSecret), nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		userClaims := UserClaims{
			UserID: claims["user_id"].(string),
			Role:   claims["role"].(string),
		}
		if email, ok := claims["email"].(string); ok {
			userClaims.Email = email
		}
		if tid, ok := claims["tenant_id"].(string); ok {
			userClaims.TenantID = tid
		}

		ctx := context.WithValue(r.Context(), UserContextKey, userClaims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(UserContextKey).(UserClaims)
			if !ok {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			authorized := false
			for _, role := range roles {
				if claims.Role == role {
					authorized = true
					break
				}
			}

			if !authorized {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (s *Server) RequireTenantAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(UserContextKey).(UserClaims)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Platform admins can access any tenant
		if claims.Role == "platform_admin" {
			next.ServeHTTP(w, r)
			return
		}

		tenant, ok := r.Context().Value(TenantContextKey).(db.Tenant)
		if !ok {
			http.Error(w, "Tenant context missing", http.StatusInternalServerError)
			return
		}

		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			http.Error(w, "Invalid user ID in claims", http.StatusUnauthorized)
			return
		}

		isOwner, err := s.db.IsTenantOwner(r.Context(), db.IsTenantOwnerParams{
			TenantID: tenant.ID,
			UserID:   userID,
		})
		if err != nil {
			http.Error(w, "Error checking permissions", http.StatusInternalServerError)
			return
		}

		if !isOwner {
			http.Error(w, "Forbidden: Access denied to this tenant", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) TenantMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Priority: Header (set by frontend based on URL path)
		slug := r.Header.Get("X-Tenant-Slug")
		
		if slug == "" {
			// Fallback: Hostname
			host := r.Host
			parts := strings.Split(host, ".")
			if len(parts) > 1 && parts[0] != "localhost" {
				slug = parts[0]
			}
		}

		if slug == "" {
			slug = "default"
		}

		tenant, err := s.db.GetTenantBySlug(r.Context(), slug)
		if err != nil {
			http.Error(w, "Tenant not found", http.StatusNotFound)
			return
		}

		ctx := context.WithValue(r.Context(), TenantContextKey, tenant)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
