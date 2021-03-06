package main

import (
	"fmt"
	"strings"

	"github.com/Jeffail/gabs/v2"
	log "github.com/sirupsen/logrus"
)

// title for the Elastic Endpoint integration in the package registry.
// This value could change depending on the version of the package registry
// We are using the title because the feature files have to be super readable
// and the title is more readable than the name
const elasticEnpointIntegrationTitle = "Endpoint Security"

// IntegrationPackage used to share information about a integration
type IntegrationPackage struct {
	packageConfigID string          `json:"packageConfigId"`
	name            string          `json:"name"`
	title           string          `json:"title"`
	version         string          `json:"version"`
	json            *gabs.Container // json representation of the integration
}

// addIntegrationToPolicy sends a POST request to Fleet adding an integration to a configuration
func addIntegrationToPolicy(integrationPackage IntegrationPackage, policyID string) (string, error) {
	name := integrationPackage.name + "-test-name"
	description := integrationPackage.title + "-test-description"

	body, err := kibanaClient.AddIntegrationToPolicy(integrationPackage.name, name, integrationPackage.title, description, integrationPackage.version, policyID)
	if err != nil {
		return "", err
	}

	jsonParsed, err := gabs.ParseJSON([]byte(body))
	if err != nil {
		log.WithFields(log.Fields{
			"error":        err,
			"responseBody": body,
		}).Error("Could not parse response into JSON")
		return "", err
	}

	integrationConfigurationID := jsonParsed.Path("item.id").Data().(string)

	log.WithFields(log.Fields{
		"policyID":                   policyID,
		"integrationConfigurationID": integrationConfigurationID,
		"integration":                integrationPackage.name,
		"version":                    integrationPackage.version,
	}).Info("Integration added to the configuration")

	return integrationConfigurationID, nil
}

// deleteIntegrationFromPolicy sends a POST request to Fleet deleting an integration from a configuration
func deleteIntegrationFromPolicy(integrationPackage IntegrationPackage, policyID string) error {
	_, err := kibanaClient.DeleteIntegrationFromPolicy(integrationPackage.packageConfigID)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"policyID":        policyID,
		"integration":     integrationPackage.name,
		"packageConfigId": integrationPackage.packageConfigID,
		"version":         integrationPackage.version,
	}).Info("Integration deleted from the configuration")

	return nil
}

// getIntegration returns metadata from an integration from Fleet, without the package ID
func getIntegration(packageName string, version string) (IntegrationPackage, error) {
	body, err := kibanaClient.GetIntegration(packageName, version)
	if err != nil {
		return IntegrationPackage{}, err
	}

	jsonParsed, err := gabs.ParseJSON([]byte(body))
	if err != nil {
		log.WithFields(log.Fields{
			"error":        err,
			"responseBody": body,
		}).Error("Could not parse get response into JSON")
		return IntegrationPackage{}, err
	}

	response := jsonParsed.Path("response")
	integrationPackage := IntegrationPackage{
		name:    response.Path("name").Data().(string),
		title:   response.Path("title").Data().(string),
		version: response.Path("latestVersion").Data().(string),
	}

	return integrationPackage, nil
}

// getIntegrationFromAgentPolicy inspects the integrations added to an agent policy, returning the
// a struct representing the package, including the packageID for the integration in the policy
func getIntegrationFromAgentPolicy(packageName string, agentPolicyID string) (IntegrationPackage, error) {
	body, err := kibanaClient.GetIntegrationFromAgentPolicy(agentPolicyID)
	if err != nil {
		return IntegrationPackage{}, err
	}

	jsonParsed, err := gabs.ParseJSON([]byte(body))
	if err != nil {
		log.WithFields(log.Fields{
			"error":        err,
			"responseBody": body,
		}).Error("Could not parse response into JSON")
		return IntegrationPackage{}, err
	}

	packagePolicies := jsonParsed.Path("item.package_policies").Children()
	for _, packagePolicy := range packagePolicies {
		title := packagePolicy.Path("package.title").Data().(string)
		if title == packageName {
			integrationPackage := IntegrationPackage{
				packageConfigID: packagePolicy.Path("id").Data().(string),
				name:            packagePolicy.Path("package.name").Data().(string),
				title:           title,
				version:         packagePolicy.Path("package.version").Data().(string),
				json:            packagePolicy,
			}

			log.WithFields(log.Fields{
				"package":  integrationPackage,
				"policyID": agentPolicyID,
			}).Debug("Package policy found in the configuration")

			return integrationPackage, nil
		}
	}

	return IntegrationPackage{}, fmt.Errorf("%s package policy not found in the configuration", packageName)
}

// getIntegrationLatestVersion sends a GET request to Fleet for the existing integrations
// checking if the desired integration exists in the package registry. If so, it will
// return name and version (latest) of the integration
func getIntegrationLatestVersion(integrationName string) (string, string, error) {
	body, err := kibanaClient.GetIntegrations()
	if err != nil {
		return "", "", err
	}

	jsonParsed, err := gabs.ParseJSON([]byte(body))
	if err != nil {
		log.WithFields(log.Fields{
			"error":        err,
			"responseBody": body,
		}).Error("Could not parse response into JSON")
		return "", "", err
	}

	// data streams should contain array of elements
	integrations := jsonParsed.Path("response").Children()

	log.WithFields(log.Fields{
		"count": len(integrations),
	}).Trace("Integrations retrieved")

	for _, integration := range integrations {
		title := integration.Path("title").Data().(string)
		if strings.ToLower(title) == strings.ToLower(integrationName) {
			name := integration.Path("name").Data().(string)
			version := integration.Path("version").Data().(string)
			log.WithFields(log.Fields{
				"name":    name,
				"title":   title,
				"version": version,
			}).Debug("Integration in latest version found")
			return name, version, nil
		}
	}

	return "", "", fmt.Errorf("The %s integration was not found", integrationName)
}

