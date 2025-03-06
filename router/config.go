package router

import (
	"github.com/Modul-306/backend/auth"
	h "github.com/Modul-306/backend/handlers"
	"github.com/gorilla/mux"
)

func CreateRouter() *mux.Router {
	router := mux.NewRouter()

	// Auth endpoints
	router.HandleFunc("/api/v1/auth/login", auth.Login).Methods("POST")
	router.HandleFunc("/api/v1/auth/sign-up", auth.SignUp).Methods("POST")

	// Blog endpoints
	router.HandleFunc("/api/v1/blogs", h.WithBaseHandler(h.GetBlogs)).Methods("GET")
	router.HandleFunc("/api/v1/blogs/{id}", h.WithBaseHandler(h.GetBlog)).Methods("GET")
	router.HandleFunc("/api/v1/blogs", h.WithAuthAndBase(h.CreateBlog)).Methods("POST")
	router.HandleFunc("/api/v1/blogs/{id}", h.WithAuthAndBase(h.UpdateBlog)).Methods("UPDATE")
	router.HandleFunc("/api/v1/blogs/{id}", h.WithAuthAndBase(h.DeleteBlog)).Methods("DELETE")

	// User endpoints
	router.HandleFunc("/api/v1/user", h.WithAuthAndBase(h.GetUsers)).Methods("GET")
	router.HandleFunc("/api/v1/user/{id}", h.WithAuthAndBase(h.GetUser)).Methods("GET")
	router.HandleFunc("/api/v1/user/{id}", h.WithAuthAndBase(h.DeleteUser)).Methods("DELETE")
	router.HandleFunc("/api/v1/user/{id}", h.WithAuthAndBase(h.UpdateUser)).Methods("UPDATE")

	// Product endpoints
	router.HandleFunc("/api/v1/products", h.WithBaseHandler(h.GetProducts)).Methods("GET")
	router.HandleFunc("/api/v1/products/{id}", h.WithBaseHandler(h.GetProduct)).Methods("GET")
	router.HandleFunc("/api/v1/products", h.WithAuthAndBase(h.CreateProduct)).Methods("POST")
	router.HandleFunc("/api/v1/products/{id}", h.WithAuthAndBase(h.UpdateProduct)).Methods("UPDATE")
	router.HandleFunc("/api/v1/products/{id}", h.WithAuthAndBase(h.DeleteProduct)).Methods("DELETE")

	// Order endpoints
	router.HandleFunc("/api/v1/order", h.WithAuthAndBase(h.GetOrders)).Methods("GET")
	router.HandleFunc("/api/v1/order/{id}", h.WithAuthAndBase(h.GetOrder)).Methods("GET")
	router.HandleFunc("/api/v1/order", h.WithAuthAndBase(h.CreateOrder)).Methods("POST")
	router.HandleFunc("/api/v1/order/{id}", h.WithAuthAndBase(h.UpdateOrder)).Methods("UPDATE")
	router.HandleFunc("/api/v1/order/{id}", h.WithAuthAndBase(h.DeleteOrder)).Methods("DELETE")

	return router
}
