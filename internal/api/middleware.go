package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	UserContextKey   contextKey = "user"
	TenantContextKey contextKey = "tenant"
)

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

		ctx := context.WithValue(r.Context(), UserContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
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
