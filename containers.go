package main

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

func (s *Server) handleContainers() http.HandlerFunc {
	var (
		init sync.Once
		tpl  *template.Template
		err  error
	)
	type container struct {
		ID          string
		Name        string
		Image       string
		ImageID     string
		StatusColor string
	}
	return func(w http.ResponseWriter, r *http.Request) {
		init.Do(func() {
			tpl, err = s.parseTemplate("containers.html")
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

		containersResponse := make([]container, len(containers))
		for index, c := range containers {
			containersResponse[index] = container{
				ID:      c.ID,
				Name:    c.Names[0][1:],
				Image:   c.Image,
				ImageID: c.ImageID,
			}
			switch c.State {
			case "paused":
				containersResponse[index].StatusColor = "yellow"
			case "running":
				containersResponse[index].StatusColor = "green"
			default:
				containersResponse[index].StatusColor = "red"
			}
		}

		sort.Slice(containersResponse, func(i, j int) bool { return containersResponse[i].Name < containersResponse[j].Name })

		err = tpl.ExecuteTemplate(w, "containers.html", containersResponse)
		if err != nil {
			logrus.Error(err)
		}
	}
}

// TODO
//func (cli *Client) ContainerStats(ctx context.Context, containerID string, stream bool) (types.ContainerStats, error)
// func (cli *Client) ContainerTop(ctx context.Context, containerID string, arguments []string) (container.ContainerTopOKBody, error)
func (s *Server) handleContainer() http.HandlerFunc {
	var (
		init sync.Once
		tpl  *template.Template
		err  error
	)
	type containerResponse struct {
		Name            string
		State           string
		RestartCount    int
		Created         string
		Command         string
		ImageID         string
		ResolvConfPath  string
		HostnamePath    string
		HostsPath       string
		LogPath         string
		AppArmorProfile string

		TopTitles    []string
		TopProcesses [][]string
	}
	return func(w http.ResponseWriter, r *http.Request) {
		init.Do(func() {
			tpl, err = s.parseTemplate("container.html")
		})
		if err != nil {
			logrus.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		containerID := mux.Vars(r)["id"]
		container, err := s.docker.ContainerInspect(context.Background(), containerID)
		if err != nil {
			logrus.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response := &containerResponse{
			Name:            container.Name[1:],
			State:           container.State.Status,
			RestartCount:    container.RestartCount,
			Created:         container.Created,
			Command:         fmt.Sprintf("%s %s", container.Path, strings.Join(container.Args, "")),
			ImageID:         container.Image,
			ResolvConfPath:  container.ResolvConfPath,
			HostnamePath:    container.HostnamePath,
			HostsPath:       container.HostsPath,
			LogPath:         container.LogPath,
			AppArmorProfile: container.AppArmorProfile,
		}

		if container.State.Status == "running" {
			top, err := s.docker.ContainerTop(context.Background(), containerID, []string{})
			if err != nil {
				logrus.Error(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			response.TopTitles = top.Titles
			response.TopProcesses = top.Processes
		}

		err = tpl.ExecuteTemplate(w, "container.html", response)
		if err != nil {
			logrus.Error(err)
		}
	}

}
