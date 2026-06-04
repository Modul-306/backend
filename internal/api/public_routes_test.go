package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/M306/backend/internal/db/sqlc"
	"github.com/stretchr/testify/require"
)

func TestPublicRoutes(t *testing.T) {
	ctx := context.Background()

	// 1. Create a tenant
	_, err := testQueries.CreateTenant(ctx, db.CreateTenantParams{
		Name: "Test Farm",
		Slug: "testfarm",
	})
	require.NoError(t, err)

	server := NewServer(testQueries, testDB, &mockStorage{}, &MockEmailService{}, "secret", "", "")
	r := server.Routes()

	t.Run("GET products without token", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/products", nil)
		req.Header.Set("X-Tenant-Slug", "testfarm")
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		// Expected to succeed (200 OK) since it's a public route
		require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	})

	t.Run("GET blogs without token", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/blogs", nil)
		req.Header.Set("X-Tenant-Slug", "testfarm")
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		// Expected to succeed (200 OK) since it's a public route
		require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	})

	t.Run("GET products with invalid token", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/products", nil)
		req.Header.Set("X-Tenant-Slug", "testfarm")
		req.Header.Set("Authorization", "Bearer invalidtoken")
		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		// Should still succeed (200 OK) because products route doesn't require authentication!
		require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	})
}
