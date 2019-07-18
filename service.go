package main

import (
	"context"

	testcontainers "github.com/testcontainers/testcontainers-go"
)

// Service represents the contract for services
type Service interface {
	Destroy() error
	ExposePorts() []string
	Run() (testcontainers.Container, error)
}

// DockerService represents a Docker service to be run
type DockerService struct {
	// Daemon indicates if the service must be run as a daemon
	Daemon         bool
	ExposedPorts   []ExposedPort
	ImageTag       string
	RunningService testcontainers.Container
}

// ExposePorts returns an array of exposed ports
func (s *DockerService) ExposePorts() []string {
	ports := []string{}

	for _, p := range s.ExposedPorts {
		ports = append(ports, p.toString())
	}

	return ports
}

// ExposedPort represents the structure for how services expose ports
type ExposedPort struct {
	Address       string
	ContainerPort string
	HostPort      string
	Protocol      string
}

func (e *ExposedPort) toString() string {
	return e.Address + ":" + e.HostPort + ":" + e.ContainerPort + "/" + e.Protocol
}

// Destroy destroys the underlying container
func (s *DockerService) Destroy() error {
	ctx := context.Background()

	s.RunningService.Terminate(ctx)

	return nil
}

// Run runs a container for the service
func (s *DockerService) Run() (testcontainers.Container, error) {
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        s.ImageTag,
		ExposedPorts: s.ExposePorts(),
	}

	service, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	s.RunningService = service

	return service, nil
}

// AsDaemon marks this service to be run as daemon
func (s *DockerService) AsDaemon() *DockerService {
	s.Daemon = true

	return s
}