// getMetadataFromSecurityApp sends a POST request to Endpoint retrieving the metadata that
// is listed in the Security App
func getMetadataFromSecurityApp() (*gabs.Container, error) {
	body, err := kibanaClient.GetMetadataFromSecurityApp()
	if err != nil {
		return nil, err
	}

	jsonParsed, err := gabs.ParseJSON([]byte(body))
	if err != nil {
		log.WithFields(log.Fields{
			"error":        err,
			"responseBody": body,
		}).Error("Could not parse response into JSON")
		return nil, err
	}

	hosts := jsonParsed.Path("hosts")

	log.WithFields(log.Fields{
		"hosts": hosts,
	}).Trace("Hosts in the Security App")

	return hosts, nil
}

// installIntegration sends a POST request to Fleet installing the assets for an integration
func installIntegrationAssets(integration string, version string) (IntegrationPackage, error) {
	body, err := kibanaClient.InstallIntegrationAssets(integration, version)
	if err != nil {
		return IntegrationPackage{}, err
	}

	log.WithFields(log.Fields{
		"integration": integration,
		"version":     version,
	}).Info("Assets for the integration where installed")

	jsonParsed, err := gabs.ParseJSON([]byte(body))
	if err != nil {
		log.WithFields(log.Fields{
			"error":        err,
			"responseBody": body,
		}).Error("Could not parse install response into JSON")
		return IntegrationPackage{}, err
	}
	response := jsonParsed.Path("response").Index(0)

	packageConfigID := response.Path("id").Data().(string)

	// get the integration again in the case it's already installed
	integrationPackage, err := getIntegration(integration, version)
	if err != nil {
		return IntegrationPackage{}, err
	}

	integrationPackage.packageConfigID = packageConfigID

	return integrationPackage, nil
}

// isAgentListedInSecurityApp retrieves the hosts from Endpoint to check if a hostname
// is listed in the Security App. For that, we will inspect the metadata, and will iterate
// through the hosts, until we get the proper hostname.
func isAgentListedInSecurityApp(hostName string) (*gabs.Container, error) {
	hosts, err := getMetadataFromSecurityApp()
	if err != nil {
		return nil, err
	}

	for _, host := range hosts.Children() {
		metadataHostname := host.Path("metadata.host.hostname").Data().(string)
		if metadataHostname == hostName {
			log.WithFields(log.Fields{
				"hostname": hostName,
			}).Debug("Hostname for the agent listed in the Security App")

			return host, nil
		}
	}

	return nil, nil
}

// isAgentListedInSecurityAppWithStatus inspects the metadata field for a hostname, obtained from
// the security App. We will check if the status matches the desired status, returning an error
// if the agent is not present in the Security App
func isAgentListedInSecurityAppWithStatus(hostName string, desiredStatus string) (bool, error) {
	host, err := isAgentListedInSecurityApp(hostName)
	if err != nil {
		log.WithFields(log.Fields{
			"hostname": hostName,
			"error":    err,
		}).Error("There was an error getting the agent in the Administration view in the Security app")
		return false, err
	}

	if host == nil {
		return false, fmt.Errorf("The host %s is not listed in the Administration view in the Security App", hostName)
	}

	hostStatus := host.Path("host_status").Data().(string)
	log.WithFields(log.Fields{
		"desiredStatus": desiredStatus,
		"hostname":      hostName,
		"status":        hostStatus,
	}).Debug("Hostname for the agent listed with desired status in the Administration view in the Security App")

	return (hostStatus == desiredStatus), nil
}

// isPolicyResponseListedInSecurityApp sends a POST request to Endpoint to check if a hostname
// is listed in the Security App. For that, we will inspect the metadata, and will iterate
// through the hosts, until we get the policy status, finally checking for the success
// status.
func isPolicyResponseListedInSecurityApp(agentID string) (bool, error) {
	hosts, err := getMetadataFromSecurityApp()
	if err != nil {
		return false, err
	}

	for _, host := range hosts.Children() {
		metadataAgentID := host.Path("metadata.elastic.agent.id").Data().(string)
		name := host.Path("metadata.Endpoint.policy.applied.name").Data().(string)
		status := host.Path("metadata.Endpoint.policy.applied.status").Data().(string)
		if metadataAgentID == agentID {
			log.WithFields(log.Fields{
				"agentID": agentID,
				"name":    name,
				"status":  status,
			}).Debug("Policy response for the agent listed in the Security App")

			return (status == "success"), nil
		}
	}

	return false, nil
}

// updateIntegrationPackageConfig sends a PUT request to Fleet updating integration
// configuration
func updateIntegrationPackageConfig(packageConfigID string, payload string) (*gabs.Container, error) {
	body, err := kibanaClient.UpdateIntegrationPackageConfig(packageConfigID, payload)
	if err != nil {
		return nil, err
	}

	jsonParsed, err := gabs.ParseJSON([]byte(body))
	if err != nil {
		log.WithFields(log.Fields{
			"error":        err,
			"responseBody": body,
		}).Error("Could not parse response into JSON")
		return nil, err
	}

	log.WithFields(log.Fields{
		"policyID": packageConfigID,
	}).Debug("Configuration for the integration was updated")

	return jsonParsed, nil
}
