package main

import (
	"bufio"
	"context"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	log "github.com/sirupsen/logrus"
)

func (s *Server) handleLogs() http.HandlerFunc {
	var (
		init sync.Once
		tpl  *template.Template
		err  error
	)
	return func(w http.ResponseWriter, r *http.Request) {
		init.Do(func() {
			tpl, err = template.ParseFiles("templates/partials.html", "templates/logs.html")
		})
		if err != nil {
			log.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = tpl.ExecuteTemplate(w, "logs.html", nil)
		if err != nil {
			log.Error(err)
		}
	}
}

func (s *Server) handleLogsEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Print("New Event listener")

		containersID := r.URL.Query()["containers_id"]

		if len(containersID) == 0 {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			containers, err := s.docker.ContainerList(ctx, types.ContainerListOptions{})
			if err != nil {
				log.Error("Docker containers list", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if len(containers) == 0 {
				log.Error("No running container")
				http.Error(w, "No running container", http.StatusNotFound)
				return
			}

			for _, container := range containers {
				containersID = append(containersID, container.ID)
			}
		}

		log.Info("Containers", containersID)

		logsReaders := []io.Reader{}
		for _, containerID := range containersID {
			logsReader, err := s.docker.ContainerLogs(context.Background(), containerID, types.ContainerLogsOptions{Follow: true, ShowStdout: true, ShowStderr: true})
			if err != nil {
				log.Error("Docker container logs", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer logsReader.Close()
			logsReaders = append(logsReaders, logsReader)
		}

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

		notify := w.(http.CloseNotifier).CloseNotify()
		go func() {
			<-notify
			log.Println("HTTP connection just closed.")
			// for _, logsReader := range logsReaders {
			// 	err := logsReader.Close()
			// 	if err != nil {
			// 		log.Error(err)
			// 	}
			// }
		}()

		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Transfer-Encoding", "chunked")
		w.Header().Set("Expire", "0")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

		multiReader := io.MultiReader(logsReaders...)
		scanner := bufio.NewScanner(multiReader)
		log.Info("Scan")
		for scanner.Scan() {
			event := NewEvent("", scanner.Text())
			log.Info("E", event)
			fmt.Fprint(w, event)
			f.Flush()
		}
		if err := scanner.Err(); err != nil {
			log.Fatal("reading standard input:", err)
		}
	}
}
