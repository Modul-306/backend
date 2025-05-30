package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Modul-306/backend/db"
)

func Login(w http.ResponseWriter, r *http.Request) {
	var creds Credentials
	err := json.NewDecoder(r.Body).Decode(&creds)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Authenticate the user
	conn, err := db.CreateDBConnection()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer conn.Close(context.Background())

	db := db.New(conn)

	user, err := db.GetUserByUsername(context.Background(), creds.Username)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if !VerifyPassword(creds.Password, user.Password) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	expirationTime := time.Now().Add(5 * time.Minute)

	tokenString, err := CreateToken(creds.Username, expirationTime)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:    "token",
		Value:   tokenString,
		Expires: expirationTime,
	})
}

func SignUp(w http.ResponseWriter, r *http.Request) {
	var creds SignUpCredentials
	err := json.NewDecoder(r.Body).Decode(&creds)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Authenticate the user
	conn, err := db.CreateDBConnection()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer conn.Close(context.Background())

	hashedPassword, err := HashPassword(creds.Password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	dbConn := db.New(conn)
	_, err = dbConn.CreateUser(context.Background(), db.CreateUserParams{
		Name:     creds.Username,
		Password: hashedPassword,
		Email:    creds.Email,
	})

	if err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	expirationTime := time.Now().Add(5 * time.Minute)

	tokenString, err := CreateToken(creds.Username, expirationTime)
	if err != nil {
		http.Error(w, "Failed to create token", http.StatusInternalServerError)
		return
	}

	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:    "token",
		Value:   tokenString,
		Expires: expirationTime,
	})

	w.WriteHeader(http.StatusCreated)
}
