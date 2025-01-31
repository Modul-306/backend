package db

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
)

func getConnString() (string, error) {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	if host == "" || port == "" || user == "" || password == "" || dbname == "" {
		return "", fmt.Errorf("missing environment variables")
	}

	return fmt.Sprintf("postgresql://%s:%s/%s?user=%s&password=%s", host, port, dbname, user, password), nil
}

func CreateDBConnection() (*pgx.Conn, error) {
	connString, err := getConnString()
	if err != nil {
		return nil, err
	}
	return pgx.Connect(context.Background(), connString)
}
