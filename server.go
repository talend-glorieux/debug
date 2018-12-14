package main

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/gobuffalo/packr"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
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

func (s *Server) handleEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		closeNotify := w.(http.CloseNotifier).CloseNotify()
		eventChan, errChan := s.docker.Events(context.Background(), types.EventsOptions{})
		for {
			select {
			case msg := <-eventChan:
				fmt.Println("received message", msg.Type, msg.Action, msg.Actor.Attributes)
				event := &Event{
					ID:   msg.Actor.ID,
					Type: fmt.Sprintf("%s-%s", msg.Type, msg.Action),
				}
				fmt.Fprint(w, event)
				f.Flush()
			case err := <-errChan:
				log.Error(err)
			case <-closeNotify:
				log.Println("HTTP connection just closed.")
				return
			}
		}
	}
}
