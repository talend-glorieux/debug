package main

import (
	"log"
	"net/http"

	"github.com/gorilla/handlers"
	"glorieux.io/adapter"
)

func main() {
	server, err := NewServer()
	if err != nil {
		log.Fatal(err)
	}
	http.ListenAndServe(":4242", adapter.Adapt(
		server,
		adapter.Timing(),
		handlers.CompressHandler,
	))
}
