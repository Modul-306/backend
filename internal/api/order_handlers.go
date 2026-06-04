package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/M306/backend/internal/db/sqlc"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type orderItemRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int32  `json:"quantity"`
}

type placeOrderRequest struct {
	Items         []orderItemRequest `json:"items"`
	PaymentMethod string             `json:"payment_method"`
	Street        string             `json:"street"`
	ZipCode       string             `json:"zip_code"`
	City          string             `json:"city"`
	FullName      string             `json:"full_name"`
}

type placeOrderResponse struct {
	Order       db.Order `json:"order"`
	RedirectURL string   `json:"redirect_url,omitempty"`
}

func (s *Server) handlePlaceOrder(w http.ResponseWriter, r *http.Request) {
	tenant := r.Context().Value(TenantContextKey).(db.Tenant)
	claims := r.Context().Value(UserContextKey).(UserClaims)
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		s.errorResponse(w, r, http.StatusUnauthorized, "Invalid user session")
		return
	}

	// Verify user profile exists
	userProfile, err := s.db.GetUserByID(r.Context(), userID)
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to retrieve user profile")
		return
	}

	var req placeOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "Invalid input")
		return
	}

	shippingStreet := req.Street
	if shippingStreet == "" && userProfile.Street.Valid {
		shippingStreet = userProfile.Street.String
	}
	shippingZipCode := req.ZipCode
	if shippingZipCode == "" && userProfile.ZipCode.Valid {
		shippingZipCode = userProfile.ZipCode.String
	}
	shippingCity := req.City
	if shippingCity == "" && userProfile.City.Valid {
		shippingCity = userProfile.City.String
	}
	shippingFullName := req.FullName
	if shippingFullName == "" && userProfile.FullName.Valid {
		shippingFullName = userProfile.FullName.String
	}

	if shippingStreet == "" || shippingZipCode == "" || shippingCity == "" {
		s.errorResponse(w, r, http.StatusBadRequest, "Please provide a delivery address or set it in your profile before placing an order")
		return
	}

	if len(req.Items) == 0 {
		s.errorResponse(w, r, http.StatusBadRequest, "Order must contain at least one item")
		return
	}

	paymentMethod := req.PaymentMethod
	if paymentMethod == "" {
		paymentMethod = "cash"
	}

	if paymentMethod != "cash" && paymentMethod != "online" {
		s.errorResponse(w, r, http.StatusBadRequest, "Invalid payment method. Must be 'cash' or 'online'")
		return
	}

	if paymentMethod == "cash" && !tenant.AllowsCashPayment {
		s.errorResponse(w, r, http.StatusBadRequest, "Cash payment is not enabled for this store")
		return
	}

	if paymentMethod == "online" && !tenant.AllowsOnlinePayment {
		s.errorResponse(w, r, http.StatusBadRequest, "Online payment is not enabled for this store")
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

	var initialStatus string
	if paymentMethod == "online" {
		initialStatus = "pending_payment"
	} else {
		initialStatus = "pending"
	}

	order, err := qtx.CreateOrder(r.Context(), db.CreateOrderParams{
		TenantID:         tenant.ID,
		UserID:           userID,
		Status:           initialStatus,
		TotalAmount:      fmt.Sprintf("%.2f", total),
		PaymentMethod:    paymentMethod,
		PayrexxGatewayID: sql.NullInt32{Valid: false},
		PaymentStatus:    "unpaid",
		ShippingStreet:   sql.NullString{String: shippingStreet, Valid: shippingStreet != ""},
		ShippingZipCode:  sql.NullString{String: shippingZipCode, Valid: shippingZipCode != ""},
		ShippingCity:     sql.NullString{String: shippingCity, Valid: shippingCity != ""},
		ShippingFullName: sql.NullString{String: shippingFullName, Valid: shippingFullName != ""},
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
			s.errorResponse(w, r, http.StatusBadRequest, "Failed to update stock (insufficient funds or concurrent order)")
			return
		}
		_ = updatedProduct
	}

	if err := tx.Commit(); err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to commit transaction")
		return
	}

	// Recalculate and update user loyalty tier based on number of orders placed in the last 365 days
	userOrders, err := s.db.ListOrdersByUser(r.Context(), userID)
	if err == nil {
		orderCount := 0
		oneYearAgo := time.Now().AddDate(-1, 0, 0)
		for _, o := range userOrders {
			if o.CreatedAt.Valid && o.CreatedAt.Time.After(oneYearAgo) {
				orderCount++
			}
		}
		var newTier string
		if orderCount >= 20 {
			newTier = "Harvest Elite"
		} else if orderCount >= 10 {
			newTier = "Harvester"
		} else if orderCount >= 3 {
			newTier = "Sprout"
		} else {
			newTier = "Seedling"
		}

		if userProfile.LoyaltyTier.String != newTier {
			_, _ = s.db.UpdateUserLoyaltyTier(r.Context(), db.UpdateUserLoyaltyTierParams{
				ID:          userID,
				LoyaltyTier: sql.NullString{String: newTier, Valid: true},
			})
		}
	}

	// For online payments, create the gateway link via Payrexx
	var redirectURL string
	if paymentMethod == "online" {
		gatewayID, payLink, err := s.createPayrexxGateway(order.ID, total)
		if err != nil {
			s.errorResponse(w, r, http.StatusInternalServerError, "Failed to create Payrexx payment gateway: "+err.Error())
			return
		}
		redirectURL = payLink

		// Update order gateway ID in database
		_, err = s.db.UpdateOrderGatewayID(r.Context(), db.UpdateOrderGatewayIDParams{
			ID:               order.ID,
			PayrexxGatewayID: sql.NullInt32{Int32: int32(gatewayID), Valid: true},
		})
		if err != nil {
			s.errorResponse(w, r, http.StatusInternalServerError, "Failed to save payment gateway ID on order")
			return
		}
	}

	// Post-commit: Send notifications for cash orders (online orders get emails on webhook confirmation)
	if paymentMethod == "cash" {
		go func() {
			_ = s.notifications.SendEmail(claims.Email, "Order Confirmation", fmt.Sprintf("Your order #%s for %.2f has been received.", order.ID.String()[:8], total))
			
			for _, item := range items {
				p, _ := s.db.GetProduct(context.Background(), item.productID)
				if p.Stock < 5 {
					owners, _ := s.db.ListTenantOwners(context.Background(), tenant.ID)
					for _, o := range owners {
						_ = s.notifications.SendEmail(o.Email, "LOW STOCK ALERT", fmt.Sprintf("Product %s is low on stock: %d remaining.", p.Name, p.Stock))
					}
				}
			}
		}()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(placeOrderResponse{
		Order:       order,
		RedirectURL: redirectURL,
	})
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

	order, err := s.db.GetOrder(r.Context(), id)
	if err != nil {
		s.errorResponse(w, r, http.StatusNotFound, "Order not found")
		return
	}

	// Double check status with Payrexx in real-time if pending payment
	if order.PaymentMethod == "online" && order.PaymentStatus != "paid" && order.Status == "pending_payment" && order.PayrexxGatewayID.Valid {
		payrexxStatus, err := s.checkPayrexxGatewayStatus(int(order.PayrexxGatewayID.Int32))
		if err != nil {
			log.Printf("Failed to check Payrexx gateway status for order %s: %v", order.ID, err)
		} else {
			var newStatus string
			var newPaymentStatus string
			switch payrexxStatus {
			case "confirmed":
				newStatus = "completed"
				newPaymentStatus = "paid"
			case "cancelled", "declined", "error":
				newStatus = "cancelled"
				newPaymentStatus = "unpaid"
			}

			if newStatus != "" {
				// Update order status and payment status in database
				updatedOrder, dbErr := s.db.UpdateOrderPaymentStatus(r.Context(), db.UpdateOrderPaymentStatusParams{
					ID:            order.ID,
					Status:        newStatus,
					PaymentStatus: newPaymentStatus,
				})
				if dbErr == nil {
					order = updatedOrder

					// Send email asynchronously
					go func() {
						u, err := s.db.GetUserByID(context.Background(), order.UserID)
						if err == nil {
							if newStatus == "completed" {
								_ = s.notifications.SendEmail(u.Email, "Order Payment Confirmed", "Your order payment has been successfully processed! Your order status is now completed.")
							} else if newStatus == "cancelled" {
								_ = s.notifications.SendEmail(u.Email, "Order Payment Failed", "Your order payment has failed or was cancelled. Your order status is now cancelled.")
							}
						}
					}()
				} else {
					log.Printf("Failed to update order status in DB: %v", dbErr)
				}
			}
		}
	}

	items, err := s.db.GetOrderItems(r.Context(), id)
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to get order details")
		return
	}

	type orderDetailsResponse struct {
		Order db.Order              `json:"order"`
		Items []db.GetOrderItemsRow `json:"items"`
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orderDetailsResponse{
		Order: order,
		Items: items,
	})
}

