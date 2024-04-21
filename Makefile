client.build: gen
	@docker build -t client -f ./build/Dockerfile.client .
client: client.build
	@docker run -it --rm -p 8080:8080 client
server.build:
	@docker build -t server -f ./build/Dockerfile.server .
server: server.build
	@docker run -it --rm -p 8081:8081 server

dev: gen
	@go run ./cmd/client/main.go

gen:
	@tailwindcss --input web/input.css --output assets/app.css
	@go generate ./...

test:
	@go test -v ./...

lint: gen
	@templ fmt .
	@gofumpt -d -w .
	@golangci-lint run

.PHONY: client.build client server.build server dev gen test lint
