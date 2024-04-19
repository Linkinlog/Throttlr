client.build: gen
	docker build -t client -f ./build/Dockerfile.client .
client: client.build
	docker run -it --rm -p 8080:8080 client
server.build:
	docker build -t server -f ./build/Dockerfile.server .
server: server.build
	docker run -it --rm -p 8081:8081 server

gen:
	go generate ./...

test:
	go test -v ./...

.PHONY: client.build client server.build server
