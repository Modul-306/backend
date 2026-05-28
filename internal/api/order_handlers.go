package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/M306/backend/internal/db/sqlc"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type orderItemRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int32  `json:"quantity"`
}

type placeOrderRequest struct {
	Items []orderItemRequest `json:"items"`
}

func (s *Server) handlePlaceOrder(w http.ResponseWriter, r *http.Request) {
	tenant := r.Context().Value(TenantContextKey).(db.Tenant)
	claims := r.Context().Value(UserContextKey).(UserClaims)
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		s.errorResponse(w, r, http.StatusUnauthorized, "Invalid user session")
		return
	}

	var req placeOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "Invalid input")
		return
	}

	if len(req.Items) == 0 {
		s.errorResponse(w, r, http.StatusBadRequest, "Order must contain at least one item")
		return
	}

	// 1. Fetch products and calculate total outside transaction to minimize lock time
	var total float64
	type itemWithPrice struct {
		productID uuid.UUID
		quantity  int32
		price     string
	}
	var items []itemWithPrice

	products, err := s.db.ListProducts(r.Context(), db.ListProductsParams{
		TenantID: tenant.ID,
		Column2:  sql.NullString{String: "", Valid: true},
		Category: sql.NullString{String: "", Valid: true},
	})
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to retrieve products")
		return
	}

	for _, item := range req.Items {
		pID, err := uuid.Parse(item.ProductID)
		if err != nil {
			s.errorResponse(w, r, http.StatusBadRequest, "Invalid product ID: "+item.ProductID)
			return
		}

		var foundProduct *db.Product
		for _, p := range products {
			if p.ID == pID {
				foundProduct = &p
				break
			}
		}

		if foundProduct == nil {
			s.errorResponse(w, r, http.StatusBadRequest, "Product not found: "+item.ProductID)
			return
		}

		if foundProduct.Stock < item.Quantity {
			s.errorResponse(w, r, http.StatusBadRequest, "Insufficient stock for product: "+foundProduct.Name)
			return
		}

		var price float64
		fmt.Sscanf(foundProduct.Price, "%f", &price)
		total += price * float64(item.Quantity)
		items = append(items, itemWithPrice{
			productID: pID,
			quantity:  item.Quantity,
			price:     foundProduct.Price,
		})
	}

	// Apply loyalty discount
	discountPercentStr, _ := s.db.GetUserDiscount(r.Context(), userID)
	var discountPercent float64
	fmt.Sscanf(discountPercentStr, "%f", &discountPercent)
	
	if discountPercent > 0 {
		total = total * (1 - (discountPercent / 100))
	}

	// 2. Execute order placement in a transaction
	tx, err := s.conn.BeginTx(r.Context(), nil)
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to start transaction")
		return
	}
	defer tx.Rollback()

	qtx := s.db.WithTx(tx)

	order, err := qtx.CreateOrder(r.Context(), db.CreateOrderParams{
		TenantID:    tenant.ID,
		UserID:      userID,
		Status:      "pending",
		TotalAmount: fmt.Sprintf("%.2f", total),
	})
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to create order")
		return
	}

	for _, item := range items {
		_, err = qtx.CreateOrderItem(r.Context(), db.CreateOrderItemParams{
			OrderID:      order.ID,
			ProductID:    item.productID,
			Quantity:     item.quantity,
			PriceAtTime:  item.price,
		})
		if err != nil {
			s.errorResponse(w, r, http.StatusInternalServerError, "Failed to create order item")
			return
		}

		updatedProduct, err := qtx.UpdateProductStock(r.Context(), db.UpdateProductStockParams{
			ID:       item.productID,
			TenantID: tenant.ID,
			Stock:    item.quantity,
		})
		if err != nil {
			// This will fail if stock < quantity due to CHECK constraint or WHERE clause
			s.errorResponse(w, r, http.StatusBadRequest, "Failed to update stock (insufficient funds or concurrent order)")
			return
		}
		_ = updatedProduct
	}

	if err := tx.Commit(); err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to commit transaction")
		return
	}

	// Post-commit: Send notifications (non-blocking or at least outside tx)
	go func() {
		_ = s.notifications.SendEmail(claims.Email, "Order Confirmation", fmt.Sprintf("Your order #%s for %.2f has been received.", order.ID.String()[:8], total))
		
		// Check for low stock and notify tenant owners?
		// For simplicity, just log it for now
		for _, item := range items {
			p, _ := s.db.GetProduct(context.Background(), item.productID)
			if p.Stock < 5 {
				// Fetch owners and notify them
				owners, _ := s.db.ListTenantOwners(context.Background(), tenant.ID)
				for _, o := range owners {
					_ = s.notifications.SendEmail(o.Email, "LOW STOCK ALERT", fmt.Sprintf("Product %s is low on stock: %d remaining.", p.Name, p.Stock))
				}
			}
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(order)
}

func (s *Server) handleListOrders(w http.ResponseWriter, r *http.Request) {
	tenant, ok := r.Context().Value(TenantContextKey).(db.Tenant)
	if !ok {
		s.errorResponse(w, r, http.StatusNotFound, "Tenant not found")
		return
	}
	orders, err := s.db.ListOrdersByTenant(r.Context(), tenant.ID)
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to list orders")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

func (s *Server) handleListMyOrders(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(UserContextKey).(UserClaims)
	if !ok {
		s.errorResponse(w, r, http.StatusUnauthorized, "Unauthorized")
		return
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		s.errorResponse(w, r, http.StatusUnauthorized, "Invalid user session")
		return
	}
	orders, err := s.db.ListOrdersByUser(r.Context(), userID)
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to list your orders")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

func (s *Server) handleGetOrderDetails(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "Invalid order ID")
		return
	}

	items, err := s.db.GetOrderItems(r.Context(), id)
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to get order details")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func (s *Server) handleUpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	tenant, ok := r.Context().Value(TenantContextKey).(db.Tenant)
	if !ok {
		s.errorResponse(w, r, http.StatusNotFound, "Tenant not found")
		return
	}
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "Invalid order ID")
		return
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "Invalid input")
		return
	}

	order, err := s.db.UpdateOrderStatus(r.Context(), db.UpdateOrderStatusParams{
		ID:       id,
		TenantID: tenant.ID,
		Status:   req.Status,
	})
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to update order status")
		return
	}

	// Notify user
	go func() {
		u, _ := s.db.GetUserByID(context.Background(), order.UserID)
		_ = s.notifications.SendEmail(u.Email, "Order Status Update", fmt.Sprintf("Your order #%s status has been updated to: %s", order.ID.String()[:8], order.Status))
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}
