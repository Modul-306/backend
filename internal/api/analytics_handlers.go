package api

import (
	"encoding/json"
	"net/http"

	"github.com/M306/backend/internal/db/sqlc"
)

func (s *Server) handleGetRevenueAnalytics(w http.ResponseWriter, r *http.Request) {
	tenant := r.Context().Value(TenantContextKey).(db.Tenant)
	revenue, err := s.db.GetRevenueByDay(r.Context(), tenant.ID)
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to fetch analytics")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(revenue)
}

func (s *Server) handleGetTopProducts(w http.ResponseWriter, r *http.Request) {
	tenant := r.Context().Value(TenantContextKey).(db.Tenant)
	products, err := s.db.GetTopSellingProducts(r.Context(), tenant.ID)
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to fetch top products")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(products)
}
