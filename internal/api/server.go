package api

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/M306/backend/internal/db/sqlc"
	"github.com/M306/backend/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type Server struct {
	db            *db.Queries
	conn          *sql.DB
	storage       storage.Storage
	notifications NotificationService
	jwtSecret     string
}

func NewServer(queries *db.Queries, conn *sql.DB, storage storage.Storage, notifications NotificationService, jwtSecret string) *Server {
	return &Server{
		db:            queries,
		conn:          conn,
		storage:       storage,
		notifications: notifications,
		jwtSecret:     jwtSecret,
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
		r.Get("/tenants/{slug}", s.handleGetTenant) // Moved out of middleware
		r.Get("/tenants/categories", s.handleListTenantCategories)
		
		r.Group(func(r chi.Router) {
			r.Use(s.TenantMiddleware)
			r.Get("/products", s.handleListProducts)
			r.Get("/categories", s.handleListCategories)
			r.Get("/blogs", s.handleListBlogs)
			r.Get("/blogs/{id}", s.handleGetBlog)
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
				r.Post("/products/{id}/reviews", s.handleCreateReview)
			})
		})

		r.Group(func(r chi.Router) {
			r.Use(s.TenantMiddleware)
			r.Get("/products/{id}", s.handleGetProduct)
			r.Get("/products/{id}/reviews", s.handleListReviews)
			r.Get("/products/{id}/reviews/stats", s.handleGetReviewStats)
		})

		// --- Global Admin Routes ---
		r.Group(func(r chi.Router) {
			r.Use(s.JWTMiddleware)
			r.Use(s.RequireRole("platform_admin"))
			
			r.Post("/tenants", s.handleCreateTenant)
			r.Put("/tenants/{id}", s.handleUpdateTenant)
			r.Put("/tenants/{id}/owner", s.handleSetTenantOwner)
			r.Post("/tenants/{id}/owners", s.handleAddTenantOwner)
			r.Delete("/tenants/{id}/owners/{userID}", s.handleRemoveTenantOwner)
			r.Get("/tenants/{id}/owners", s.handleListTenantOwners)
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
			r.Get("/orders", s.handleListOrders)
			r.Put("/orders/{id}/status", s.handleUpdateOrderStatus)
			r.Get("/analytics/revenue", s.handleGetRevenueAnalytics)
			r.Get("/analytics/top-products", s.handleGetTopProducts)
			r.Get("/categories", s.handleListCategories)
			r.Get("/loyalty", s.handleGetUserLoyalty)
			
			r.Group(func(r chi.Router) {
				r.Use(s.RequireRole("farmer_admin", "platform_admin", "staff"))
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
