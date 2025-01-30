package main

import (
	"log"
	"net/http"

	"github.com/Modul-306/backend/auth"
	"github.com/gorilla/mux"
)

type BaseHandler struct {
	w        http.ResponseWriter
	r        *http.Request
	id       string
	username string
}

func newBaseHandler(w http.ResponseWriter, r *http.Request) BaseHandler {
	vars := mux.Vars(r)
	return BaseHandler{
		w:        w,
		r:        r,
		id:       vars["id"],
		username: auth.GetUsername(r),
	}
}

// Handler type definition
type HandlerFunc func(BaseHandler)

// Middleware to create BaseHandler
func withBaseHandler(handler HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := newBaseHandler(w, r)
		handler(h)
	}
}

// Combined middleware for auth and BaseHandler
func withAuthAndBase(handler HandlerFunc) http.HandlerFunc {
	return auth.IsAuthorized(withBaseHandler(handler))
}

func main() {
	router := mux.NewRouter()

	// Public list endpoints
	router.HandleFunc("/blogs", withBaseHandler(GetBlogs)).Methods("GET")
	router.HandleFunc("/products", withBaseHandler(GetProducts)).Methods("GET")

	// Protected list endpoints
	router.HandleFunc("/blogs", withAuthAndBase(CreateBlog)).Methods("POST")
	router.HandleFunc("/user", withAuthAndBase(GetUsers)).Methods("GET")
	router.HandleFunc("/products", withAuthAndBase(CreateProduct)).Methods("POST")
	router.HandleFunc("/order", withAuthAndBase(GetOrders)).Methods("GET")
	router.HandleFunc("/order", withAuthAndBase(CreateOrder)).Methods("POST")

	// Public endpoints
	router.HandleFunc("/auth/login", auth.Login).Methods("POST")
	router.HandleFunc("/auth/sign-up", auth.SignUp).Methods("POST")
	router.HandleFunc("/blogs/{id}", withBaseHandler(GetBlog)).Methods("GET")
	router.HandleFunc("/products/{id}", withBaseHandler(GetProduct)).Methods("GET")

	// Protected endpoints
	router.HandleFunc("/blogs/{id}", withAuthAndBase(DeleteBlog)).Methods("DELETE")
	router.HandleFunc("/blogs/{id}", withAuthAndBase(UpdateBlog)).Methods("UPDATE")
	router.HandleFunc("/user/{id}", withAuthAndBase(GetUser)).Methods("GET")
	router.HandleFunc("/user/{id}", withAuthAndBase(DeleteUser)).Methods("DELETE")
	router.HandleFunc("/products/{id}", withAuthAndBase(DeleteProduct)).Methods("DELETE")
	router.HandleFunc("/products/{id}", withAuthAndBase(UpdateProduct)).Methods("UPDATE")
	router.HandleFunc("/order/{id}", withAuthAndBase(GetOrder)).Methods("GET")
	router.HandleFunc("/order/{id}", withAuthAndBase(DeleteOrder)).Methods("DELETE")
	router.HandleFunc("/order/{id}", withAuthAndBase(UpdateOrder)).Methods("UPDATE")

	log.Fatal(http.ListenAndServe(":8000", router))
}

// List endpoints
func GetBlogs(h BaseHandler) {
	// Implementation for getting all blogs
}

func CreateBlog(h BaseHandler) {
	// Implementation for creating a blog
}

func GetProducts(h BaseHandler) {
	// Implementation for getting all products
}

func CreateProduct(h BaseHandler) {
	// Implementation for creating a product
}

func GetUsers(h BaseHandler) {
	// Implementation for getting all users
}

func GetOrders(h BaseHandler) {
	// Implementation for getting all orders
}

func CreateOrder(h BaseHandler) {
	// Implementation for creating an order
}

// Blog handlers
func GetBlog(h BaseHandler) {
	// Use h.id, h.w, h.r
}

func UpdateBlog(h BaseHandler) {
	// Use h.id, h.w, h.r
}

func DeleteBlog(h BaseHandler) {
	// Use h.id, h.w, h.r
}

// User handlers
func GetUser(h BaseHandler) {
	// Use h.id, h.w, h.r
}

func DeleteUser(h BaseHandler) {
	// Use h.id, h.w, h.r
}

// Product handlers
func GetProduct(h BaseHandler) {
	// Use h.id, h.w, h.r
}

func DeleteProduct(h BaseHandler) {
	// Use h.id, h.w, h.r
}

func UpdateProduct(h BaseHandler) {
	// Use h.id, h.w, h.r
}

// Order handlers
func GetOrder(h BaseHandler) {
	// Use h.id, h.w, h.r
}

func DeleteOrder(h BaseHandler) {
	// Use h.id, h.w, h.r
}

func UpdateOrder(h BaseHandler) {
	// Use h.id, h.w, h.r
}
