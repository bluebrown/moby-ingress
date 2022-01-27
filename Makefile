all: loadbalancer.build manager.build stack test

clean:
	docker service rm test || true
	docker stack rm my-stack || true

%.build:
	docker build -t swarm-haproxy-$* $*/

stack:
	docker stack deploy -c stack.yml my-stack

test:
	docker service create \
		--label ingress.class=haproxy \
		--label ingress.port=80 \
		--label 'ingress.frontend.default=use_backend {{ .Name }} if { path -i -m beg "/test/" }' \
		--label 'ingress.backend=http-request set-path "%[path,regsub(^/test/,/)]"' \
		--name test \
		--network my-stack_default \
		nginx

testclean:
	docker service rm test || true
