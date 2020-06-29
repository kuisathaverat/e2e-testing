@fleet_mode
Feature: Fleet Mode Agent
  Scenarios for the Agent in Fleet mode connecting to Ingest Manager application.

@enroll
Scenario: Deploying an agent
  Given the "Fleet" Kibana setup has been executed
  When an agent is deployed to Fleet
  Then the agent is listed in Fleet as online
    And system package dashboards are listed in Fleet
    And there is data in the index

@start-agent
Scenario: Starting the agent starts backend processes
  When an agent is deployed to Fleet
  Then the "filebeat" process is in the "started" state on the host
    And the "metricbeat" process is in the "started" state on the host

@stop-agent
Scenario: Stopping the agent stops backend processes
  Given an agent is deployed to Fleet
  When the "agent" process is "stopped" on the host
  Then the "filebeat" process is in the "stopped" state on the host
    And the "metricbeat" process is in the "stopped" state on the host

@unenroll
Scenario: Un-enrolling an agent
  Given an agent is deployed to Fleet
  When the agent is un-enrolled
  Then the agent is not listed as online in Fleet
    And there is no data in the index

@reenroll
Scenario: Re-enrolling an agent
  Given an agent is deployed to Fleet
    And the agent is un-enrolled
    And the "agent" process is "stopped" on the host
  When the agent is re-enrolled on the host
    And the "agent" process is "started" on the host
  Then the agent is listed in Fleet as online
    And there is data in the index

@revoke-token
Scenario: Revoking the enrollment token for an agent
  Given an agent is deployed to Fleet
  When the enrollment token is revoked
  Then an attempt to enroll a new agent fails
