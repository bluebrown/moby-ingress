# Docker Swarm Haproxy Ingress Controller

The aim of the project is it to create dynamic ingress rules for swarm services through labels. This allows to create new services and change the haproxy configuration without any downtime or container rebuild.

The manager service is responsible for generating a valid haproxy configuration file from the labels. The loadbalancer instances scrape the configuration periodically and reload the worker "hitless" if the content has changed.

## Synopsis

```yml
version: "3.9"

services:
  loadbalancer:
    image: swarm-haproxy-loadbalancer
    environment:
      MANAGER_ENDPOINT: http://manager:8080/
      SCRAPE_INTERVAL: "25"
      STARTUP_DELAY: "5"
    ports:
      - 3000:80 # ingress port
      - 4450:4450 # stats page
      - 9876:9876 # socket cli

  manager:
    image: swarm-haproxy-manager
    # this is the default template path. the flag is only set here
    # to show how to override the default template path
    command: --template templates/haproxy.cfg.template
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:rw
    ports:
      - 8080:8080
    deploy:
      mode: global
      placement:
        constraints:
          # needs to be on a manager node to read the services
          - "node.role==manager"
    # manager labels are added to the container
    # instead of the services under the deploy key
    labels:
      # if the ingress class is provided the services are
      # filtered by the ingress class otherwise all services are checked
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
      ingress.frontend.default: |
        bind *:80
        option forwardfor except 127.0.0.1
        option forwardfor header X-Real-IP
        http-request disable-l7-retry unless METH_GET

  app:
    image: nginx
    deploy:
      replicas: 2
      # service labels are added under the deploy key
      labels:
        # the ingress class of the manager
        ingress.class: haproxy
        # the application port inside the container (default: 80)
        ingress.port: "80"
        # rules are merged with corresponding frontend
        # the service name is used available in go template format
        ingress.frontend.default: |
          default_backend {{ .Name }}
        # backend snippet are added to the backend created from
        # this service definition
        ingress.backend: |
          balance leastconn
          option httpchk GET /
          acl allowed_method method HEAD GET POST
          http-request deny unless allowed_method
```

See the [official haproxy documentation](https://www.haproxy.com/blog/the-four-essential-sections-of-an-haproxy-configuration/) to learn more about haproxy configuration. The settings are identical to the official haproxy version.

Currently it only works when deploying the *backend* services with swarm. The manager can be deployed with a normal container. This is because the labels for the manager are provided on container level while the backends are created from service definitions and their labels.

> Note
> *Be careful which ports you publish in production*

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
    stats socket /var/run/haproxy.pid mode 600 expose-fd listeners level user
    stats timeout 2m
    {{ .Global | indent 4 | trim }}

defaults
    log global
    mode http
    option httplog
    {{ .Defaults | indent 4 | trim }}

listen stats
    bind *:4450
    stats enable
    stats uri /
    stats refresh 15s
    stats show-legends
    stats show-node {{ println "" }}

{{- range $frontend, $config := .Frontend }}
frontend {{$frontend}}
    {{$config | indent 4 | trim }}
{{ end }}

{{- range $backend, $config := .Backend }}
backend {{$backend}}
    {{ $config.Backend | indent 4 | trim }}
    server-template {{ $backend }}- {{ $config.Replicas }} tasks.{{ $backend }}:{{ $config.Port }} resolvers docker init-addr libc,none check
{{ end }}

{{ println ""}}
```

The data types passed into the template have the following format. The ingressClass is used to filter services. That way it is possible to run the separate stacks with separate controllers and a different ingress class if required.

Frontend snippets in the backend struct are executed as template and merged with the frontend config from the ConfigData struct. That is why it is not returned as json and not directly used in the template.

```go
type ConfigData struct {
 IngressClass string             `json:"-" mapstructure:"class"`
 Global       string             `json:"global,omitempty"`
 Defaults     string             `json:"defaults,omitempty"`
 Frontend     map[string]string  `json:"frontend,omitempty"`
 Backend      map[string]Backend `json:"backend,omitempty"`
}

type Backend struct {
 Port     string            `json:"port,omitempty"`
 Replicas uint64            `json:"replicas,omitempty"`
 Frontend map[string]string `json:"-"`
 Backend  string            `json:"backend,omitempty"`
}
```

### Configuration

The configuration can be fetched via the root endpoint of the manager service. The raw data to populate the template is also available in json format.

```shell
# returns the rendered template
curl -i localhost:8080
# returns the raw json data
curl -i -H 'Accept: application/json' localhost:8080
```

#### Example Config Response

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

listen stats
    bind *:4450
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
    default_backend my-stack_app
    use_backend test if { path -i -m beg "/test/" }

backend my-stack_app
    balance leastconn
    option httpchk GET /
    acl allowed_method method HEAD GET POST
    http-request deny unless allowed_method
    server-template my-stack_app- 2 tasks.my-stack_app:80 resolvers docker init-addr libc,none check

backend test
    http-request set-path "%[path,regsub(^/test/,/)]"
    server-template test- 1 tasks.test:80 resolvers docker init-addr libc,none check
```

#### Example JSON response

```json
{
  "global": "spread-checks 15\n",
  "defaults": "timeout connect 5s\ntimeout check 5s\ntimeout client 2m\ntimeout server 2m\nretries 1\nretry-on all-retryable-errors\noption redispatch 1\n",
  "frontend": {
    "default": "bind *:80\noption forwardfor except 127.0.0.1\noption forwardfor header X-Real-IP\nhttp-request disable-l7-retry unless METH_GET\ndefault_backend my-stack_app\nuse_backend test if { path -i -m beg \"/test/\" }"
  },
  "backend": {
    "my-stack_app": {
      "port": "80",
      "replicas": 2,
      "backend": "balance leastconn\noption httpchk GET /\nacl allowed_method method HEAD GET POST\nhttp-request deny unless allowed_method\n"
    },
    "test": {
      "port": "80",
      "replicas": 1,
      "backend": "http-request set-path \"%[path,regsub(^/test/,/)]\""
    }
  }
}
```

#### Update the template at runtime

It is possible to update the template at runtime via patch request.

```shell
curl -i -X PATCH localhost:8080/update --data-binary @path/to/template
```

## Local Development

If you have the repository locally, you can use the `Makefile` to run a example deployment.

```shell
make # build the images deploy a stack and a service
curl -i "localhost:3000" # backend my-stack_app
curl -i "localhost:3000/test/" # backend test
curl -i "localhost:4450" # haproxy stats page
curl -i "localhost:8080" # rendered template
curl -i -H 'Accept: application/json' "localhost:8080" # json data
make clean # remove the stack and service
```

## Haproxy Socket

If you publish the port 9876 on the loadbalancer you can use `socat` to connect to the socket cli.

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
