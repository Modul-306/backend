package db

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
)

func getConnString() (string, error) {
	host, isSet := os.LookupEnv("DB_HOST")
	if !isSet {
		fmt.Println("DB_HOST is empty")
	}

	port, isSet := os.LookupEnv("DB_PORT")
	if !isSet {
		fmt.Println("DB_PORT is empty")
	}

	user, isSet := os.LookupEnv("DB_USER")
	if !isSet {
		fmt.Println("DB_USER is empty")
	}

	password, isSet := os.LookupEnv("DB_PASSWORD")
	if !isSet {
		fmt.Println("DB_PASSWORD is empty")
	}

	dbname, isSet := os.LookupEnv("DB_NAME")
	if !isSet {
		fmt.Println("DB_NAME is empty")
	}

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
