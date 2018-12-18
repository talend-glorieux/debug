package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/handlers"
	"github.com/sirupsen/logrus"
	"github.com/skratchdot/open-golang/open"
	"glorieux.io/adapter"
)

const applicationName = "docker-console"

// Version is the application version
// Set at build time
var Version string

func main() {
	showVersion := flag.Bool("version", false, fmt.Sprintf("Show %s version.", applicationName))
	flag.Parse()
	if *showVersion {
		fmt.Println(Version)
		return
	}

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
