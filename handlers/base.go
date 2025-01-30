package handlers

import (
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

func NewBaseHandler(w http.ResponseWriter, r *http.Request) BaseHandler {
	vars := mux.Vars(r)
	return BaseHandler{
		w:        w,
		r:        r,
		id:       vars["id"],
		username: auth.GetUsername(r),
	}
}

// HandlerFunc is a function that takes a BaseHandler
type HandlerFunc func(BaseHandler)

// WithBaseHandler wraps a HandlerFunc with BaseHandler creation
func WithBaseHandler(handler HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := NewBaseHandler(w, r)
		handler(h)
	}
}

// WithAuthAndBase combines auth check and BaseHandler creation
func WithAuthAndBase(handler HandlerFunc) http.HandlerFunc {
	return auth.IsAuthorized(WithBaseHandler(handler))
}
