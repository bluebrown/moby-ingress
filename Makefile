stack: build
	docker stack deploy -c stack.yml my-stack

run: build
	docker network create --driver overlay --attachable testnet
	docker run -ti  -v /var/run/docker.sock:/var/run/docker.sock -p 8080:8080 --name manager --rm --network testnet -d manager
	docker run --network testnet -e MANAGER_ENDPOINT=http://manager:8080 --name lb -p 3000:80 --rm -d lb
	docker service create --name bar --label ingress.path=/ --label ingress.port=80 --network testnet  nginx

build:
	docker build -t manager manager/
	docker build -t lb loadbalancer/

clean:
	docker stack rm my-stack || true
	docker service rm bar || true
	docker stop lb manager || true
	sleep 10
	docker network rm testnet || true
	sleep 20
	docker image rm lb manager || true