all: loadbalancer.build manager.build

%.build:
	docker build -t swarm-haproxy-$* $*/

stack: loadbalancer.build manager.build
	docker stack deploy -c stack.yml my-stack

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
