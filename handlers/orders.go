package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Modul-306/backend/db"
	"github.com/jackc/pgx/v5/pgtype"
)

type OrderRequest struct {
	Address     string `json:"address"`
	IsCompleted bool   `json:"is_completed"`
}

type OrderResponse struct {
	ID          int    `json:"id"`
	Address     string `json:"address"`
	UserID      int    `json:"user_id"`
	IsCompleted bool   `json:"is_completed"`
}

func GetOrders(h BaseHandler) {
	conn, err := db.CreateDBConnection()
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close(h.r.Context())

	orders, err := db.New(conn).GetOrders(h.r.Context())
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(h.w).Encode(orders)
}

func GetOrder(h BaseHandler) {
	conn, err := db.CreateDBConnection()
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close(h.r.Context())

	id, err := strconv.Atoi(h.id)
	if err != nil {
		http.Error(h.w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	order, err := db.New(conn).GetOrder(h.r.Context(), int32(id))
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(h.w).Encode(order)
}

func CreateOrder(h BaseHandler) {
	conn, err := db.CreateDBConnection()
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close(h.r.Context())

	var req OrderRequest
	if err := json.NewDecoder(h.r.Body).Decode(&req); err != nil {
		http.Error(h.w, err.Error(), http.StatusBadRequest)
		return
	}

	dbConn := db.New(conn)

	user, err := dbConn.GetUserByUsername(h.r.Context(), h.username)
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}

	order, err := dbConn.CreateOrder(h.r.Context(), db.CreateOrderParams{
		Address: req.Address,
		UserID:  user.ID,
	})
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.w.WriteHeader(http.StatusCreated)
	json.NewEncoder(h.w).Encode(order)
}

func UpdateOrder(h BaseHandler) {
	conn, err := db.CreateDBConnection()
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close(h.r.Context())

	var req OrderRequest
	if err := json.NewDecoder(h.r.Body).Decode(&req); err != nil {
		http.Error(h.w, err.Error(), http.StatusBadRequest)
		return
	}

	dbConn := db.New(conn)

	user, err := dbConn.GetUserByUsername(h.r.Context(), h.username)
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}

	id, err := strconv.Atoi(h.id)
	if err != nil {
		http.Error(h.w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	isCompleted := pgtype.Bool{}
	err = isCompleted.Scan(req.IsCompleted)
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusBadRequest)
		return
	}

	order, err := dbConn.UpdateOrder(h.r.Context(), db.UpdateOrderParams{
		ID:          int32(id),
		Address:     req.Address,
		UserID:      user.ID,
		IsCompleted: isCompleted,
	})
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(h.w).Encode(order)
}

func DeleteOrder(h BaseHandler) {
	conn, err := db.CreateDBConnection()
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close(h.r.Context())

	id, err := strconv.Atoi(h.id)
	if err != nil {
		http.Error(h.w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	order, err := db.New(conn).DeleteOrder(h.r.Context(), int32(id))
	if err != nil {
		http.Error(h.w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.w.WriteHeader(http.StatusNoContent)
	json.NewEncoder(h.w).Encode(order)
}
