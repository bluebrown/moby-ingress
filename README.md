# Docker Swarm Haproxy Manager

## Run Example

```shell
make
```

Wait some time and use curl to fetch the nginx page.

```shell
curl localhost:3000
```

The dynamic configuration is served by the manager.

```shell
curl localhost:8080
```

If you are done with testing use the clean command.

```shell
make clean
```
