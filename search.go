package main

import (
	"context"
	"html/template"
	"net/http"
	"sync"

	"github.com/blevesearch/bleve"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	log "github.com/sirupsen/logrus"
)

func (s *Server) buildIndex() error {
	mapping := bleve.NewIndexMapping()
	var err error
	s.index, err = bleve.New("docker-console.bleve", mapping)
	if err != nil {
		return err
	}
	s.index.SetName("containers")

	containers, err := s.docker.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		return err
	}
	for _, container := range containers {
		err = s.index.Index(container.ID, container)
		if err != nil {
			return err
		}
	}
	docCount, _ := s.index.DocCount()
	log.Println("Index", docCount)
	return nil
}

func (s *Server) handleSearch() http.HandlerFunc {
	go s.buildIndex()
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
			tpl, err = s.parseTemplate("search.html")
		})
		if err != nil {
			log.Error(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		q := r.URL.Query().Get("q")
		if q != "" {
			search := bleve.NewSearchRequest(bleve.NewMatchQuery(q))
			searchResults, err := s.index.Search(search)
			if err != nil {
				log.Error(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			containerListOptions := types.ContainerListOptions{All: true, Filters: filters.NewArgs()}
			for _, r := range searchResults.Hits {
				containerListOptions.Filters.Add("id", r.ID)
			}

			containers, err := s.docker.ContainerList(context.Background(), containerListOptions)
			if err != nil {
				log.Error(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			containersResponse := make([]container, len(containers))
			for index, c := range containers {
				containersResponse[index] = container{
					ID:   c.ID,
					Name: c.Names[0][1:],
				}
			}
			err = tpl.ExecuteTemplate(w, "search.html", containersResponse)
		} else {
			err = tpl.ExecuteTemplate(w, "search.html", nil)
		}
		if err != nil {
			log.Error(err)
		}
	}
}
