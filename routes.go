package main

import (
	"net/http"

	"github.com/gobuffalo/packr"
)

func (s *Server) routes() {
	assets := packr.NewBox("./assets")
	s.router.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", http.FileServer(assets)))

	s.router.HandleFunc("/images", s.handleImages()).Methods(http.MethodGet)
	s.router.HandleFunc("/images", s.handleImagesClean()).Methods(http.MethodDelete)
	s.router.HandleFunc("/images/{id}", s.handleImage()).Methods(http.MethodGet)

	s.router.HandleFunc("/containers", s.handleContainers()).Methods(http.MethodGet)
	s.router.HandleFunc("/containers/{id}", s.handleContainer()).Methods(http.MethodGet)

	s.router.HandleFunc("/volumes", s.handleVolumes()).Methods(http.MethodGet)

	s.router.HandleFunc("/logs", s.handleLogs()).Methods(http.MethodGet)
	s.router.HandleFunc("/logs/events", s.handleLogsEvents())

	s.router.HandleFunc("/search", s.handleSearch()).Methods(http.MethodGet)

	s.router.HandleFunc("/events", s.handleEvents())

	s.router.HandleFunc("/", s.handleIndex()).Methods(http.MethodGet)
}
