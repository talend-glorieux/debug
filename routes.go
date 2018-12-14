package main

import (
	"net/http"

	"github.com/gobuffalo/packr"
)

func (s *Server) routes() {
	assets := packr.NewBox("./assets")
	s.router.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", http.FileServer(assets)))
	s.router.HandleFunc("/images", s.handleImages())
	s.router.HandleFunc("/images/{id}", s.handleImage())
	s.router.HandleFunc("/containers", s.handleContainers())
	s.router.HandleFunc("/containers/{id}", s.handleContainer())
	s.router.HandleFunc("/logs", s.handleLogs())
	s.router.HandleFunc("/logs/events", s.handleLogsEvents())
	s.router.HandleFunc("/search", s.handleSearch())
	s.router.HandleFunc("/events", s.handleEvents())
	s.router.HandleFunc("/", s.handleIndex())
}
