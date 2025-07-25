---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-module-http.html
---

% This file is generated! See scripts/docs_collector.py

# HTTP module [metricbeat-module-http]

The HTTP module is a Metricbeat module used to call arbitrary HTTP endpoints for which a dedicated Metricbeat module is not available.

Multiple endpoints can be configured which are polled in a regular interval and the result is shipped to the configured output channel. It is recommended to install a Metricbeat instance on each host from which data should be fetched.

This module is inspired by the Logstash [http_poller](logstash-docs-md://lsr/plugins-inputs-http_poller.md) input filter but doesn’t require that the endpoint is reachable by Logstash as the Metricbeat module pushes the data to the configured output channels, e.g. Logstash or Elasticsearch.

This is often necessary in security restricted network setups, where Logstash is not able to reach all servers. Instead the server to be monitored itself has Metricbeat installed and can send the data or a collector server has Metricbeat installed which is deployed in the secured network environment and can reach all servers to be monitored.

::::{note}
As the HTTP metricsets also fetch headers, this can lead to lots of fields in Elasticsearch in case there are many different headers. If this is the case for you and you don’t need the headers, we recommend to use processors to filter out the header field.
::::


## Example configuration [_example_configuration]

The HTTP module supports the standard configuration options that are described in [Modules](/reference/metricbeat/configuration-metricbeat.md). Here is an example configuration:

```yaml
metricbeat.modules:
- module: http
  #metricsets:
  #  - json
  period: 10s
  hosts: ["localhost:80"]
  namespace: "json_namespace"
  path: "/"
  #body: ""
  #method: "GET"
  #username: "user"
  #password: "secret"
  #request.enabled: false
  #response.enabled: false
  #json.is_array: false
  #dedot.enabled: false

- module: http
  #metricsets:
  #  - server
  host: "localhost"
  port: "8080"
  enabled: false
  #paths:
  #  - path: "/foo"
  #    namespace: "foo"
  #    fields: # added to the the response in root. overwrites existing fields
  #      key: "value"
```

This module supports TLS connections when using `ssl` config field, as described in [SSL](/reference/metricbeat/configuration-ssl.md). It also supports the options described in [Standard HTTP config options](/reference/metricbeat/configuration-metricbeat.md#module-http-config-options).


## Metricsets [_metricsets]

The following metricsets are available:

* [json](/reference/metricbeat/metricbeat-metricset-http-json.md)
* [server](/reference/metricbeat/metricbeat-metricset-http-server.md)
