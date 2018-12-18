package main

import (
	"context"
	"html/template"
	"net/http"
	"sort"
	"sync"

	"github.com/docker/docker/api/types/filters"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

func (s *Server) handleVolumes() http.HandlerFunc {
	var (
		init sync.Once
		tpl  *template.Template
		err  error
	)
	type volume struct {
		ID      string
		Name    string
		Created string
		Size    int
	}
	return func(w http.ResponseWriter, r *http.Request) {
		init.Do(func() {
			tpl, err = s.parseTemplate("volumes.html")
		})
		if err != nil {
			logrus.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		volumesResp, err := s.docker.VolumeList(context.Background(), filters.NewArgs())
		if err != nil {
			logrus.Error("Docker volumes list", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		volumesResponse := make([]volume, len(volumesResp.Volumes))
		for index, vol := range volumesResp.Volumes {
			volumesResponse[index] = volume{
				Name:    vol.Name,
				Created: vol.CreatedAt,
				// Size:    int(vol.UsageData.Size),
			}
			log.Println(vol.UsageData)
		}

		sort.Slice(volumesResponse, func(i, j int) bool { return volumesResponse[i].Name < volumesResponse[j].Name })

		err = tpl.ExecuteTemplate(w, "volumes.html", volumesResponse)
		if err != nil {
			logrus.Error(err)
		}

	}
}
