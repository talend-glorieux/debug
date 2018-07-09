//go:generate packr

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"path"
	"time"

	"github.com/gobuffalo/packr"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/talend-glorieux/debug"
)

func main() {
	var debugFlag = flag.Bool("debug", false, "")
	flag.Parse()

	if *debugFlag {
		log.SetLevel(log.DebugLevel)
	}

	collectors := make(map[string]debug.Collector)
	collectors["docker"] = &debug.Docker{}
	collectors["weave"] = &debug.Weave{}

	statusUpdates := make(chan debug.ServiceStatus)
	errChan := make(chan error)

	servicesStatuses := make(map[string]debug.ServiceStatus)
	go func() {
		for err := range errChan {
			log.Error(err)
		}
	}()
	updates := make(chan int)
	go func() {
		for serviceStatus := range statusUpdates {
			servicesStatuses[serviceStatus.Name] = serviceStatus
			updates <- 1
		}
	}()

	go func() {
		c := time.Tick(10 * time.Second)
		for _ = range c {
			for name, collector := range collectors {
				log.Debugf("Collecting data from %s", name)
				if err := collector.Collect(statusUpdates, errChan); err != nil {
					log.Error(err)
				}
			}
		}
	}()

	go serve(&servicesStatuses, updates)

	for name, collector := range collectors {
		log.Debugf("Collecting data from %s", name)
		checkError(collector.Init())
		if err := collector.Collect(statusUpdates, errChan); err != nil {
			log.Error(err)
		}
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	close(statusUpdates)
	close(errChan)
	for name, collector := range collectors {
		log.Debugf("Stopping %s collector", name)
		checkError(collector.Stop())
	}
}

func serve(servicesStatuses *map[string]debug.ServiceStatus, updates chan int) {
	box := packr.NewBox("./public")
	file, err := box.MustString("index.html")
	checkError(err)
	tmpl, err := template.New("home").Parse(file)
	checkError(err)

	type StatusPage struct {
		Period           int
		ServicesStatuses *map[string]debug.ServiceStatus
		Logs             []byte
	}

	log.Println("Dashboard available at http://localhost:4242")
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		log.Debug("REQUEST", req.URL)

		if req.URL.Path == "/favicon.ico" {
			return
		}

		statusPage := &StatusPage{
			ServicesStatuses: servicesStatuses,
			Period:           5,
		}
		serviceName := path.Base(req.URL.Path)
		if serviceName != "/" {
			_, ok := (*servicesStatuses)[serviceName]
			if ok {
				statusPage.Logs = (*servicesStatuses)[serviceName].Logs
			} else {
				log.Errorf("Unknow service %s", serviceName)
			}
		} else {
			for _, serviceStatus := range *servicesStatuses {
				statusPage.Logs = serviceStatus.Logs
				break
			}
		}

		var out bytes.Buffer
		err = tmpl.Execute(&out, statusPage)
		if err != nil {
			log.Error(err)
		}
		w.Write(out.Bytes())
	})

	http.HandleFunc("/services", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		b, err := json.Marshal(serviceStatusMapToSlice(*servicesStatuses))
		if err != nil {
			fmt.Println("error:", err)
		}
		w.Write(b)
	})

	var upgrader = websocket.Upgrader{
		CheckOrigin:       func(r *http.Request) bool { return true },
		EnableCompression: true,
	}
	http.HandleFunc("/ws", func(w http.ResponseWriter, req *http.Request) {
		c, err := upgrader.Upgrade(w, req, nil)
		if err != nil {
			log.Print("upgrade:", err)
			log.Error(err)
			return
		}
		defer c.Close()
		for _ = range updates {
			b, err := json.Marshal(serviceStatusMapToSlice(*servicesStatuses))
			if err != nil {
				log.Error(err)
			}
			err = c.WriteMessage(websocket.TextMessage, b)
			if err != nil {
				log.Error(err)
				break
			}
		}
	})
	http.ListenAndServe(":4242", nil)
}

func checkError(err error) {
	if err != nil {
		log.Error(err)
		os.Exit(-1)
	}
}

func serviceStatusMapToSlice(in map[string]debug.ServiceStatus) []debug.ServiceStatus {
	ss := make([]debug.ServiceStatus, 0, len(in))
	for _, value := range in {
		ss = append(ss, value)
	}
	return ss
}
