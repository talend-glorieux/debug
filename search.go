package main

import (
	"context"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	log "github.com/sirupsen/logrus"
)

const (
	containersIndexName = "containers"
	imagesIndexName     = "images"
)

func splitResultByTypes(results search.DocumentMatchCollection) (containers []string, images []string) {
	for _, result := range results {
		switch result.Index {
		case containersIndexName:
			containers = append(containers, result.ID)
		case imagesIndexName:
			images = append(images, result.ID)
		default:
			log.Error("Unknown index type")
		}
	}
	return
}

func (s *Server) resolveContainers(containersID ...string) ([]types.Container, error) {
	containerListOptions := types.ContainerListOptions{All: true, Filters: filters.NewArgs()}
	for _, id := range containersID {
		containerListOptions.Filters.Add("id", id)
	}
	return s.docker.ContainerList(context.Background(), containerListOptions)
}

func (s *Server) resolveImages(imagesID ...string) ([]types.ImageInspect, error) {
	log.Warn(imagesID)
	images := make([]types.ImageInspect, len(imagesID))
	for index, id := range imagesID {
		image, _, err := s.docker.ImageInspectWithRaw(context.Background(), id)
		if err == nil {
			images[index] = image
		}
	}
	return images, nil
}

func (s *Server) buildIndex() error {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	os.Mkdir(filepath.Join(cacheDir, applicationName), os.ModePerm)
	containerIndexFilePath := filepath.Join(cacheDir, applicationName, "containers.index")
	err = os.RemoveAll(containerIndexFilePath)
	if err != nil {
		log.Error("New index", err)
	}
	containersIndex, err := bleve.New(containerIndexFilePath, bleve.NewIndexMapping())
	if err != nil {
		log.Error("New index", err)
		return err
	}
	containersIndex.SetName(containersIndexName)
	containers, err := s.docker.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		log.Error("Search container list", err)
		return err
	}
	for _, container := range containers {
		err = containersIndex.Index(container.ID, container)
		if err != nil {
			log.Error("Container index error", err)
			return err
		}
	}

	imagesIndexFilePath := filepath.Join(cacheDir, applicationName, "images.index")
	err = os.RemoveAll(imagesIndexFilePath)
	if err != nil {
		log.Error("New index", err)
	}
	imagesIndex, err := bleve.New(imagesIndexFilePath, bleve.NewIndexMapping())
	if err != nil {
		log.Error("New index", err)
		return err
	}
	imagesIndex.SetName(imagesIndexName)
	images, err := s.docker.ImageList(context.Background(), types.ImageListOptions{})
	if err != nil {
		log.Error("Search images list", err)
		return err
	}
	for _, image := range images {
		err = imagesIndex.Index(image.ID, image)
		if err != nil {
			log.Error("Images index error", err)
			return err
		}
	}

	s.index = bleve.NewIndexAlias(containersIndex, imagesIndex)
	docCount, _ := s.index.DocCount()
	log.Infof("%d containers indexed.", docCount)
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
		ID   string
		Name string
	}
	type image struct {
		ID   string
		Name string
	}
	type searchResponse struct {
		Hits       int
		Containers []container
		Images     []image
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
			containersID, imagesID := splitResultByTypes(searchResults.Hits)
			containers, err := s.resolveContainers(containersID...)
			if err != nil {
				log.Error(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			images, err := s.resolveImages(imagesID...)
			if err != nil {
				log.Error(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			searchResponse := searchResponse{
				Hits:       int(searchResults.Total),
				Containers: make([]container, len(containers)),
				Images:     make([]image, len(images)),
			}
			for index, c := range containers {
				searchResponse.Containers[index] = container{ID: c.ID, Name: c.Names[0][1:]}
			}
			for index, img := range images {
				searchResponse.Images[index] = image{ID: img.ID, Name: ""}
				if len(img.RepoTags) > 0 {
					searchResponse.Images[index].Name = img.RepoTags[0]
				}
			}

			err = tpl.ExecuteTemplate(w, "search.html", searchResponse)
		} else {
			err = tpl.ExecuteTemplate(w, "search.html", nil)
		}
		if err != nil {
			log.Error(err)
		}
	}
}
