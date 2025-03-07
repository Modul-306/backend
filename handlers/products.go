package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Modul-306/backend/db"
	"github.com/jackc/pgx/v5/pgtype"
)

type ProductRequest struct {
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	ImageURL    string  `json:"image_url"`
	IsAvailable bool    `json:"is_available"`
}

type ProductResponse struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	ImageURL    string  `json:"image_url"`
	IsAvailable bool    `json:"is_available"`
}

func GetProducts(h BaseHandler) {
	conn, err := db.CreateDBConnection()
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close(h.r.Context())

	products, err := db.New(conn).GetProducts(h.r.Context())
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(h.w).Encode(products)
}

func GetProduct(h BaseHandler) {
	conn, err := db.CreateDBConnection()
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close(h.r.Context())

	id, err := strconv.Atoi(h.id)
	if err != nil {
		http.Error(h.w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	product, err := db.New(conn).GetProduct(h.r.Context(), int32(id))
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(h.w).Encode(product)
}

func CreateProduct(h BaseHandler) {
	conn, err := db.CreateDBConnection()
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close(h.r.Context())

	var req ProductRequest
	if err := json.NewDecoder(h.r.Body).Decode(&req); err != nil {
		http.Error(h.w, err.Error(), http.StatusBadRequest)
		return
	}

	var price pgtype.Numeric
	err = price.Scan(fmt.Sprintf("%.2f", req.Price))
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusBadRequest)
		return
	}

	isAvailable := pgtype.Bool{}
	err = isAvailable.Scan(req.IsAvailable)
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusBadRequest)
		return
	}

	product, err := db.New(conn).CreateProduct(h.r.Context(), db.CreateProductParams{
		Name:        req.Name,
		Price:       price,
		ImageUrl:    req.ImageURL,
		IsAvailable: isAvailable,
	})
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.w.WriteHeader(http.StatusCreated)
	json.NewEncoder(h.w).Encode(product)
}

func UpdateProduct(h BaseHandler) {
	conn, err := db.CreateDBConnection()
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close(h.r.Context())

	var req ProductRequest
	if err := json.NewDecoder(h.r.Body).Decode(&req); err != nil {
		http.Error(h.w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(h.id)
	if err != nil {
		http.Error(h.w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	var price pgtype.Numeric
	err = price.Scan(fmt.Sprintf("%.2f", req.Price))
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusBadRequest)
		return
	}

	isAvailable := pgtype.Bool{}
	err = isAvailable.Scan(req.IsAvailable)
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusBadRequest)
		return
	}

	product, err := db.New(conn).UpdateProduct(h.r.Context(), db.UpdateProductParams{
		ID:          int32(id),
		Name:        req.Name,
		Price:       price,
		ImageUrl:    req.ImageURL,
		IsAvailable: isAvailable,
	})
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(h.w).Encode(product)
}

func DeleteProduct(h BaseHandler) {
	conn, err := db.CreateDBConnection()
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close(h.r.Context())

	id, err := strconv.Atoi(h.id)
	if err != nil {
		http.Error(h.w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	productsDB := db.New(conn)

	user, err := productsDB.GetUserByUsername(h.r.Context(), h.username)
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}

	product, err := productsDB.DeleteProduct(h.r.Context(), db.DeleteProductParams{ID: int32(id), Column2: user.ID})
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.w.WriteHeader(http.StatusNoContent)
	json.NewEncoder(h.w).Encode(product)
}
