# Notes

## TODO

- [ ] Improve test setup
- [x] handle errors on manager
- [x] use multi staging and non root user
- [ ] allow to use manager with compose
- [x] implement config caching
- [x] use port from docker client if ingress port is not specified - not possible with swarm WONT FIX
- [x] use ingress class to filter

## Questions

- Should the loadbalancer socket be exposed over an http listener?
- should the lb fetch script run as haproxy sidecar process? see https://github.com/haproxy/haproxy/issues/1324
