package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

type mockStorage struct{}

func (m *mockStorage) UploadFile(ctx context.Context, bucket, key string, body io.Reader) (string, error) {
	return "mock-url", nil
}

func TestLogin(t *testing.T) {
	server := NewServer(nil, &mockStorage{}, "secret")
	r := server.Routes()

	req, _ := http.NewRequest("POST", "/api/v1/auth/login", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)

	var resp map[string]string
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, "dummy-token", resp["token"])
}
