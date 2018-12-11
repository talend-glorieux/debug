package main

import (
	"context"
	"html/template"
	"net/http"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/sirupsen/logrus"
)

func (s *Server) handleImages() http.HandlerFunc {
	var (
		init sync.Once
		tpl  *template.Template
		err  error
	)
	return func(w http.ResponseWriter, r *http.Request) {
		init.Do(func() {
			tpl, err = template.ParseFiles("templates/partials.html", "templates/images.html")
		})
		if err != nil {
			logrus.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		images, err := s.docker.ImageList(ctx, types.ImageListOptions{})
		if err != nil {
			logrus.Error("Docker images list", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = tpl.ExecuteTemplate(w, "images.html", images)
		if err != nil {
			logrus.Error(err)
		}
	}
}

func (s *Server) handleImage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {}
}
