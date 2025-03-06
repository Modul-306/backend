// filepath: /handlers/blogs_test.go
package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Modul-306/backend/auth"
	"github.com/Modul-306/backend/db"
	"github.com/Modul-306/backend/handlers"
	"github.com/Modul-306/backend/router"
	"github.com/Modul-306/backend/tests/containers"
	"github.com/Modul-306/backend/tests/testhelpers"
	"github.com/stretchr/testify/assert"
)

func TestGetBlogs(t *testing.T) {
	// Setup container
	postgres, err := containers.NewTestPostgres(t)
	if err != nil {
		t.Fatalf("failed to create test container: %v", err)
	}
	defer postgres.Cleanup(t)

	// Setup database
	conn, err := db.CreateDBConnection()
	if err != nil {
		t.Fatalf("failed to create db connection: %v", err)
		return
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
			name: "CreateBlog",
			setup: func() *http.Request {
				test := handlers.BlogRequest{
					Title:   "test",
					Content: "test",
					Path:    "/test",
				}
				body, _ := json.Marshal(test)
				req := httptest.NewRequest("POST", "/api/v1/blogs", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				req.AddCookie(authCookie)
				return req
			},
			wantCode: http.StatusCreated,
			validator: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var blog handlers.BlogResponse
				fmt.Println(rec.Body)
				assert.NoError(t, json.NewDecoder(rec.Body).Decode(&blog))
				assert.Equal(t, "test", blog.Title)
			},
		},
		{
			name: "GetBlogs",
			setup: func() *http.Request {
				req := httptest.NewRequest("GET", "/api/v1/blogs", nil)
				return req
			},
			wantCode: http.StatusOK,
			validator: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var blogs []handlers.BlogResponse
				assert.NoError(t, json.NewDecoder(rec.Body).Decode(&blogs))
			},
		},
		{
			name: "GetBlog",
			setup: func() *http.Request {
				req := httptest.NewRequest("GET", "/api/v1/blogs/1", nil)
				return req
			},
			wantCode: http.StatusOK,
			validator: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var blog handlers.BlogResponse
				assert.NoError(t, json.NewDecoder(rec.Body).Decode(&blog))
				assert.Equal(t, "test", blog.Title)
			},
		},
		{
			name: "UpdateBlog",
			setup: func() *http.Request {
				blog := handlers.BlogRequest{
					Title:   "Updated Blog",
					Content: "Updated Content",
					Path:    "/updated-blog",
				}
				body, _ := json.Marshal(blog)
				req := httptest.NewRequest("UPDATE", "/api/v1/blogs/1", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				req.AddCookie(authCookie)
				return req
			},
			wantCode: http.StatusOK,
			validator: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var blog handlers.BlogResponse
				assert.NoError(t, json.NewDecoder(rec.Body).Decode(&blog))
				assert.Equal(t, "Updated Blog", blog.Title)
			},
		},
		{
			name: "DeleteBlog",
			setup: func() *http.Request {
				req := httptest.NewRequest("DELETE", "/api/v1/blogs/1", nil)
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
