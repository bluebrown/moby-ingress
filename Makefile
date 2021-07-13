run:
	docker network create --driver overlay --attachable testnet
	docker build -t manager manager/
	docker build -t lb loadbalancer/
	docker run -ti  -v /var/run/docker.sock:/var/run/docker.sock -p 8080:8080 --name manager --rm --network testnet -d manager
	docker run --network testnet -e MANAGER_ENDPOINT=http://manager:8080 --name lb -p 3000:80 --rm -d lb
	docker service create --name bar --label ingress.path=/ --label ingress.port=80 --network testnet  nginx

clean:
	docker service rm bar
	docker stop lb manager
	docker network rm testnet