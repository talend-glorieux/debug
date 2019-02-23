package main

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"sync"

	"github.com/blevesearch/bleve"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/gobuffalo/packr"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

type Server struct {
	router    *mux.Router
	templates packr.Box
	docker    *client.Client
	index     bleve.Index
	logsIndex bleve.Index
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

func sumVolumesSize(volumes []*types.Volume) (sum int) {
	for i := range volumes {
		sum += int(volumes[i].UsageData.Size)
	}
	return sum
}

func sumImagesSize(images []*types.ImageSummary) (sum int) {
	for i := range images {
		sum += int(images[i].Size)
	}
	return sum
}

func sumContainersSize(containers []*types.Container) (sum int) {
	for i := range containers {
		sum += int(containers[i].SizeRootFs)
	}
	return sum
}

func (s *Server) handleIndex() http.HandlerFunc {
	var (
		init           sync.Once
		tpl            *template.Template
		err            error
		diskUsageCache types.DiskUsage
	)
	type response struct {
		Info           types.Info
		LayersSize     int
		VolumesSize    int
		ImagesSize     int
		ContainersSize int
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		init.Do(func() {
			tpl, err = s.parseTemplate("index.html")
		})
		if err != nil {
			logrus.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		diskUsage, err := s.docker.DiskUsage(ctx)
		if err != nil && err != context.Canceled {
			logrus.Error("Docker disk usage", err)
			diskUsage = diskUsageCache
		} else {
			diskUsageCache = diskUsage
		}

		info, err := s.docker.Info(ctx)
		if err != nil && err != context.Canceled {
			logrus.Error("Docker info", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = tpl.ExecuteTemplate(w, "index.html", response{
			Info:           info,
			LayersSize:     int(diskUsage.LayersSize),
			VolumesSize:    sumVolumesSize(diskUsage.Volumes),
			ImagesSize:     sumImagesSize(diskUsage.Images),
			ContainersSize: sumContainersSize(diskUsage.Containers),
		})
		if err != nil {
			logrus.Error(err)
		}
	}
}

func (s *Server) handleEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log.Print("New events listener")
		lastEventID := r.Header.Get("Last-Event-ID")
		if lastEventID != "" {
			log.Printf("Last event ID: %s", lastEventID)
		}

		// TODO: If not headers send all events?
		f, ok := w.(http.Flusher)
		if !ok {
			log.Error("Streaming unsupported!")
			http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Transfer-Encoding", "chunked")
		w.Header().Set("Expire", "0")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

		notify := w.(http.CloseNotifier).CloseNotify()
		go func() {
			<-notify
			log.Println("HTTP connection just closed.")
		}()

		eventChan, errChan := s.docker.Events(ctx, types.EventsOptions{
			Filters: filters.NewArgs(
				filters.Arg("type", "container"),
				filters.Arg("type", "image"),
				filters.Arg("event", "start"),
				filters.Arg("event", "stop"),
			),
		})
		for {
			select {
			case msg, ok := <-eventChan:
				if !ok {
					return
				}
				fmt.Println("received message", msg.Type, msg.Action, msg.Actor.Attributes)
				event := &Event{
					ID:   msg.Actor.ID,
					Data: "New Message",
				}
				fmt.Fprint(w, event)
				f.Flush()
			case err, ok := <-errChan:
				if !ok {
					return
				}
				log.Error(err)
				f.Flush()
			}
		}
	}
}
