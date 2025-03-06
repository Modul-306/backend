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
)

func TestUserHandlers(t *testing.T) {
	postgres, err := containers.NewTestPostgres(t)
	if err != nil {
		t.Fatalf("failed to create test container: %v", err)
	}
	defer postgres.Cleanup(t)

	// Use container's connection
	conn, err := pgx.Connect(context.Background(), postgres.URI)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer conn.Close(context.Background())

	// Setup test database
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
			name: "GetUser",
			setup: func() *http.Request {
				req := httptest.NewRequest("GET", "/api/v1/user/1", nil)
				req.AddCookie(authCookie)
				return req
			},
			wantCode: http.StatusOK,
			validator: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var user handlers.UserResponse
				if err := json.NewDecoder(rec.Body).Decode(&user); err != nil {
					t.Errorf("failed to decode response: %v", err)
					t.Errorf("response body: %s", rec.Body.String())
				}
				if user.Name != "testuser" {
					t.Errorf("got user name %s, want testuser", user.Name)
				}
			},
		},
		{
			name: "UpdateUser",
			setup: func() *http.Request {
				update := handlers.UserRequest{
					Name:  "updated",
					Email: "updated@example.com",
				}
				body, _ := json.Marshal(update)
				req := httptest.NewRequest("UPDATE", "/api/v1/user/1", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				req.AddCookie(authCookie)
				return req
			},
			wantCode: http.StatusOK,
		},
		{
			name: "DeleteUser",
			setup: func() *http.Request {
				req := httptest.NewRequest("DELETE", "/api/v1/user/1", nil)
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
