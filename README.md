# Docker Compose/Swarm Haproxy Ingress Controller

The aim of the project is it to create dynamic ingress rules for swarm services through labels. This allows to create new services and change the haproxy configuration without any downtime or container rebuild.

## Synopsis

```yml
services:
  ingress:
    image: bluebrown/moby-ingress:standalone
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:rw
    dns:
      - 127.0.0.11:53
    ports:
      - published: 8765
        target: 8765
        protocol: tcp
        mode: host
      - published: 8080
        target: 8080
        protocol: tcp
        mode: host
    labels:
      ingress.class: haproxy
      ingress.global: |
        spread-checks 15
      ingress.defaults: |
        timeout connect 5s
        timeout check 5s
        timeout client 2m
        timeout server 2m
        retries 1
        retry-on all-retryable-errors
        option redispatch 1
        default-server check inter 30s
      ingress.frontend.default: |
        bind *:8080
        option forwardfor except 127.0.0.1
        option forwardfor header X-Real-IP
        http-request disable-l7-retry unless METH_GET

  app:
    image: traefik/whoami
    deploy:
      replicas: 2
      labels:
        ingress.class: haproxy
        ingress.port: "80"
        ingress.frontend.default: |
          default_backend {{ .Name }}
        ingress.backend: |
          balance leastconn
          option httpchk GET /
          acl allowed_method method HEAD GET POST
          http-request deny unless allowed_method
```

