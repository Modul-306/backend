package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/M306/backend/internal/api"
	"github.com/M306/backend/internal/db/sqlc"
	"github.com/M306/backend/internal/storage"
	_ "github.com/lib/pq"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	var conn *sql.DB
	var err error

	// Wait for DB to be ready
	for i := 0; i < 10; i++ {
		conn, err = sql.Open("postgres", dbURL)
		if err == nil {
			err = conn.Ping()
			if err == nil {
				break
			}
		}
		log.Printf("Waiting for database... %v", err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatal("Could not connect to database:", err)
	}

	// Run migrations
	entries, err := os.ReadDir("migrations")
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".up.sql") {
				migrationSql, err := os.ReadFile("migrations/" + entry.Name())
				if err != nil {
					log.Printf("Warning: could not read migration file %s: %v", entry.Name(), err)
					continue
				}

				// Ensure pgcrypto for UUIDs if needed
				_, _ = conn.Exec("CREATE EXTENSION IF NOT EXISTS pgcrypto;")
				
				queries := strings.Split(string(migrationSql), ";")
				for _, q := range queries {
					q = strings.TrimSpace(q)
					if q == "" {
						continue
					}
					_, err = conn.Exec(q)
					if err != nil {
						// Don't fail if table/column already exists
						if !strings.Contains(err.Error(), "already exists") && !strings.Contains(err.Error(), "duplicate column name") {
							log.Printf("Migration part failed in %s: %v", entry.Name(), err)
						}
					}
				}
				log.Printf("Migration %s processed", entry.Name())
			}
		}
	} else {
		log.Printf("Warning: could not read migrations directory: %v", err)
	}

	dbQueries := db.New(conn)

	endpoint := os.Getenv("MINIO_ENDPOINT")
	accessKey := os.Getenv("MINIO_ACCESS_KEY")
	secretKey := os.Getenv("MINIO_SECRET_KEY")
	
	// Wait for MinIO
	var s3Storage *storage.S3Storage
	for i := 0; i < 10; i++ {
		s3Storage, err = storage.NewS3Storage(context.Background(), endpoint, accessKey, secretKey)
		if err == nil {
			break
		}
		log.Printf("Waiting for MinIO... %v", err)
		time.Sleep(2 * time.Second)
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	server := api.NewServer(dbQueries, conn, s3Storage, jwtSecret)

	// Seed default tenant and admin user
	ctx := context.Background()
	_, err = dbQueries.GetTenantBySlug(ctx, "default")
	if err != nil {
		log.Println("Seeding default tenant...")
		_, _ = dbQueries.CreateTenant(ctx, db.CreateTenantParams{
			Name: "Default Farm",
			Slug: "default",
		})
	}

	admin, err := dbQueries.GetUserByEmail(ctx, "admin@cattlehof.ch")
	if err != nil {
		log.Println("Seeding admin user...")
		// password: admin
		hash, _ := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
		_, _ = dbQueries.CreateUser(ctx, db.CreateUserParams{
			TenantID:     uuid.NullUUID{},
			Email:        "admin@cattlehof.ch",
			PasswordHash: string(hash),
			Role:         "platform_admin",
		})
	} else if admin.Role != "platform_admin" {
		log.Println("Fixing admin user role...")
		_, _ = dbQueries.UpdateUserRole(ctx, db.UpdateUserRoleParams{
			ID:   admin.ID,
			Role: "platform_admin",
		})
	}

	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", server.Routes()); err != nil {
		log.Fatal(err)
	}
}
