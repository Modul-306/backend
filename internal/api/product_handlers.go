package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/M306/backend/internal/db/sqlc"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (s *Server) handleListProducts(w http.ResponseWriter, r *http.Request) {
	tenant, ok := r.Context().Value(TenantContextKey).(db.Tenant)
	if !ok {
		s.errorResponse(w, r, http.StatusNotFound, "Tenant not found in context")
		return
	}
	search := r.URL.Query().Get("search")
	category := r.URL.Query().Get("category")

	products, err := s.db.ListProducts(r.Context(), db.ListProductsParams{
		TenantID: tenant.ID,
		Column2:  sql.NullString{String: search, Valid: true},
		Category: sql.NullString{String: category, Valid: true},
	})
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to list products")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(products)
}

func (s *Server) handleGetProduct(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "Invalid product ID")
		return
	}

	product, err := s.db.GetProduct(r.Context(), id)
	if err != nil {
		s.errorResponse(w, r, http.StatusNotFound, "Product not found")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(product)
}

func (s *Server) handleAddProduct(w http.ResponseWriter, r *http.Request) {
	tenant := r.Context().Value(TenantContextKey).(db.Tenant)
	
	var req struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Price       float64 `json:"price"`
		Stock       int32   `json:"stock"`
		ImageUrl    string  `json:"image_url"`
		Category    string  `json:"category"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input: "+err.Error(), http.StatusBadRequest)
		return
	}

	arg := db.CreateProductParams{
		TenantID:    tenant.ID,
		Name:        req.Name,
		Description: sql.NullString{String: req.Description, Valid: req.Description != ""},
		Price:       fmt.Sprintf("%.2f", req.Price),
		Stock:       req.Stock,
		ImageUrl:    sql.NullString{String: req.ImageUrl, Valid: req.ImageUrl != ""},
		Category:    sql.NullString{String: req.Category, Valid: req.Category != ""},
	}

	product, err := s.db.CreateProduct(r.Context(), arg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(product)
}

func (s *Server) handleUpdateProduct(w http.ResponseWriter, r *http.Request) {
	tenant := r.Context().Value(TenantContextKey).(db.Tenant)
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Price       float64 `json:"price"`
		Stock       int32   `json:"stock"`
		ImageUrl    string  `json:"image_url"`
		Category    string  `json:"category"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	arg := db.UpdateProductParams{
		ID:          id,
		TenantID:    tenant.ID,
		Name:        req.Name,
		Description: sql.NullString{String: req.Description, Valid: req.Description != ""},
		Price:       fmt.Sprintf("%.2f", req.Price),
		Stock:       req.Stock,
		ImageUrl:    sql.NullString{String: req.ImageUrl, Valid: req.ImageUrl != ""},
		Category:    sql.NullString{String: req.Category, Valid: req.Category != ""},
	}

	product, err := s.db.UpdateProduct(r.Context(), arg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(product)
}

func (s *Server) handleDeleteProduct(w http.ResponseWriter, r *http.Request) {
	tenant := r.Context().Value(TenantContextKey).(db.Tenant)
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	err = s.db.DeleteProduct(r.Context(), db.DeleteProductParams{
		ID:       id,
		TenantID: tenant.ID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListCategories(w http.ResponseWriter, r *http.Request) {
	tenant := r.Context().Value(TenantContextKey).(db.Tenant)
	categories, err := s.db.ListCategories(r.Context(), tenant.ID)
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to list categories")
		return
	}
	
	// Convert sql.NullString to string array
	res := make([]string, 0)
	for _, c := range categories {
		if c.Valid {
			res = append(res, c.String)
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}
