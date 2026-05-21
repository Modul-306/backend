package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/M306/backend/internal/db/sqlc"
	"github.com/M306/backend/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Server struct {
	db        *db.Queries
	conn      *sql.DB
	storage   storage.Storage
	jwtSecret string
}

func NewServer(queries *db.Queries, conn *sql.DB, storage storage.Storage, jwtSecret string) *Server {
	return &Server{
		db:        queries,
		conn:      conn,
		storage:   storage,
		jwtSecret: jwtSecret,
	}
}

func (s *Server) errorResponse(w http.ResponseWriter, r *http.Request, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Tenant-Slug"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Route("/api/v1", func(r chi.Router) {
		// --- Public Routes ---
		r.Post("/auth/login", s.handleLogin)
		r.Post("/auth/register", s.handleRegister)
		r.Get("/tenants", s.handleListTenants)
		
		r.Group(func(r chi.Router) {
			r.Use(s.TenantMiddleware)
			r.Get("/tenants/{slug}", s.handleGetTenant)
			r.Get("/products", s.handleListProducts)
			r.Get("/blogs", s.handleListBlogs)
		})

		// --- Authenticated Routes (Any Role) ---
		r.Group(func(r chi.Router) {
			r.Use(s.JWTMiddleware)
			r.Get("/auth/me", s.handleMe)
			r.Post("/upload", s.handleUpload)
			
			r.Group(func(r chi.Router) {
				r.Use(s.TenantMiddleware)
				r.Post("/orders", s.handlePlaceOrder)
				r.Get("/orders/my", s.handleListMyOrders)
				r.Get("/orders/{id}", s.handleGetOrderDetails)
			})
		})

		// --- Global Admin Routes ---
		r.Group(func(r chi.Router) {
			r.Use(s.JWTMiddleware)
			r.Use(s.RequireRole("platform_admin"))
			
			r.Post("/tenants", s.handleCreateTenant)
			r.Put("/tenants/{id}", s.handleUpdateTenant)
			r.Put("/tenants/{id}/owner", s.handleSetTenantOwner)
			r.Delete("/tenants/{id}", s.handleDeleteTenant)
			r.Get("/users", s.handleListUsers)
		})

		// --- Tenant Management Routes (Owner/Platform Admin only) ---
		r.Group(func(r chi.Router) {
			r.Use(s.JWTMiddleware)
			r.Use(s.TenantMiddleware)
			r.Use(s.RequireTenantAccess) // Ensures user is owner or platform_admin
			
			r.Put("/tenants/icon", s.handleUpdateTenantIcon)
			r.Put("/tenants/appearance", s.handleUpdateTenantAppearance)
			r.Get("/orders", s.handleListOrders) // List all orders for this tenant
			r.Put("/orders/{id}/status", s.handleUpdateOrderStatus)
			
			r.Group(func(r chi.Router) {
				r.Use(s.RequireRole("farmer_admin", "platform_admin"))
				r.Post("/products", s.handleAddProduct)
				r.Put("/products/{id}", s.handleUpdateProduct)
				r.Delete("/products/{id}", s.handleDeleteProduct)
				r.Post("/blogs", s.handleCreateBlog)
				r.Put("/blogs/{id}", s.handleUpdateBlog)
				r.Delete("/blogs/{id}", s.handleDeleteBlog)
			})
		})
	})

	return r
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "Invalid input")
		return
	}

	if req.Email == "" || req.Password == "" {
		s.errorResponse(w, r, http.StatusBadRequest, "Email and password are required")
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to process password")
		return
	}

	arg := db.CreateUserParams{
		TenantID:     uuid.NullUUID{},
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Role:         "customer",
	}

	user, err := s.db.CreateUser(r.Context(), arg)
	if err != nil {
		if strings.Contains(err.Error(), "unique constraint") {
			s.errorResponse(w, r, http.StatusConflict, "Email already in use")
			return
		}
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to create user")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "Invalid input")
		return
	}

	user, err := s.db.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		s.errorResponse(w, r, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		s.errorResponse(w, r, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID.String(),
		"email":   user.Email,
		"tenant_id": func() string {
			if user.TenantID.Valid {
				return user.TenantID.UUID.String()
			}
			return ""
		}(),
		"role": user.Role,
		"exp":  time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token":   tokenString,
		"role":    user.Role,
		"user_id": user.ID.String(),
	})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(UserContextKey).(UserClaims)
	if !ok {
		s.errorResponse(w, r, http.StatusUnauthorized, "Unauthorized")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(claims)
}

func (s *Server) handleListTenants(w http.ResponseWriter, r *http.Request) {
	tenants, err := s.db.ListTenants(r.Context())
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to list tenants")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tenants)
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

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.db.ListUsers(r.Context())
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to list users")
		return
	}
	type safeUser struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	result := make([]safeUser, 0)
	for _, u := range users {
		result = append(result, safeUser{
			ID:    u.ID.String(),
			Email: u.Email,
			Role:  u.Role,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20) // 10MB
	if err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "File too large")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		s.errorResponse(w, r, http.StatusBadRequest, "Invalid file")
		return
	}
	defer file.Close()

	url, err := s.storage.UploadFile(r.Context(), "cattlehof", header.Filename, file)
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to upload file")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"url": url})
}

func (s *Server) handleListProducts(w http.ResponseWriter, r *http.Request) {
	tenant, ok := r.Context().Value(TenantContextKey).(db.Tenant)
	if !ok {
		s.errorResponse(w, r, http.StatusNotFound, "Tenant not found in context")
		return
	}
	products, err := s.db.ListProducts(r.Context(), tenant.ID)
	if err != nil {
		s.errorResponse(w, r, http.StatusInternalServerError, "Failed to list products")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(products)
}

func (s *Server) handleListBlogs(w http.ResponseWriter, r *http.Request) {
	tenant := r.Context().Value(TenantContextKey).(db.Tenant)
	blogs, err := s.db.ListBlogs(r.Context(), tenant.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(blogs)
}

func (s *Server) handleAddProduct(w http.ResponseWriter, r *http.Request) {
	tenant := r.Context().Value(TenantContextKey).(db.Tenant)
	
	var req struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Price       float64 `json:"price"`
		Stock       int32   `json:"stock"`
		ImageUrl    string  `json:"image_url"`
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
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	product, err := s.db.UpdateProduct(r.Context(), db.UpdateProductParams{
		ID:          id,
		TenantID:    tenant.ID,
		Name:        req.Name,
		Description: sql.NullString{String: req.Description, Valid: req.Description != ""},
		Price:       fmt.Sprintf("%.2f", req.Price),
		Stock:       req.Stock,
		ImageUrl:    sql.NullString{String: req.ImageUrl, Valid: req.ImageUrl != ""},
	})
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

	products, err := s.db.ListProducts(r.Context(), tenant.ID)
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
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
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tenant)
}
