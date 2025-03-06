package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Modul-306/backend/auth"
	"github.com/Modul-306/backend/handlers"
	"github.com/Modul-306/backend/router"
	"github.com/Modul-306/backend/tests/containers"
	"github.com/Modul-306/backend/tests/testhelpers"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
)

func TestProductHandlers(t *testing.T) {
	// Setup container
	postgres, err := containers.NewTestPostgres(t)
	if err != nil {
		t.Fatalf("failed to create test container: %v", err)
	}
	defer postgres.Cleanup(t)

	// Setup database connection
	conn, err := pgx.Connect(context.Background(), postgres.URI)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer conn.Close(context.Background())

	testhelpers.SetupTestDB(t, conn)
	defer testhelpers.CleanupTestDB(t, conn)

	// Create test user and get token
	hashedPassword, _ := auth.HashPassword("testpass")
	_, err = conn.Exec(context.Background(), `
        INSERT INTO users (name, password, email, is_admin) 
        VALUES ($1, $2, $3, $4)
        RETURNING id`, "testuser", hashedPassword, "test@example.com", true)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	token, err := auth.CreateToken("testuser", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("failed to create auth token: %v", err)
	}

	authCookie := &http.Cookie{
		Name:  "token",
		Value: token,
	}

	tests := []struct {
		name      string
		setup     func() *http.Request
		wantCode  int
		validator func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "GetProducts",
			setup: func() *http.Request {
				return httptest.NewRequest("GET", "/api/v1/products", nil)
			},
			wantCode: http.StatusOK,
			validator: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var products []handlers.ProductResponse
				assert.NoError(t, json.NewDecoder(rec.Body).Decode(&products))
			},
		},
		{
			name: "CreateProduct",
			setup: func() *http.Request {
				product := handlers.ProductRequest{
					Name:        "New Product",
					Price:       29.99,
					ImageURL:    "test.jpg",
					IsAvailable: true,
				}
				body, _ := json.Marshal(product)
				req := httptest.NewRequest("POST", "/api/v1/products", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				req.AddCookie(authCookie)
				return req
			},
			wantCode: http.StatusCreated,
			validator: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var product handlers.ProductResponse
				assert.NoError(t, json.NewDecoder(rec.Body).Decode(&product))
				assert.Equal(t, "New Product", product.Name)
			},
		},
		{
			name: "GetProduct",
			setup: func() *http.Request {
				return httptest.NewRequest("GET", "/api/v1/products/1", nil)
			},
			wantCode: http.StatusOK,
			validator: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var product handlers.ProductResponse
				assert.NoError(t, json.NewDecoder(rec.Body).Decode(&product))
				assert.Equal(t, "New Product", product.Name)
			},
		},
		{
			name: "UpdateProduct",
			setup: func() *http.Request {
				product := handlers.ProductRequest{
					Name:        "Updated Product",
					Price:       39.99,
					ImageURL:    "updated.jpg",
					IsAvailable: true,
				}
				body, _ := json.Marshal(product)
				req := httptest.NewRequest("UPDATE", "/api/v1/products/1", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				req.AddCookie(authCookie)
				return req
			},
			wantCode: http.StatusOK,
			validator: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var product handlers.ProductResponse
				assert.NoError(t, json.NewDecoder(rec.Body).Decode(&product))
				assert.Equal(t, "Updated Product", product.Name)
			},
		},
		{
			name: "DeleteProduct",
			setup: func() *http.Request {
				req := httptest.NewRequest("DELETE", "/api/v1/products/1", nil)
				req.AddCookie(authCookie)
				return req
			},
			wantCode: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setup()
			rec := httptest.NewRecorder()

			sut := router.CreateRouter()

			// Act
			// changed act - calling GetById through production router
			sut.ServeHTTP(rec, req)

			if rec.Code != tt.wantCode {
				t.Errorf("%s status = %v, want %v", tt.name, rec.Code, tt.wantCode)
			}

			if tt.validator != nil {
				tt.validator(t, rec)
			}
		})
	}
}
