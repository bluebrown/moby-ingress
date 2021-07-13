# Docker Swarm Haproxy Manager

## Run Example

Start the boilerplate with make.

```shell
make
```

Wait some time and use curl to fetch the nginx page.

```shell
curl localhost:3000
```

The dynamic configration is served by the manager.

```shell
curl localhost:8080
```
