# sg-agent

## Set of sg-core application/transport plugins for spawning availability monitoring agent STF

sg-agent serves as availability monitoring agent for STF. It executes
self-scheduled and/or sg-core requested checks and reports results back.

As a matter of fact sg-agent will not need sg-core as a server side component.
The plan for sg-agent is to be self sufficient without server side or cooperate
well with other availability monitoring solutions such as sensu-core, sensu-go,
Nagios, Icinga, Zabbix or even Datadog.

TO-DO:

  - [ ] basic functionality
  - [ ] sensu-core mediator (via RabbitMQ)
  - [ ] sensu-go mediator (via websockets)
  - [ ] DataDog mediator (via DogStatsD)
  - [ ] Nagios mediator (via special Nagios plugin)
  - [ ] Icinga mediator (via Icinga API)
  - [ ] Zabbix mediator (via Zabbix API)
