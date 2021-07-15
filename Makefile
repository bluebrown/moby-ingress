stack: build
	docker stack deploy -c stack.yml my-stack

build:
	docker build -t bluebrown/swarm-haproxy-manager manager/
	docker build -t bluebrown/swarm-haproxy-loadbalancer loadbalancer/

clean:
	docker stack rm my-stack || true
