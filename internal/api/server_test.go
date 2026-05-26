package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/M306/backend/internal/db/sqlc"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var testQueries *db.Queries
var testDB *sql.DB

func TestMain(m *testing.M) {
	ctx := context.Background()

	migrationsDir := filepath.Join("..", "..", "migrations")
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		log.Fatalf("failed to read migrations directory: %s", err)
	}

	var initScripts []string
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".sql" && (len(file.Name()) > 6 && file.Name()[len(file.Name())-6:] == "up.sql") {
			initScripts = append(initScripts, filepath.Join(migrationsDir, file.Name()))
		}
	}

	pgContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("cattlehof_test"),
		postgres.WithUsername("user"),
		postgres.WithPassword("password"),
		postgres.WithInitScripts(initScripts...),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
	)
	if err != nil {
		log.Fatalf("failed to start container: %s", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatalf("failed to get connection string: %s", err)
	}

	testDB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("failed to connect to database: %s", err)
	}

	testQueries = db.New(testDB)

	code := m.Run()

	testDB.Close()
	pgContainer.Terminate(ctx)
	os.Exit(code)
}

type mockStorage struct{}

func (m *mockStorage) UploadFile(ctx context.Context, bucket, key string, body io.Reader) (string, error) {
	return "mock-url", nil
}
func TestLogin(t *testing.T) {
	server := NewServer(testQueries, testDB, &mockStorage{}, &MockEmailService{}, "secret")
	r := server.Routes()

	body, _ := json.Marshal(map[string]string{
		"email":    "nonexistent@example.com",
		"password": "password",
	})
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	// Since the user doesn't exist, we expect 401 Unauthorized, not a 500 Panic
	require.Equal(t, http.StatusUnauthorized, rr.Code)
}
