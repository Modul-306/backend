package auth

import (
	"encoding/json"
	"net/http"
	"time"
)

func Login(w http.ResponseWriter, r *http.Request) {
	var creds Credentials
	err := json.NewDecoder(r.Body).Decode(&creds)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Authenticate the user
	if creds.Username != "user" || creds.Password != "password" {
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
	// Implement sign-up logic
}
