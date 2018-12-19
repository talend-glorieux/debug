package main

import (
	"html/template"
	"net/http"
	"sort"
	"sync"

	"github.com/sirupsen/logrus"
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
		ctx := r.Context()
		init.Do(func() {
			tpl, err = s.parseTemplate("volumes.html")
		})
		if err != nil {
			logrus.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		diskUsage, err := s.docker.DiskUsage(ctx)
		if err != nil {
			logrus.Error("Docker disk usage", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		volumesResponse := make([]volume, len(diskUsage.Volumes))
		for index, vol := range diskUsage.Volumes {
			volumesResponse[index] = volume{
				Name:    vol.Name,
				Created: vol.CreatedAt,
				Size:    int(vol.UsageData.Size),
			}
		}

		sort.Slice(volumesResponse, func(i, j int) bool {
			return volumesResponse[i].Size > volumesResponse[j].Size
		})

		err = tpl.ExecuteTemplate(w, "volumes.html", volumesResponse)
		if err != nil {
			logrus.Error(err)
		}
	}
}
