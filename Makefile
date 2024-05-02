client.build: gen
	@docker build --build-arg ENV=dev -t client -f ./build/Dockerfile.client .
client: client.build
	@docker run -it --rm -p 8080:8080 client
server.build:
	@docker build -t server -f ./build/Dockerfile.server .
server: server.build
	@docker run -it --rm -p 8081:8081 server

migrate:
	@goose -dir migrations sqlite ./throttlr.db up

dev: gen
	@go run ./cmd/client/main.go

gen:
	@tailwindcss --input web/input.css --output assets/app.css --minify
	@go generate ./...

test:
	@go test -v ./...

lint: gen
	@go mod tidy
	@templ fmt .
	@gofumpt -d -w .
	@golangci-lint run

.PHONY: client.build client server.build server dev gen test lint
