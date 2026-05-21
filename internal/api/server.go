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
	db        db.Querier
	storage   storage.Storage
	jwtSecret string
}

func NewServer(db db.Querier, storage storage.Storage, jwtSecret string) *Server {
	return &Server{
		db:        db,
		storage:   storage,
		jwtSecret: jwtSecret,
	}
}

func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Tenant-Slug"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/login", s.handleLogin)
		r.Post("/auth/register", s.handleRegister)
		r.Get("/auth/me", s.handleMe)
		r.Get("/tenants", s.handleListTenants)
		r.Get("/tenants/{slug}", s.handleGetTenant)
		r.Post("/tenants", s.handleCreateTenant)
		r.Put("/tenants/{id}", s.handleUpdateTenant)
		r.Put("/tenants/{id}/owner", s.handleSetTenantOwner)
		r.Delete("/tenants/{id}", s.handleDeleteTenant)
		r.Get("/users", s.handleListUsers)
		r.Post("/upload", s.handleUpload)

		r.Group(func(r chi.Router) {
			r.Use(s.TenantMiddleware)

			r.Get("/products", s.handleListProducts)
			r.Get("/blogs", s.handleListBlogs)

			r.Group(func(r chi.Router) {
				r.Use(s.JWTMiddleware)

				r.Put("/tenants/icon", s.handleUpdateTenantIcon)
				r.Put("/tenants/appearance", s.handleUpdateTenantAppearance)
				r.Post("/products", s.handleAddProduct)
				r.Post("/blogs", s.handleCreateBlog)
				r.Post("/orders", s.handlePlaceOrder)
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
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	user, err := s.db.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":   user.ID.String(),
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"token":   tokenString,
		"role":    user.Role,
		"user_id": user.ID.String(),
	})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id":   claims["user_id"],
		"tenant_id": claims["tenant_id"],
		"role":      claims["role"],
	})
}

func (s *Server) handleListTenants(w http.ResponseWriter, r *http.Request) {
	tenants, err := s.db.ListTenants(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(tenants)
}

func (s *Server) handleCreateTenant(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string `json:"name"`
		Slug    string `json:"slug"`
		OwnerID string `json:"owner_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	tenant, err := s.db.CreateTenant(r.Context(), db.CreateTenantParams{
		Name: req.Name,
		Slug: req.Slug,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// If an existing owner_id is provided, link them as owner
	if req.OwnerID != "" {
		ownerUUID, err := uuid.Parse(req.OwnerID)
		if err != nil {
			http.Error(w, "Invalid owner_id UUID", http.StatusBadRequest)
			return
		}
		tenant, err = s.db.SetTenantOwner(r.Context(), db.SetTenantOwnerParams{
			ID:      tenant.ID,
			OwnerID: uuid.NullUUID{UUID: ownerUUID, Valid: true},
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(tenant)
}

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.db.ListUsers(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Return only safe fields (no password hashes)
	type safeUser struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	var result []safeUser
	for _, u := range users {
		result = append(result, safeUser{
			ID:    u.ID.String(),
			Email: u.Email,
			Role:  u.Role,
		})
	}
	if result == nil {
		result = []safeUser{}
	}
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20) // 10MB
	if err != nil {
		http.Error(w, "File too large", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Invalid file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// In production, ensure the bucket exists
	url, err := s.storage.UploadFile(r.Context(), "cattlehof", header.Filename, file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"url": url})
}

func (s *Server) handleListProducts(w http.ResponseWriter, r *http.Request) {
	tenant := r.Context().Value(TenantContextKey).(db.Tenant)
	products, err := s.db.ListProducts(r.Context(), tenant.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
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

func (s *Server) handlePlaceOrder(w http.ResponseWriter, r *http.Request) {
	// Implementation placeholder
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) handleUpdateTenantIcon(w http.ResponseWriter, r *http.Request) {
	tenant := r.Context().Value(TenantContextKey).(db.Tenant)

	var req struct {
		IconUrl string `json:"icon_url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input: "+err.Error(), http.StatusBadRequest)
		return
	}

	arg := db.UpdateTenantIconParams{
		ID:      tenant.ID,
		IconUrl: sql.NullString{String: req.IconUrl, Valid: req.IconUrl != ""},
	}

	updatedTenant, err := s.db.UpdateTenantIcon(r.Context(), arg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(updatedTenant)
}

func (s *Server) handleUpdateTenantAppearance(w http.ResponseWriter, r *http.Request) {
	tenant := r.Context().Value(TenantContextKey).(db.Tenant)

	var req struct {
		CoverUrl    string `json:"cover_url"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input: "+err.Error(), http.StatusBadRequest)
		return
	}

	arg := db.UpdateTenantAppearanceParams{
		ID:          tenant.ID,
		CoverUrl:    sql.NullString{String: req.CoverUrl, Valid: req.CoverUrl != ""},
		Description: sql.NullString{String: req.Description, Valid: req.Description != ""},
	}

	updatedTenant, err := s.db.UpdateTenantAppearance(r.Context(), arg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(updatedTenant)
}

func (s *Server) handleGetTenant(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	tenant, err := s.db.GetTenantBySlug(r.Context(), slug)
	if err != nil {
		http.Error(w, "Tenant not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(tenant)
}

func (s *Server) handleUpdateTenant(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid tenant ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input: "+err.Error(), http.StatusBadRequest)
		return
	}

	arg := db.UpdateTenantParams{
		ID:   id,
		Name: req.Name,
		Slug: req.Slug,
	}

	updatedTenant, err := s.db.UpdateTenant(r.Context(), arg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(updatedTenant)
}

func (s *Server) handleDeleteTenant(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid tenant ID", http.StatusBadRequest)
		return
	}

	err = s.db.DeleteTenant(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleSetTenantOwner(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid tenant ID", http.StatusBadRequest)
		return
	}

	var req struct {
		OwnerID string `json:"owner_id"` // empty string = clear owner
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	var ownerID uuid.NullUUID
	if req.OwnerID != "" {
		parsed, err := uuid.Parse(req.OwnerID)
		if err != nil {
			http.Error(w, "Invalid owner_id UUID", http.StatusBadRequest)
			return
		}
		ownerID = uuid.NullUUID{UUID: parsed, Valid: true}
	}

	tenant, err := s.db.SetTenantOwner(r.Context(), db.SetTenantOwnerParams{
		ID:      id,
		OwnerID: ownerID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(tenant)
}