func (s *Server) handlePayOrder(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value(UserContextKey).(UserClaims)
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		s.errorResponse(w, r, http.StatusUnauthorized, "Invalid user session")
		return
	}

	idStr := chi.URLParam(r, "id")
	orderID, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "Invalid order ID")
		return
	}

	order, err := s.db.GetOrder(r.Context(), orderID)
	if err != nil {
		s.errorResponse(w, r, http.StatusNotFound, "Order not found")
		return
	}

	if order.UserID != userID {
		s.errorResponse(w, r, http.StatusForbidden, "You do not own this order")
		return
	}

	if order.Status != "pending_payment" || order.PaymentStatus == "paid" {
		s.errorResponse(w, r, http.StatusBadRequest, "Order is already paid or cannot be paid")
		return
	}

	var total float64
	fmt.Sscanf(order.TotalAmount, "%f", &total)

	gatewayID, payLink, err := s.createPayrexxGateway(order.ID, total)
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to create Payrexx payment gateway: "+err.Error())
		return
	}

	// Update order gateway ID in database
	_, err = s.db.UpdateOrderGatewayID(r.Context(), db.UpdateOrderGatewayIDParams{
		ID:               order.ID,
		PayrexxGatewayID: sql.NullInt32{Int32: int32(gatewayID), Valid: true},
	})
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to save payment gateway ID on order")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"redirect_url": payLink})
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
