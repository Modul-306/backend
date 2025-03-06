package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Modul-306/backend/db"
)

type BlogRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Path    string `json:"path"`
}

type BlogResponse struct {
	ID      int    `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
	UserID  int    `json:"user_id"`
	Path    string `json:"path"`
}

func GetBlogs(h BaseHandler) {
	conn, err := db.CreateDBConnection()
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close(h.r.Context())

	blogs, err := db.New(conn).GetBlogs(h.r.Context())
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(h.w).Encode(blogs)
}

func GetBlog(h BaseHandler) {
	conn, err := db.CreateDBConnection()
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close(h.r.Context())

	id, err := strconv.Atoi(h.id)
	if err != nil {
		http.Error(h.w, "Invalid blog ID", http.StatusBadRequest)
		return
	}

	blog, err := db.New(conn).GetBlog(h.r.Context(), int32(id))
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(h.w).Encode(blog)
}

func CreateBlog(h BaseHandler) {
	conn, err := db.CreateDBConnection()
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close(h.r.Context())

	var req BlogRequest
	if err := json.NewDecoder(h.r.Body).Decode(&req); err != nil {
		http.Error(h.w, err.Error(), http.StatusBadRequest)
		return
	}

	dbConn := db.New(conn)
	users, err := dbConn.GetUsers(h.r.Context())
	fmt.Println(users)

	fmt.Println(h.username)
	user, err := dbConn.GetUserByUsername(h.r.Context(), h.username)
	fmt.Println(user)
	fmt.Println(err)
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}

	blog, err := dbConn.CreateBlog(h.r.Context(), db.CreateBlogParams{
		Title:   req.Title,
		Content: req.Content,
		UserID:  user.ID,
		Path:    req.Path,
	})
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.w.WriteHeader(http.StatusCreated)
	json.NewEncoder(h.w).Encode(blog)
}

func UpdateBlog(h BaseHandler) {
	conn, err := db.CreateDBConnection()
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close(h.r.Context())

	var req BlogRequest
	if err := json.NewDecoder(h.r.Body).Decode(&req); err != nil {
		http.Error(h.w, err.Error(), http.StatusBadRequest)
		return
	}

	dbConn := db.New(conn)

	user, err := dbConn.GetUserByUsername(h.r.Context(), h.username)
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}

	id, err := strconv.Atoi(h.id)
	if err != nil {
		http.Error(h.w, "Invalid blog ID", http.StatusBadRequest)
		return
	}

	blog, err := db.New(conn).UpdateBlog(h.r.Context(), db.UpdateBlogParams{
		ID:      int32(id),
		Title:   req.Title,
		Content: req.Content,
		UserID:  user.ID,
		Path:    req.Path,
	})
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(h.w).Encode(blog)
}

func DeleteBlog(h BaseHandler) {
	conn, err := db.CreateDBConnection()
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close(h.r.Context())

	id, err := strconv.Atoi(h.id)
	if err != nil {
		http.Error(h.w, "Invalid blog ID", http.StatusBadRequest)
		return
	}

	blog, err := db.New(conn).DeleteBlog(h.r.Context(), int32(id))
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.w.WriteHeader(http.StatusNoContent)
	json.NewEncoder(h.w).Encode(blog)
}
