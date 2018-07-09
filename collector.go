package debug

import (
	"context"
	"io"
	"path"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// ServiceStatus represents a services status
type ServiceStatus struct {
	Name   string    `json:"name"`
	State  string    `json:"state"`
	Health string    `json:"health"`
	Logs   io.Reader `json:"-"`
}

// String return a string representation of the service's current state
func (s ServiceStatus) String() string {
	healthyStates := "created running"
	unhealthyStates := "paused restarting removing exited dead"

	if strings.Contains(healthyStates, s.State) && s.Health == types.Healthy {
		return "healthy"
	}
	if strings.Contains(healthyStates, s.State) && s.Health == types.Unhealthy {
		return "warn"
	}
	if strings.Contains(unhealthyStates, s.State) {
		return "error"
	}
	return "warn"
}

// Collector collects all kind of debug data for services
type Collector interface {
	Init() error
	Collect(chan ServiceStatus, chan error) error
	Stop() error
}

// Docker is a Docker container status collector
type Docker struct {
	dockerClient *client.Client
}

// Init creates
func (d *Docker) Init() error {
	const dockerClientVersion = "1.35"

	dockerClient, err := client.NewClientWithOpts(client.WithVersion(dockerClientVersion))
	if err != nil {
		return err
	}
	d.dockerClient = dockerClient
	return nil
}

// Collect gathers status data on docker containers
func (d *Docker) Collect(out chan ServiceStatus, errChan chan error) error {
	const dockerComposeServiceNameLabel = "com.docker.compose.service"

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	containers, err := d.dockerClient.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return err
	}

	for _, container := range containers {
		go func(container types.Container, out chan ServiceStatus) {
			serviceStatus := ServiceStatus{
				Name:  container.Names[0],
				State: container.State,
			}
			if container.Labels[dockerComposeServiceNameLabel] != "" {
				serviceStatus.Name = container.Labels[dockerComposeServiceNameLabel]
			}
			serviceStatus.Name = path.Base(serviceStatus.Name)
			log.Debugf("[Docker] Collecting %s status", serviceStatus.Name)
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			c, err := d.dockerClient.ContainerInspect(ctx, container.ID)
			if err != nil {
				errChan <- errors.Wrapf(err, "Error inspecting %s container", serviceStatus.Name)
				return
			}
			if c.State.Health != nil {
				serviceStatus.Health = c.State.Health.Status
			}

			ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if ServiceStatus.Logs == nil {
				logsReader, err := d.dockerClient.ContainerLogs(ctx, container.ID, types.ContainerLogsOptions{
					ShowStdout: true,
					ShowStderr: true,
					Follow:     true,
				})
				if err != nil {
					errChan <- errors.Wrap(err, "Error collecting docker container logs")
					return
				}
				serviceStatus.Logs = logsReader
			}
			out <- serviceStatus
		}(container, out)
	}
	return nil
}

// Stop closes the docker client connection
func (d *Docker) Stop() error {
	return d.dockerClient.Close()
}

type Weave struct{}

func (w *Weave) Init() error {
	return nil
}

func (w *Weave) Collect(out chan ServiceStatus, errChan chan error) error {
	return nil
}

func (w *Weave) Stop() error {
	return nil
}
