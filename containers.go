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

// TODO
//func (cli *Client) ContainerStats(ctx context.Context, containerID string, stream bool) (types.ContainerStats, error)
// func (cli *Client) ContainerTop(ctx context.Context, containerID string, arguments []string) (container.ContainerTopOKBody, error)
func (s *Server) handleContainers() http.HandlerFunc {
	var (
		init sync.Once
		tpl  *template.Template
		err  error
	)
	return func(w http.ResponseWriter, r *http.Request) {
		init.Do(func() {
			tpl, err = template.ParseFiles("templates/partials.html", "templates/containers.html")
		})
		if err != nil {
			logrus.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		containers, err := s.docker.ContainerList(ctx, types.ContainerListOptions{All: true})
		if err != nil {
			logrus.Error("Docker containers list", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = tpl.ExecuteTemplate(w, "containers.html", containers)
		if err != nil {
			logrus.Error(err)
		}
	}
}

func (s *Server) handleContainer() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {}
}
