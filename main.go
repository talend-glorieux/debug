package main

import (
	"log"
	"net/http"

	"github.com/gorilla/handlers"
	"github.com/sirupsen/logrus"
	"github.com/skratchdot/open-golang/open"
	"glorieux.io/adapter"
)

func main() {
	server, err := NewServer()
	if err != nil {
		log.Fatal(err)
	}
	err = open.Run("http://localhost:4242")
	if err != nil {
		logrus.Error("Can't launch browser at https://localhost:4242")
	}
	http.ListenAndServe(":4242", adapter.Adapt(
		server,
		adapter.Timing(),
		handlers.CompressHandler,
	))
}
