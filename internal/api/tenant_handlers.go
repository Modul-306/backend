package api

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/M306/backend/internal/db/sqlc"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (s *Server) handleListTenants(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	category := r.URL.Query().Get("category")

	tenants, err := s.db.ListTenants(r.Context(), db.ListTenantsParams{
		Column1:  sql.NullString{String: search, Valid: true},
		Category: sql.NullString{String: category, Valid: true},
	})
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to list tenants")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tenants)
}

func (s *Server) handleGetTenant(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	tenant, err := s.db.GetTenantBySlug(r.Context(), slug)
	if err != nil {
		s.errorResponse(w, r, http.StatusNotFound, "Tenant not found")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tenant)
}

func (s *Server) handleCreateTenant(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string `json:"name"`
		Slug    string `json:"slug"`
		OwnerID string `json:"owner_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "Invalid input")
		return
	}

	tenant, err := s.db.CreateTenant(r.Context(), db.CreateTenantParams{
		Name: req.Name,
		Slug: req.Slug,
	})
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to create tenant")
		return
	}

	if req.OwnerID != "" {
		ownerUUID, err := uuid.Parse(req.OwnerID)
		if err != nil {
			s.errorResponse(w, r, http.StatusBadRequest, "Invalid owner_id UUID")
			return
		}
		tenant, err = s.db.SetTenantOwner(r.Context(), db.SetTenantOwnerParams{
			ID:      tenant.ID,
			OwnerID: uuid.NullUUID{UUID: ownerUUID, Valid: true},
		})
		if err != nil {
			s.errorResponse(w, r, http.StatusInternalServerError, "Failed to set tenant owner")
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(tenant)
}

func (s *Server) handleUpdateTenant(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "Invalid tenant ID")
		return
	}

	var req struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "Invalid input")
		return
	}

	arg := db.UpdateTenantParams{
		ID:   id,
		Name: req.Name,
		Slug: req.Slug,
	}

	updatedTenant, err := s.db.UpdateTenant(r.Context(), arg)
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to update tenant")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedTenant)
}

func (s *Server) handleDeleteTenant(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "Invalid tenant ID")
		return
	}

	err = s.db.DeleteTenant(r.Context(), id)
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to delete tenant")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleUpdateTenantIcon(w http.ResponseWriter, r *http.Request) {
	tenant, ok := r.Context().Value(TenantContextKey).(db.Tenant)
	if !ok {
		s.errorResponse(w, r, http.StatusNotFound, "Tenant not found")
		return
	}

	var req struct {
		IconUrl string `json:"icon_url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "Invalid input")
		return
	}

	arg := db.UpdateTenantIconParams{
		ID:      tenant.ID,
		IconUrl: sql.NullString{String: req.IconUrl, Valid: req.IconUrl != ""},
	}

	updatedTenant, err := s.db.UpdateTenantIcon(r.Context(), arg)
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to update icon")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedTenant)
}

func (s *Server) handleUpdateTenantAppearance(w http.ResponseWriter, r *http.Request) {
	tenant, ok := r.Context().Value(TenantContextKey).(db.Tenant)
	if !ok {
		s.errorResponse(w, r, http.StatusNotFound, "Tenant not found")
		return
	}

	var req struct {
		CoverUrl    string `json:"cover_url"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "Invalid input")
		return
	}

	arg := db.UpdateTenantAppearanceParams{
		ID:          tenant.ID,
		CoverUrl:    sql.NullString{String: req.CoverUrl, Valid: req.CoverUrl != ""},
		Description: sql.NullString{String: req.Description, Valid: req.Description != ""},
	}

	updatedTenant, err := s.db.UpdateTenantAppearance(r.Context(), arg)
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to update appearance")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedTenant)
}

func (s *Server) handleSetTenantOwner(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "Invalid tenant ID")
		return
	}

	var req struct {
		OwnerID string `json:"owner_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "Invalid input")
		return
	}

	var ownerID uuid.NullUUID
	if req.OwnerID != "" {
		parsed, err := uuid.Parse(req.OwnerID)
		if err != nil {
			s.errorResponse(w, r, http.StatusBadRequest, "Invalid owner ID")
			return
		}
		ownerID = uuid.NullUUID{UUID: parsed, Valid: true}
	}

	tenant, err := s.db.SetTenantOwner(r.Context(), db.SetTenantOwnerParams{
		ID:      id,
		OwnerID: ownerID,
	})
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to set owner")
		return
	}

	// Also add to tenant_owners table for compatibility and promote role
	if ownerID.Valid {
		_ = s.db.AddTenantOwner(r.Context(), db.AddTenantOwnerParams{
			TenantID: id,
			UserID:   ownerID.UUID,
		})
		
		// Promote user to farmer_admin if they aren't platform_admin
		u, err := s.db.GetUserByID(r.Context(), ownerID.UUID)
		if err == nil && u.Role == "customer" {
			_, _ = s.db.UpdateUserRole(r.Context(), db.UpdateUserRoleParams{
				ID:   ownerID.UUID,
				Role: "farmer_admin",
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tenant)
}

func (s *Server) handleAddTenantOwner(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	tenantID, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "Invalid tenant ID")
		return
	}

	var req struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "Invalid input")
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "Invalid user ID")
		return
	}

	err = s.db.AddTenantOwner(r.Context(), db.AddTenantOwnerParams{
		TenantID: tenantID,
		UserID:   userID,
	})
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to add owner")
		return
	}
	
	// Promote user to farmer_admin if they aren't platform_admin
	u, err := s.db.GetUserByID(r.Context(), userID)
	if err == nil && u.Role == "customer" {
		_, _ = s.db.UpdateUserRole(r.Context(), db.UpdateUserRoleParams{
			ID:   userID,
			Role: "farmer_admin",
		})
	}

	w.WriteHeader(http.StatusCreated)
}

func (s *Server) handleRemoveTenantOwner(w http.ResponseWriter, r *http.Request) {
	tenantID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "Invalid tenant ID")
		return
	}

	userID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "Invalid user ID")
		return
	}

	err = s.db.RemoveTenantOwner(r.Context(), db.RemoveTenantOwnerParams{
		TenantID: tenantID,
		UserID:   userID,
	})
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to remove owner")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListTenantOwners(w http.ResponseWriter, r *http.Request) {
	tenantID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "Invalid tenant ID")
		return
	}

	owners, err := s.db.ListTenantOwners(r.Context(), tenantID)
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to list owners")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(owners)
}

func (s *Server) handleListTenantCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := s.db.ListTenantCategories(r.Context())
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to list tenant categories")
		return
	}
	
	res := make([]string, 0)
	for _, c := range categories {
		if c.Valid {
			res = append(res, c.String)
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}
