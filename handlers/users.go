package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Modul-306/backend/db"
	"github.com/jackc/pgx/v5/pgtype"
)

type UserRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
	Email    string `json:"email"`
	IsAdmin  bool   `json:"is_admin"`
}

type UserResponse struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	IsAdmin bool   `json:"is_admin"`
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

	userDB := db.New(conn)

	reqUser, err := userDB.GetUserByUsername(h.r.Context(), h.username)
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}

	user, err := userDB.DeleteUser(h.r.Context(), db.DeleteUserParams{ID: int32(id), Column2: reqUser.ID})
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.w.WriteHeader(http.StatusNoContent)
	json.NewEncoder(h.w).Encode(user)
}

func UpdateUser(h BaseHandler) {
	conn, err := db.CreateDBConnection()
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close(h.r.Context())

	var req UserRequest
	if err := json.NewDecoder(h.r.Body).Decode(&req); err != nil {
		http.Error(h.w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(h.id)
	if err != nil {
		http.Error(h.w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	IsAdmin := pgtype.Bool{}
	err = IsAdmin.Scan(req.IsAdmin)
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := db.New(conn).UpdateUser(h.r.Context(), db.UpdateUserParams{
		ID:       int32(id),
		Name:     req.Name,
		Password: req.Password,
		Email:    req.Email,
		IsAdmin:  IsAdmin,
	})
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(h.w).Encode(user)
}
