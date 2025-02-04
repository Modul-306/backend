package main

import (
	"log"
	"net/http"

	"github.com/Modul-306/backend/auth"
	"github.com/Modul-306/backend/handlers"
	"github.com/gorilla/mux"
)

func main() {
	router := CreateRouter()

	log.Fatal(http.ListenAndServe(":8000", router))
}

func CreateRouter() *mux.Router {
	router := mux.NewRouter()

	// Auth endpoints
	router.HandleFunc("/api/v1/auth/login", auth.Login).Methods("POST")
	router.HandleFunc("/api/v1/auth/sign-up", auth.SignUp).Methods("POST")

	// Blog endpoints
	router.HandleFunc("/api/v1/blogs", handlers.WithBaseHandler(handlers.GetBlogs)).Methods("GET")
	router.HandleFunc("/api/v1/blogs/{id}", handlers.WithBaseHandler(handlers.GetBlog)).Methods("GET")
	router.HandleFunc("/api/v1/blogs", handlers.WithAuthAndBase(handlers.CreateBlog)).Methods("POST")
	router.HandleFunc("/api/v1/blogs/{id}", handlers.WithAuthAndBase(handlers.UpdateBlog)).Methods("UPDATE")
	router.HandleFunc("/api/v1/blogs/{id}", handlers.WithAuthAndBase(handlers.DeleteBlog)).Methods("DELETE")

	// User endpoints
	router.HandleFunc("/api/v1/user", handlers.WithAuthAndBase(handlers.GetUsers)).Methods("GET")
	router.HandleFunc("/api/v1/user/{id}", handlers.WithAuthAndBase(handlers.GetUser)).Methods("GET")
	router.HandleFunc("/api/v1/user/{id}", handlers.WithAuthAndBase(handlers.DeleteUser)).Methods("DELETE")
	router.HandleFunc("/api/v1/user/{id}", handlers.WithAuthAndBase(handlers.UpdateUser)).Methods("UPDATE")

	// Product endpoints
	router.HandleFunc("/api/v1/products", handlers.WithBaseHandler(handlers.GetProducts)).Methods("GET")
	router.HandleFunc("/api/v1/products/{id}", handlers.WithBaseHandler(handlers.GetProduct)).Methods("GET")
	router.HandleFunc("/api/v1/products", handlers.WithAuthAndBase(handlers.CreateProduct)).Methods("POST")
	router.HandleFunc("/api/v1/products/{id}", handlers.WithAuthAndBase(handlers.UpdateProduct)).Methods("UPDATE")
	router.HandleFunc("/api/v1/products/{id}", handlers.WithAuthAndBase(handlers.DeleteProduct)).Methods("DELETE")

	// Order endpoints
	router.HandleFunc("/api/v1/order", handlers.WithAuthAndBase(handlers.GetOrders)).Methods("GET")
	router.HandleFunc("/api/v1/order/{id}", handlers.WithAuthAndBase(handlers.GetOrder)).Methods("GET")
	router.HandleFunc("/api/v1/order", handlers.WithAuthAndBase(handlers.CreateOrder)).Methods("POST")
	router.HandleFunc("/api/v1/order/{id}", handlers.WithAuthAndBase(handlers.UpdateOrder)).Methods("UPDATE")
	router.HandleFunc("/api/v1/order/{id}", handlers.WithAuthAndBase(handlers.DeleteOrder)).Methods("DELETE")

	return router
}
