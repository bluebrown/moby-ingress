# Docker Swarm Haproxy Manager

The aim of this project is to allow writing haproxy configuration in docker-compose.yml files. Currently it only works when deploying the services with swarm.

## Synopsis

```yml
# the proxy service fetches the dynamic configuration
# from the manager periodically
haproxy:
    image: bluebrown/swarm-ingress-haproxy
    environment: 
        MANAGER_ENDPOINT: http://manager:8080
    depends_on:
        - manager
    ports:
        - 3000:80 # ingress port
        - 4450:4450 # stats page

# the manager service defines global defaults and frontend configs
manager:
    image: bluebrown/swarm-ingress-manager
    # default template path, this is not required
    # but can be useful when mounting a config file as volume
    command: --template /src/haproxy.cfg.template 
    volumes: 
        -  /var/run/docker.sock:/var/run/docker.sock
    ports:
        - 8080:8080
    labels:
        ingress.defaults: |
            timeout connect 5s
            timeout check 5s
            timeout client 2m
            timeout server 2m
        ingress.global: |
            spread-checks 15
        ingress.frontend.default: |
            bind *:80

# each app service defines its own backend config
# and can provide a frontend snippet for 1 or more frontend.
# The snippet will be merged with the frontend config from the
# manager service
some-app:
    image: nginx
    deploy:
        replicas: 2
        labels:
            # the application port inside the container
            ingress.port: "80"
            # rules are merged with corresponding frontend
            ingress.frontend.default: |
                use_backend {{ .Name }} if { path -i -m beg /foo/ }
            # backend snippet are added to the backend created from
            # this service definition
            ingress.backend: |
                balance roundrobin
                option httpchk GET /
                http-request set-path "%[path,regsub(^/foo/,/)]"
```

> Note  
> *Be careful which ports you publish in production*

## Template

The labels are parsed and passed into a haproxy.cfg template. The default template looks like the below.

```go
listen stats
    bind *:4450
    stats enable
    stats uri /
    stats refresh 15s
    stats show-legends
    stats show-node

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
    {{ .Defaults | indent 4 | trim }}{{ println "" }}

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

The data types passed into the template have the following format.

```go
type ConfigData struct {
 Global   string             `json:"global,omitempty"`
 Defaults string             `json:"defaults,omitempty"`
 Backend  map[string]Backend `json:"backend,omitempty"`
 // frontend contains merged snippet from backend
 Frontend map[string]string  `json:"frontend,omitempty"`
}

type Backend struct {
 Port     string            `json:"port,omitempty"`
 Replicas uint64            `json:"replicas,omitempty"`
 Backend  string            `json:"backend,omitempty"`
 // frontend snippets are executed as template and merged
 // with the frontend config from the ConfigData struct
 // so its no returned as json and not really used in the template
 Frontend map[string]string `json:"-"`
}
```

### Update the template at runtime

It is possible to update the template at runtime via post request.

```shell
curl -i -X POST localhost:8080/update --data-binary @path/to/template
```

## Example

Use the `Makefile` to run a example deployment.

```shell
make
curl -i localhost:3000 # backend app
curl -i localhost:3000/foo/ # backend foo
curl -i localhost:4450 # stats page
```

The configuration can be fetched via the root endpoint of the manager service. The raw data to populate the template is available in json format.

```shell
curl -i localhost:8080
curl -i localhost:8080/json
```

### Example JSON response

```json
{
    "global": "spread-checks 15\n",
    "defaults": "timeout connect 5s\ntimeout check 5s\ntimeout client 2m\ntimeout server 2m\n",
    "frontend": {
        "default": "bind *:80\noption forwardfor except 127.0.0.1\noption forwardfor header X-Real-IP\nuse_backend my-stack_foo if { path -i -m beg /foo/ }\n"
    },
    "backend": {
        "my-stack_foo": {
            "port": "80",
            "replicas": 2,
            "backend": "balance roundrobin\noption httpchk GET /\nhttp-request set-path \"%[path,regsub(^/foo/,/)]\"\n"
        }
    }
}
```

### Example Config Response

```c
listen stats
    bind *:4450
    stats enable
    stats uri /
    stats refresh 15s
    stats show-legends
    stats show-node

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
    spread-checks 15

defaults
    log global
    mode http
    option httplog
    timeout connect 5s
    timeout check 5s
    timeout client 2m
    timeout server 2m

frontend default
    bind *:80
    option forwardfor except 127.0.0.1
    option forwardfor header X-Real-IP
    use_backend my-stack_foo if { path -i -m beg /foo/ }

backend my-stack_foo
    balance roundrobin
    option httpchk GET /
    http-request set-path "%[path,regsub(^/foo/,/)]"
    server-template my-stack_foo- 2 tasks.my-stack_foo:80 resolvers docker init-addr libc,none check

```
