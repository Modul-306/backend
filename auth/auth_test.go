package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Modul-306/backend/auth"
	"github.com/Modul-306/backend/router"
	"github.com/Modul-306/backend/tests/containers"
	"github.com/Modul-306/backend/tests/testhelpers"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
)

func TestAuthHandlers(t *testing.T) {
	postgres, err := containers.NewTestPostgres(t)
	if err != nil {
		t.Fatalf("failed to create test container: %v", err)
	}
	defer postgres.Cleanup(t)

	conn, err := pgx.Connect(context.Background(), postgres.URI)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer conn.Close(context.Background())

	testhelpers.SetupTestDB(t, conn)
	defer testhelpers.CleanupTestDB(t, conn)

	tests := []struct {
		name      string
		setup     func() *http.Request
		wantCode  int
		validator func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "SignUp",
			setup: func() *http.Request {
				creds := auth.SignUpCredentials{
					Username: "newuser",
					Password: "password123",
					Email:    "new@example.com",
				}
				body, _ := json.Marshal(creds)
				req := httptest.NewRequest("POST", "/api/v1/auth/sign-up", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				return req
			},
			wantCode: http.StatusCreated,
			validator: func(t *testing.T, rec *httptest.ResponseRecorder) {
				cookies := rec.Result().Cookies()
				hasToken := false
				for _, cookie := range cookies {
					if cookie.Name == "token" {
						hasToken = true
						assert.NotEmpty(t, cookie.Value)
					}
				}
				assert.True(t, hasToken, "token cookie not found")
			},
		},
		{
			name: "Login",
			setup: func() *http.Request {
				// Create test user first
				hashedPassword, _ := auth.HashPassword("testpass")
				_, err := conn.Exec(context.Background(), `
                    INSERT INTO users (name, password, email) 
                    VALUES ($1, $2, $3)
                `, "testuser", hashedPassword, "test@example.com")
				if err != nil {
					t.Fatalf("failed to create test user: %v", err)
				}

				creds := auth.Credentials{
					Username: "testuser",
					Password: "testpass",
				}
				body, _ := json.Marshal(creds)
				req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				return req
			},
			wantCode: http.StatusOK,
			validator: func(t *testing.T, rec *httptest.ResponseRecorder) {
				cookies := rec.Result().Cookies()
				var hasToken bool
				for _, cookie := range cookies {
					if cookie.Name == "token" {
						hasToken = true
						assert.NotEmpty(t, cookie.Value)
					}
				}
				assert.True(t, hasToken, "token cookie not found")
			},
		},
		{
			name: "Invalid Login",
			setup: func() *http.Request {
				creds := auth.Credentials{
					Username: "testuser",
					Password: "wrongpass",
				}
				body, _ := json.Marshal(creds)
				req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				return req
			},
			wantCode: http.StatusUnauthorized,
		},
		{
			name: "Protected Endpoint",
			setup: func() *http.Request {
				req := httptest.NewRequest("GET", "/api/v1/order", nil)
				token, _ := auth.CreateToken("testuser", time.Now().Add(time.Hour))
				req.AddCookie(&http.Cookie{
					Name:  "token",
					Value: token,
				})
				return req
			},
			wantCode: http.StatusOK,
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

func TestPassword(t *testing.T) {
	password := "testpass"
	hash, err := auth.HashPassword(password)
	assert.NoError(t, err)
	assert.True(t, auth.VerifyPassword(password, hash))
	assert.False(t, auth.VerifyPassword("wrongpass", hash))
}

func TestToken(t *testing.T) {
	username := "testuser"
	expiry := time.Now().Add(time.Hour)
	token, err := auth.CreateToken(username, expiry)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: token})
	assert.Equal(t, username, auth.GetUsername(req))
}
