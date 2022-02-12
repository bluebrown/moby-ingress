build: loadbalancer.build manager.build

%.build:
	docker build -t swarm-haproxy-$* $*/

compose:
	docker compose -f assets/yaml/compose.yml up

stack:
	docker stack deploy -c assets/yaml/stack.yml my-stack

cli-service:
	docker service create \
		--label 'ingress.class=haproxy' \
		--label 'ingress.frontend.default=use_backend {{ .Name }} if { path -i -m beg "/test/" }' \
		--label 'ingress.backend=http-request set-path "%[path,regsub(^/test/,/)]"' \
		--name test \
		--network my-stack_default \
		nginx
