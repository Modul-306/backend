package api

import (
	"encoding/json"
	"net/http"

	"github.com/M306/backend/internal/db/sqlc"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (s *Server) handleListBlogs(w http.ResponseWriter, r *http.Request) {
	tenant := r.Context().Value(TenantContextKey).(db.Tenant)
	blogs, err := s.db.ListBlogs(r.Context(), tenant.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(blogs)
}

func (s *Server) handleGetBlog(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "Invalid blog ID")
		return
	}

	blog, err := s.db.GetBlog(r.Context(), id)
	if err != nil {
		s.errorResponse(w, r, http.StatusNotFound, "Blog not found")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(blog)
}

func (s *Server) handleCreateBlog(w http.ResponseWriter, r *http.Request) {
	tenant := r.Context().Value(TenantContextKey).(db.Tenant)
	var arg db.CreateBlogParams
	if err := json.NewDecoder(r.Body).Decode(&arg); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	arg.TenantID = tenant.ID
	blog, err := s.db.CreateBlog(r.Context(), arg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(blog)
}

func (s *Server) handleUpdateBlog(w http.ResponseWriter, r *http.Request) {
	tenant := r.Context().Value(TenantContextKey).(db.Tenant)
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid blog ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Title     string `json:"title"`
		ContentMd string `json:"content_md"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	blog, err := s.db.UpdateBlog(r.Context(), db.UpdateBlogParams{
		ID:        id,
		TenantID:  tenant.ID,
		Title:     req.Title,
		ContentMd: req.ContentMd,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(blog)
}

func (s *Server) handleDeleteBlog(w http.ResponseWriter, r *http.Request) {
	tenant := r.Context().Value(TenantContextKey).(db.Tenant)
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid blog ID", http.StatusBadRequest)
		return
	}

	err = s.db.DeleteBlog(r.Context(), db.DeleteBlogParams{
		ID:       id,
		TenantID: tenant.ID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
