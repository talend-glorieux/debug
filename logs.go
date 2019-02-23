package main

import (
	"bufio"
	"context"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search"
	"github.com/docker/docker/api/types"
	log "github.com/sirupsen/logrus"
)

func (s *Server) buildLogsIndex() error {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	os.Mkdir(filepath.Join(cacheDir, applicationName), os.ModePerm)
	logsIndexFilePath := filepath.Join(cacheDir, applicationName, "logs.index")
	err = os.RemoveAll(logsIndexFilePath)
	if err != nil {
		log.Error("New index", err)
	}
	logsIndex, err := bleve.New(logsIndexFilePath, bleve.NewIndexMapping())
	if err != nil {
		log.Error("New index", err)
		return err
	}
	s.logsIndex = logsIndex
	containers, err := s.docker.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		log.Error("Logs container list", err)
		return err
	}
	var wg sync.WaitGroup

	for _, container := range containers {
		wg.Add(1)
		go func(containerID string) {
			defer wg.Done()
			logsReader, err := s.docker.ContainerLogs(
				context.Background(),
				containerID,
				types.ContainerLogsOptions{
					Timestamps: true,
					ShowStdout: true,
					ShowStderr: true,
				},
			)
			if err != nil {
				log.Error("Docker container logs", err)
				return
			}
			defer logsReader.Close()
			scanner := bufio.NewScanner(logsReader)
			i := 0
			for scanner.Scan() {
				id := fmt.Sprintf("%s_%s", containerID, strings.SplitN(scanner.Text()[8:], " ", 2)[0])
				logsIndex.Index(id, scanner.Text())
				i++
			}
			if err := scanner.Err(); err != nil {
				log.Error("Logs scanner:", err)
			}
		}(container.ID)
	}
	wg.Wait()
	log.Info("Done indexing logs")
	return nil
}

func (s *Server) handleLogs() http.HandlerFunc {
	go s.buildLogsIndex()
	var (
		init sync.Once
		tpl  *template.Template
		err  error
	)
	return func(w http.ResponseWriter, r *http.Request) {
		init.Do(func() {
			tpl, err = s.parseTemplate("logs.html")
		})
		if err != nil {
			log.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		q := r.URL.Query().Get("q")
		if q != "" {
			search := bleve.NewSearchRequest(bleve.NewMatchQuery(q))
			searchResults, err := s.logsIndex.Search(search)
			if err != nil {
				log.Error(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		err = tpl.ExecuteTemplate(w, "logs.html", nil)
		if err != nil {
			log.Error(err)
		}
	}
}

type logLine struct {
	ContainerID   string
	ContainerName string
	Log           string
}

func (s *Server) resolveLog(results search.DocumentMatchCollection) []logLine {
	logsReader, err := s.docker.ContainerLogs(
		context.Background(),
		containerID,
		types.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
		},
	)
	if err != nil {
		log.Error("Docker container logs", err)
		return
	}
	defer logsReader.Close()

}

func (s *Server) handleLogsEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log.Print("New Event listener")
		containersID := r.URL.Query()["containers_id"]

		if len(containersID) == 0 {
			containers, err := s.docker.ContainerList(ctx, types.ContainerListOptions{})
			if err != nil && err != context.Canceled {
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

		logsReaders := []io.Reader{}
		for _, containerID := range containersID {
			logsReader, err := s.docker.ContainerLogs(
				context.Background(),
				containerID,
				types.ContainerLogsOptions{
					Follow:     true,
					ShowStdout: true,
					ShowStderr: true,
				},
			)
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
		for scanner.Scan() {
			event := NewEvent("", scanner.Text())
			fmt.Fprint(w, event)
			f.Flush()
		}
		if err := scanner.Err(); err != nil {
			log.Fatal("reading standard input:", err)
		}
	}
}
