// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package compose

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/elastic/e2e-testing/cli/config"
	state "github.com/elastic/e2e-testing/internal/state"
	"go.elastic.co/apm"

	log "github.com/sirupsen/logrus"
	tc "github.com/testcontainers/testcontainers-go"
)

// ServiceManager manages lifecycle of a service
type ServiceManager interface {
	AddServicesToCompose(ctx context.Context, profile string, composeNames []string, env map[string]string, composeFilename ...string) error
	ExecCommandInService(profile string, image string, serviceName string, cmds []string, env map[string]string, detach bool) error
	RemoveServicesFromCompose(ctx context.Context, profile string, composeNames []string, env map[string]string) error
	RunCommand(profile string, composeNames []string, composeArgs []string, env map[string]string) error
	RunCompose(ctx context.Context, isProfile bool, composeNames []string, env map[string]string) error
	StopCompose(ctx context.Context, isProfile bool, composeNames []string) error
}

// DockerServiceManager implementation of the service manager interface
type DockerServiceManager struct {
}

// NewServiceManager returns a new service manager
func NewServiceManager() ServiceManager {
	return &DockerServiceManager{}
}

// AddServicesToCompose adds services to a running docker compose
func (sm *DockerServiceManager) AddServicesToCompose(ctx context.Context, profile string, composeNames []string, env map[string]string, composeFilename ...string) error {
	span, _ := apm.StartSpanOptions(ctx, "Add services to Docker Compose", "docker-compose.services.add", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	log.WithFields(log.Fields{
		"profile":  profile,
		"services": composeNames,
	}).Trace("Adding services to compose")

	newComposeNames := []string{profile}
	newComposeNames = append(newComposeNames, composeNames...)

	persistedEnv := state.Recover(profile+"-profile", config.Op.Workspace)
	for k, v := range env {
		persistedEnv[k] = v
	}

	err := executeCompose(true, newComposeNames, []string{"up", "-d"}, persistedEnv, composeFilename...)
	if err != nil {
		return err
	}

	return nil
}

// ExecCommandInService executes a command in a service from a profile
func (sm *DockerServiceManager) ExecCommandInService(profile string, image string, serviceName string, cmds []string, env map[string]string, detach bool) error {
	composes := []string{
		profile, // profile name
		image,   // image for the service
	}
	composeArgs := []string{"exec", "-T"}
	if detach {
		composeArgs = append(composeArgs, "-d")
	}
	composeArgs = append(composeArgs, serviceName)
	composeArgs = append(composeArgs, cmds...)

	err := sm.RunCommand(profile, composes, composeArgs, env)
	if err != nil {
		log.WithFields(log.Fields{
			"command": cmds,
			"error":   err,
			"service": serviceName,
		}).Error("Could not execute command in service container")

		return err
	}

	return nil
}

// RemoveServicesFromCompose removes services from a running docker compose
func (sm *DockerServiceManager) RemoveServicesFromCompose(ctx context.Context, profile string, composeNames []string, env map[string]string) error {
	span, _ := apm.StartSpanOptions(ctx, "Remove services from Docker Compose", "docker-compose.services.remove", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	log.WithFields(log.Fields{
		"profile":  profile,
		"services": composeNames,
	}).Trace("Removing services from compose")

	newComposeNames := []string{profile}
	newComposeNames = append(newComposeNames, composeNames...)

	persistedEnv := state.Recover(profile+"-profile", config.Op.Workspace)
	for k, v := range env {
		persistedEnv[k] = v
	}

	for _, composeName := range composeNames {
		command := []string{"rm", "-fvs"}
		command = append(command, composeName)

		err := executeCompose(true, newComposeNames, command, persistedEnv)
		if err != nil {
			log.WithFields(log.Fields{
				"command": command,
				"service": composeName,
				"profile": profile,
			}).Error("Could not remove service from compose")
			return err
		}
		log.WithFields(log.Fields{
			"profile": profile,
			"service": composeName,
		}).Debug("Service removed from compose")
	}

	return nil
}

// RunCommand executes a docker-compose command in a running a docker compose
func (sm *DockerServiceManager) RunCommand(profile string, composeNames []string, composeArgs []string, env map[string]string) error {
	return executeCompose(true, composeNames, composeArgs, env)
}

// RunCompose runs a docker compose by its name
func (sm *DockerServiceManager) RunCompose(ctx context.Context, isProfile bool, composeNames []string, env map[string]string) error {
	span, _ := apm.StartSpanOptions(ctx, "Starting Docker Compose files", "docker-compose.services.up", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	return executeCompose(isProfile, composeNames, []string{"up", "-d"}, env)
}

// StopCompose stops a docker compose by its name
func (sm *DockerServiceManager) StopCompose(ctx context.Context, isProfile bool, composeNames []string) error {
	span, _ := apm.StartSpanOptions(ctx, "Stopping Docker Compose files", "docker-compose.services.down", apm.SpanOptions{
		Parent: apm.SpanFromContext(ctx).TraceContext(),
	})
	defer span.End()

	composeFilePaths := make([]string, len(composeNames))
	for i, composeName := range composeNames {
		b := isProfile
		if i == 0 && !isProfile && (len(composeName) == 1) {
			b = true
		}

		composeFilePath, err := config.GetComposeFile(b, composeName)
		if err != nil {
			return fmt.Errorf("Could not get compose file: %s - %v", composeFilePath, err)
		}
		composeFilePaths[i] = composeFilePath
	}

	ID := composeNames[0] + "-service"
	if isProfile {
		ID = composeNames[0] + "-profile"
	}
	persistedEnv := state.Recover(ID, config.Op.Workspace)

	err := executeCompose(isProfile, composeNames, []string{"down", "--remove-orphans"}, persistedEnv)
	if err != nil {
		return fmt.Errorf("Could not stop compose file: %v - %v", composeFilePaths, err)
	}
	defer state.Destroy(ID, config.Op.Workspace)

	log.WithFields(log.Fields{
		"composeFilePath": composeFilePaths,
		"profile":         composeNames[0],
	}).Trace("Docker compose down.")

	return nil
}

func executeCompose(isProfile bool, composeNames []string, command []string, env map[string]string, composeFilename ...string) error {
	composeFilePaths := make([]string, len(composeNames))
	for i, composeName := range composeNames {
		b := false
		if i == 0 && isProfile {
			b = true
		}

		composeFilePath, err := config.GetComposeFile(b, composeName, composeFilename...)
		if err != nil {
			return fmt.Errorf("Could not get compose file: %s - %v", composeFilePath, err)
		}
		composeFilePaths[i] = composeFilePath
	}

	compose := tc.NewLocalDockerCompose(composeFilePaths, composeNames[0])
	execError := compose.
		WithCommand(command).
		WithEnv(env).
		Invoke()
	err := execError.Error
	if err != nil {
		return fmt.Errorf("Could not run compose file: %v - %v", composeFilePaths, err)
	}

	suffix := "-service"
	if isProfile {
		suffix = "-profile"
	}
	ID := filepath.Base(filepath.Dir(composeFilePaths[0])) + suffix
	defer state.Update(ID, config.Op.Workspace, composeFilePaths, env)

	log.WithFields(log.Fields{
		"cmd":              command,
		"composeFilePaths": composeFilePaths,
		"env":              env,
		"profile":          composeNames[0],
	}).Debug("Docker compose executed.")

	return nil
}