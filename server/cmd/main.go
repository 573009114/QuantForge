package main

import (
	"log"
	"net/http"

	"quantforge/server/internal/router"
)

func main() {
	h := router.New()
	log.Println("quantforge server listening on :8080")
	if err := http.ListenAndServe(":8080", h); err != nil {
		log.Fatal(err)
	}
}
