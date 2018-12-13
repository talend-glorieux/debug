package main

import (
	"context"
	"html/template"
	"net/http"
	"sync"
	"time"

	"github.com/docker/docker/client"
	"github.com/gobuffalo/packr"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type Server struct {
	router    *mux.Router
	docker    *client.Client
	templates packr.Box
}

func NewServer() (*Server, error) {
	dockerClient, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}
	s := &Server{
		router:    mux.NewRouter(),
		docker:    dockerClient,
		templates: packr.NewBox("./templates"),
	}
	s.routes()

	return s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) parseTemplate(name string) (*template.Template, error) {
	partialsFile, err := s.templates.FindString("partials.html")
	if err != nil {
		return nil, err
	}
	templateFile, err := s.templates.FindString(name)
	if err != nil {
		return nil, err
	}

	partialsTemplate, err := template.New("partials").Parse(partialsFile)
	if err != nil {
		return nil, err
	}
	return partialsTemplate.New(name).Parse(templateFile)
}

func (s *Server) handleIndex() http.HandlerFunc {
	var (
		init sync.Once
		tpl  *template.Template
		err  error
	)
	return func(w http.ResponseWriter, r *http.Request) {
		init.Do(func() {
			tpl, err = s.parseTemplate("index.html")
		})
		if err != nil {
			logrus.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		info, err := s.docker.Info(ctx)
		if err != nil {
			logrus.Error("Docker info", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// TODO: Add DiskUsage
		err = tpl.ExecuteTemplate(w, "index.html", info)
		if err != nil {
			logrus.Error(err)
		}
	}
}
