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

func TestOrderHandlers(t *testing.T) {
	postgres, err := containers.NewTestPostgres(t)
	if err != nil {
		t.Fatalf("failed to create test container: %v", err)
	}
	defer postgres.Cleanup(t)

	// Setup database connection using container URI
	conn, err := pgx.Connect(context.Background(), postgres.URI)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer conn.Close(context.Background())

	testhelpers.SetupTestDB(t, conn)
	defer testhelpers.CleanupTestDB(t, conn)

	// Setup test user and get auth token
	_, err = conn.Exec(context.Background(), `
        INSERT INTO users (name, password, email) 
        VALUES ('testuser', 'password', 'test@example.com')
    `)
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
			name: "GetOrders",
			setup: func() *http.Request {
				req := httptest.NewRequest("GET", "/api/v1/order", nil)
				req.AddCookie(authCookie)
				return req
			},
			wantCode: http.StatusOK,
			validator: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var orders []handlers.OrderResponse
				assert.NoError(t, json.NewDecoder(rec.Body).Decode(&orders))
			},
		},
		{
			name: "CreateOrder",
			setup: func() *http.Request {
				order := handlers.OrderRequest{
					Address:     "123 Test St",
					IsCompleted: false,
				}
				body, _ := json.Marshal(order)
				req := httptest.NewRequest("POST", "/api/v1/order", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				req.AddCookie(authCookie)
				return req
			},
			wantCode: http.StatusCreated,
			validator: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var order handlers.OrderResponse
				assert.NoError(t, json.NewDecoder(rec.Body).Decode(&order))
				assert.Equal(t, "123 Test St", order.Address)
			},
		},
		{
			name: "GetOrder",
			setup: func() *http.Request {
				req := httptest.NewRequest("GET", "/api/v1/order/1", nil)
				req.AddCookie(authCookie)
				return req
			},
			wantCode: http.StatusOK,
			validator: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var order handlers.OrderResponse
				assert.NoError(t, json.NewDecoder(rec.Body).Decode(&order))
				assert.Equal(t, "123 Test St", order.Address)
			},
		},
		{
			name: "UpdateOrder",
			setup: func() *http.Request {
				order := handlers.OrderRequest{
					Address:     "456 Updated St",
					IsCompleted: true,
				}
				body, _ := json.Marshal(order)
				req := httptest.NewRequest("UPDATE", "/api/v1/order/1", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				req.AddCookie(authCookie)
				return req
			},
			wantCode: http.StatusOK,
			validator: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var order handlers.OrderResponse
				assert.NoError(t, json.NewDecoder(rec.Body).Decode(&order))
				assert.Equal(t, "456 Updated St", order.Address)
				assert.True(t, order.IsCompleted)
			},
		},
		{
			name: "DeleteOrder",
			setup: func() *http.Request {
				req := httptest.NewRequest("DELETE", "/api/v1/order/1", nil)
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
