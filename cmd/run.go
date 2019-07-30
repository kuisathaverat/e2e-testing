package cmd

import (
	"errors"
	"fmt"

	"github.com/elastic/metricbeat-tests-poc/services"

	"github.com/spf13/cobra"
)

var versionToRun string

func init() {
	rootCmd.AddCommand(runCmd)

	initialServices := []string{
		"apache", "kafka", "mysql",
	}

	for _, s := range initialServices {
		runSubcommand := buildRunServiceCommand(s)

		runSubcommand.Flags().StringVarP(&versionToRun, "version", "v", "", "Sets the image version to run")

		runCmd.AddCommand(runSubcommand)
	}

	runCmd.AddCommand(runStackCmd)
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Runs a Service to be monitored",
	Long: `Runs a Service to be monitored by Metricbeat, spinning up a Docker container for it and exposing its internal.
	configuration so that you are able to connect to it in an easy manner`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return errors.New("run requires zero or one argument representing the image tag to be run")
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Hello run!")
	},
}

func buildRunServiceCommand(service string) *cobra.Command {
	return &cobra.Command{
		Use:   service,
		Short: `Runs a ` + service + ` service`,
		Long: `Runs a ` + service + ` service to be monitored by Metricbeat, spinning up a Docker container for it and exposing its internal
		configuration so that you are able to connect to it in an easy manner`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return errors.New("run requires zero or one argument representing the image tag to be run")
			}

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			s := buildService(service, versionToRun)

			serviceManager := services.NewServiceManager()

			serviceManager.Run(s)
		},
	}
}

func buildService(service string, version string) services.Service {
	if service == "apache" {
		return services.NewApacheService(version, true)
	} else if service == "kafka" {
		return services.NewKafkaService(version, true)
	} else if service == "mysql" {
		return services.NewMySQLService(version, true)
	}

	return nil
}

var runStackCmd = &cobra.Command{
	Use:   "stack",
	Short: "Runs an Elastic Stack (Elasticsearch + Kibana)",
	Long: `Runs an Elastic Stack (Elasticsearch + Kibana), spinning up Docker containers for them and exposing their internal
	configuration so that you are able to connect to them in an easy manner`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return errors.New("run requires zero or one argument representing the image tag to be run")
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		serviceManager := services.NewServiceManager()

		es := services.NewElasticsearchService(versionToRun, true)
		serviceManager.Run(es)

		s := services.NewKibanaService(versionToRun, true, es)
		serviceManager.Run(s)
	},
}
