package db

import (
	"context"
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var testQueries *Queries
var testDB *sql.DB

func TestMain(m *testing.M) {
	ctx := context.Background()

	migrationsDir := filepath.Join("..", "..", "..", "migrations")
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
		postgres.WithDatabase("cattlehof"),
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

	testQueries = New(testDB)

	code := m.Run()

	testDB.Close()
	if err := pgContainer.Terminate(ctx); err != nil {
		log.Fatalf("failed to terminate container: %s", err)
	}

	os.Exit(code)
}
