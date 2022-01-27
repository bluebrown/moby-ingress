# Notes

## TODO

- [ ] Improve test setup
- [x] handle errors on manager
- [ ] use multi staging and non root user
- [ ] allow to use manager with compose
- [ ] implement config caching
- [ ] use port from docker client if ingress port is not specified
- [x] use ingress class to filter

## Questions

- Should the loadbalancer socket be exposed over an http listener?
- should the lb fetch script run as haproxy sidecar process? see https://github.com/haproxy/haproxy/issues/1324
