package main

import (
	"log"
	"net/http"

	"github.com/Modul-306/backend/router"
)

func main() {
	router := router.CreateRouter()

	log.Fatal(http.ListenAndServe(":8000", router))
}