See the [official haproxy documentation](https://www.haproxy.com/blog/the-four-essential-sections-of-an-haproxy-configuration/) to learn more about haproxy configuration. The settings are identical to the official haproxy version.

## Standalone

In standalone mode, the ingress service performs service discovery and updates the config if required, like in distributed mode. Additionally it manages a haproxy process to perform the actual loadbalancer. The limitation of this approach is that global, default and the initial frontend configurations can only be changed upon a restart of the ingress container because docker labels cannot be changed on a running container. This can lead to disruptions in the traffic.

```yaml
services:
  ingress:
    image: bluebrown/moby-ingress:standalone
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:rw
    dns:
      - 127.0.0.11:53
    ports:
      - published: 8765
        target: 8765
        protocol: tcp
        mode: host
      - published: 8080
        target: 8080
        protocol: tcp
        mode: host
```

## Distributed

In the distributed  setup. The work is split into to services. The config server is responsible for generating a valid haproxy configuration file from the labels. The loadbalancer instances scrapes the configuration periodically and reload the worker "hitless" if the content has changed. This allows to loadbalancer instance to run without any interruption even though the configserver may restart for various reasons. The setup comes with more overhead, but offers a more stable and flexible approach.

```yaml
services:
  configserver:
    image: bluebrown/moby-ingress:configserver
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:rw
    dns:
      - 127.0.0.11:53
  loadbalancer:
    image: bluebrown/moby-ingress:loadbalancer
    command: --configserver http://configserver:6789
    ports:
      - published: 8765
        target: 8765
        protocol: tcp
        mode: host
      - published: 8080
        target: 8080
        protocol: tcp
        mode: host
```

## Template

The labels are parsed and passed into a haproxy.cfg template. The default template looks like the below.

```go
resolvers docker
    nameserver dns1 127.0.0.11:53
    resolve_retries 3
    timeout resolve 1s
    timeout retry   1s
    hold other      10s
    hold refused    10s
    hold nx         10s
    hold timeout    10s
    hold valid      10s
    hold obsolete   10s

global
    log          fd@2 local2
    stats timeout 2m
    {{ .Global | indent 4 | trim }}

defaults
    log global
    mode http
    option httplog
    {{ .Defaults | indent 4 | trim }}

listen stats
    bind *:8765
    stats enable
    stats uri /
    stats refresh 15s
    stats show-legends
    stats show-node {{ println "" }}

{{- range $frontend, $config := .Frontend }}
frontend {{$frontend}}
    {{ $config | nindent 4 | trim }}
{{ end }}

{{- range $backend, $config := .Backend }}
backend {{$backend}}
    {{ $config.Backend | nindent 4 | trim }}
    server-template {{ $backend }}- {{ $config.Replicas }} {{ if ne .EndpointMode "dnsrr" }}tasks.{{ end }}{{ $backend }}:{{ default "80" $config.Port }} resolvers docker init-addr libc,none
{{ end }}
```

The data types passed into the template have the following format. The ingressClass is used to filter services. That way it is possible to run the separate stacks with separate controllers and a different ingress class if required.

Frontend snippets in the backend struct are executed as template and merged with the frontend config from the ConfigData struct. That is why it is not returned as json and not directly used in the template. The endpoint mode is used to determine what dns pattern should be used to query the docker dns resolver for the service.

```go
type ConfigData struct {
 IngressClass string              `json:"-" mapstructure:"class"`
 Global       string              `json:"global,omitempty"`
 Defaults     string              `json:"defaults,omitempty"`
 Frontend     map[string]string   `json:"frontend,omitempty"`
 Backend      map[string]*Backend `json:"backend,omitempty"`
}

type Backend struct {
 EndpointMode string            `json:"endpoint_mode,omitempty"`
 Port         string            `json:"port,omitempty"`
 Replicas     uint64            `json:"replicas,omitempty"`
 Frontend     map[string]string `json:"-"`
 Backend      string            `json:"backend,omitempty"`
}
```

### Update the template at runtime

It is possible to change the template at runtime via PUT request. The request can be made against the standalone or configserver version.

```shell
curl -i -X PUT localhost:6789 --data-binary @path/to/template
```

## Config Server API

The config server API, is only available when using the config server. The standalone version does not include it as there is not need for this endpoint.

The configuration can be fetched via the root endpoint of the configserver service. The server will return the current config immediately if the `Config-Hash` header has not been set or its value is different from the the current configs hash.

The content hash is a md5sum of the current config. It is used to communicate to the server if the client has the current config already. If send hash matches the current config, the manager will respond in a long-polling fashion. That means, it will leave the connection open and not respond at all until the haproxy config file in memory hash changed and a new hash has been computed. This mechanism is meant to avoid sending data across the network that the client already has.

```shell
# immediate response
curl -i localhost:8080
# immediate response if hash does not match, otherwise long polling
curl -H 'Content-Hash: <md5sum-of-local-config>' localhost:8080
```

The raw data to populate the template is also available in json format. Since the content hash is the hash of the rendered haproxy config, it is not really meant to be used for the json response. If you still want to use it, you need to read the hash from the response header `Content-Hash` as computing the md5sum of the json content will be always different.

```shell
curl -i -H 'Accept: application/json' localhost:8080
```

Example HTTP requests can be found in the assets folder.

### Example Config Response

```c
resolvers docker
    nameserver dns1 127.0.0.11:53
    resolve_retries 3
    timeout resolve 1s
    timeout retry   1s
    hold other      10s
    hold refused    10s
    hold nx         10s
    hold timeout    10s
    hold valid      10s
    hold obsolete   10s

global
    log          fd@2 local2
    stats timeout 2m
    spread-checks 15

defaults
    log global
    mode http
    option httplog
    timeout connect 5s
    timeout check 5s
    timeout client 2m
    timeout server 2m
    retries 1
    retry-on all-retryable-errors
    option redispatch 1
    default-server check inter 30s

listen stats
    bind *:8765
    stats enable
    stats uri /
    stats refresh 15s
    stats show-legends
    stats show-node

frontend default
    bind *:80
    option forwardfor except 127.0.0.1
    option forwardfor header X-Real-IP
    http-request disable-l7-retry unless METH_GET
    default_backend app

backend app
    balance leastconn
    option httpchk GET /
    acl allowed_method method HEAD GET POST
    http-request deny unless allowed_method
    server-template app- 2 app:80 resolvers docker init-addr libc,none
```

## Haproxy Socket

If you publish the port 9876 on the loadbalancer or standalone version you can use `socat` to connect to the socket cli.

```bash
$ socat tcp-connect:127.0.0.1:9876 -
$ prompt
$ master> help
 help
The following commands are valid at this level:
  @!<pid>                                 : send a command to the <pid> process
  @<relative pid>                         : send a command to the <relative pid> process
  @master                                 : send a command to the master process
  operator                                : lower the level of the current CLI session to operator
  reload                                  : reload haproxy
  show cli level                          : display the level of the current CLI session
  show cli sockets                        : dump list of cli sockets
  show proc                               : show processes status
  show version                            : show version of the current process
  user                                    : lower the level of the current CLI session to user
  help [<command>]                        : list matching or all commands
  prompt                                  : toggle interactive mode with prompt
  quit                                    : disconnect
```
