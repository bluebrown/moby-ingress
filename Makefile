stack: build
	docker stack deploy -c stack.yml my-stack

build:
	docker build -t swarm-haproxy-manager manager/
	docker build -t swarm-haproxy-loadbalancer loadbalancer/

clean:
	docker stack rm my-stack || true
