package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/Modul-306/backend/auth"
	"github.com/gorilla/mux"
)

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func main() {
	router := mux.NewRouter()

	// Public endpoints
	router.HandleFunc("/auth/login", Login).Methods("POST")
	router.HandleFunc("/auth/sign-up", SignUp).Methods("POST")
	router.HandleFunc("/blogs", GetBlogs).Methods("GET")
	router.HandleFunc("/blogs/{id}", GetBlog).Methods("GET")
	router.HandleFunc("/products", GetProducts).Methods("GET")
	router.HandleFunc("/products/{id}", GetProduct).Methods("GET")

	// Protected endpoints
	router.HandleFunc("/blogs", auth.IsAuthorized(CreateBlog)).Methods("POST")
	router.HandleFunc("/blogs/{id}", auth.IsAuthorized(DeleteBlog)).Methods("DELETE")
	router.HandleFunc("/blogs/{id}", auth.IsAuthorized(UpdateBlog)).Methods("UPDATE")
	router.HandleFunc("/user", auth.IsAuthorized(GetUsers)).Methods("GET")
	router.HandleFunc("/user/{id}", auth.IsAuthorized(GetUser)).Methods("GET")
	router.HandleFunc("/user/{id}", auth.IsAuthorized(DeleteUser)).Methods("DELETE")
	router.HandleFunc("/products", auth.IsAuthorized(CreateProduct)).Methods("POST")
	router.HandleFunc("/products/{id}", auth.IsAuthorized(DeleteProduct)).Methods("DELETE")
	router.HandleFunc("/products/{id}", auth.IsAuthorized(UpdateProduct)).Methods("UPDATE")
	router.HandleFunc("/order", auth.IsAuthorized(GetOrders)).Methods("GET")
	router.HandleFunc("/order", auth.IsAuthorized(CreateOrder)).Methods("POST")
	router.HandleFunc("/order/{id}", auth.IsAuthorized(GetOrder)).Methods("GET")
	router.HandleFunc("/order/{id}", auth.IsAuthorized(DeleteOrder)).Methods("DELETE")
	router.HandleFunc("/order/{id}", auth.IsAuthorized(UpdateOrder)).Methods("UPDATE")

	log.Fatal(http.ListenAndServe(":8000", router))
}

func Login(w http.ResponseWriter, r *http.Request) {
	var creds Credentials
	err := json.NewDecoder(r.Body).Decode(&creds)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Authenticate the user
	if creds.Username != "user" || creds.Password != "password" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	expirationTime := time.Now().Add(5 * time.Minute)

	tokenString, err := auth.CreateToken(creds.Username, expirationTime)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:    "token",
		Value:   tokenString,
		Expires: expirationTime,
	})
}

func SignUp(w http.ResponseWriter, r *http.Request) {
	// Implement sign-up logic
}

func GetBlogs(w http.ResponseWriter, r *http.Request) {
	// Implement get blogs logic
}

func CreateBlog(w http.ResponseWriter, r *http.Request) {
	// Implement create blog logic
}

func GetBlog(w http.ResponseWriter, r *http.Request) {
	// Implement get blog logic
}

func DeleteBlog(w http.ResponseWriter, r *http.Request) {
	// Implement delete blog logic
}

func UpdateBlog(w http.ResponseWriter, r *http.Request) {
	// Implement update blog logic
}

func GetUsers(w http.ResponseWriter, r *http.Request) {
	// Implement get users logic
}

func GetUser(w http.ResponseWriter, r *http.Request) {
	// Implement get user logic
}

func DeleteUser(w http.ResponseWriter, r *http.Request) {
	// Implement delete user logic
}

func GetProducts(w http.ResponseWriter, r *http.Request) {
	// Implement get products logic
}

func CreateProduct(w http.ResponseWriter, r *http.Request) {
	// Implement create product logic
}

func GetProduct(w http.ResponseWriter, r *http.Request) {
	// Implement get product logic
}

func DeleteProduct(w http.ResponseWriter, r *http.Request) {
	// Implement delete product logic
}

func UpdateProduct(w http.ResponseWriter, r *http.Request) {
	// Implement update product logic
}

func GetOrders(w http.ResponseWriter, r *http.Request) {
	// Implement get orders logic
}

func CreateOrder(w http.ResponseWriter, r *http.Request) {
	// Implement create order logic
}

func GetOrder(w http.ResponseWriter, r *http.Request) {
	// Implement get order logic
}

func DeleteOrder(w http.ResponseWriter, r *http.Request) {
	// Implement delete order logic
}

func UpdateOrder(w http.ResponseWriter, r *http.Request) {
	// Implement update order logic
}
