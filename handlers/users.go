package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Modul-306/backend/db"
)

type UserRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

type UserResponse struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func GetUsers(h BaseHandler) {
	conn, err := db.CreateDBConnection()
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close(h.r.Context())

	users, err := db.New(conn).GetUsers(h.r.Context())
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(h.w).Encode(users)
}

func GetUser(h BaseHandler) {
	conn, err := db.CreateDBConnection()
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close(h.r.Context())

	id, err := strconv.Atoi(h.id)
	if err != nil {
		http.Error(h.w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	user, err := db.New(conn).GetUser(h.r.Context(), int32(id))
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(h.w).Encode(user)
}

func DeleteUser(h BaseHandler) {
	conn, err := db.CreateDBConnection()
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close(h.r.Context())

	id, err := strconv.Atoi(h.id)
	if err != nil {
		http.Error(h.w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	user, err := db.New(conn).DeleteUser(h.r.Context(), int32(id))
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.w.WriteHeader(http.StatusNoContent)
	json.NewEncoder(h.w).Encode(user)
}
