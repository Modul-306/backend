// filepath: /tests/testhelpers/helpers.go
package testhelpers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5"
)

// findProjectRoot looks for sql/schema.sql file walking up the directory tree
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "sql", "schema.sql")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find project root")
		}
		dir = parent
	}
}

func SetupTestDB(t *testing.T, conn *pgx.Conn) {
	root, err := findProjectRoot()
	if err != nil {
		t.Fatalf("failed to find project root: %v", err)
	}

	schemaPath := filepath.Join(root, "sql", "schema.sql")
	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("failed to read schema file at %s: %v", schemaPath, err)
	}

	_, err = conn.Exec(context.Background(), string(schema))
	if err != nil {
		t.Fatalf("failed to execute schema: %v", err)
	}
}

func CleanupTestDB(t *testing.T, conn *pgx.Conn) {
	_, err := conn.Exec(context.Background(), `
        DROP TABLE IF EXISTS order_products CASCADE;
        DROP TABLE IF EXISTS orders CASCADE;
        DROP TABLE IF EXISTS products CASCADE;
        DROP TABLE IF EXISTS blogs CASCADE;
        DROP TABLE IF EXISTS users CASCADE;
    `)
	if err != nil {
		t.Fatalf("failed to cleanup database: %v", err)
	}
}
