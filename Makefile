stack: build
	docker stack deploy -c stack.yml my-stack

build:
	docker build -t swarm-haproxy-manager manager/
	docker build -t swarm-haproxy-loadbalancer hapi/

clean:
	docker stack rm my-stack || true


test:
	docker service create \
		--label ingress.port=80 \
		--label 'ingress.frontend.default=use_backend {{ .Name }} if { path -i -m beg "/test/" }' \
		--label 'ingress.backend=http-request set-path "%[path,regsub(^/test/,/)]"' \
		--name test \
		--network my-stack_default \
		nginx

testclean:
	docker service rm test || true