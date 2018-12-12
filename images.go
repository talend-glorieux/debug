package main

import (
	"context"
	"html/template"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/gorilla/mux"
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
	var (
		init sync.Once
		tpl  *template.Template
		err  error
	)
	type imageResponse struct {
		Name               string
		Parent             string
		Comment            string
		Created            string
		Container          string
		DockerVersion      string
		Author             string
		Config             string
		Architecture       string
		Os                 string
		OsVersion          string
		Size               int
		ConfigUser         string
		ConfigExposedPorts []string
		ConfigEnv          []string
		ConfigEntrypoint   string
		ConfigCmd          string
		ConfigVolumes      []string
		ConfigLabels       map[string]string
	}
	return func(w http.ResponseWriter, r *http.Request) {
		init.Do(func() {
			tpl, err = template.ParseFiles("templates/partials.html", "templates/image.html")
		})
		if err != nil {
			logrus.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		imageID := mux.Vars(r)["id"]
		image, _, err := s.docker.ImageInspectWithRaw(context.Background(), imageID)
		if err != nil {
			logrus.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response := imageResponse{
			Name:             strings.Join(image.RepoTags, ""),
			Parent:           image.Parent,
			Comment:          image.Comment,
			Created:          image.Created,
			Container:        image.Container,
			DockerVersion:    image.DockerVersion,
			Author:           image.Author,
			Architecture:     image.Architecture,
			Os:               image.Os,
			OsVersion:        image.OsVersion,
			Size:             int(image.Size),
			ConfigUser:       image.Config.User,
			ConfigEnv:        image.Config.Env,
			ConfigEntrypoint: strings.Join(image.Config.Entrypoint, " "),
			ConfigCmd:        strings.Join(image.Config.Cmd, " "),
			ConfigLabels:     image.Config.Labels,
		}

		for port := range image.Config.ExposedPorts {
			response.ConfigExposedPorts = append(response.ConfigExposedPorts, string(port))
		}
		for volume := range image.Config.Volumes {
			response.ConfigVolumes = append(response.ConfigVolumes, volume)
		}

		err = tpl.ExecuteTemplate(w, "image.html", response)
		if err != nil {
			logrus.Error(err)
		}
	}
}
