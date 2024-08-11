client.build: gen
	@docker build --build-arg ENV=dev -t client -f ./build/Dockerfile.client .
client: client.build
	@docker run -it --rm -p 8080:8080 client
server.build:
	@docker build -t server -f ./build/Dockerfile.server .
server: server.build
	@docker run -it --rm -p 8081:8081 server

migrate:
	@goose -dir migrations sqlite ./build/db-data/throttlr.db up
migrate.fresh:
	@goose -dir migrations sqlite ./build/db-data/throttlr.db redo

dev: gen
	@go run ./cmd/client/main.go

watch.go:
	air

watch.t:
	templ generate --watch --proxy='http://localhost:8080'

watch.tw:
	tailwindcss --input web/input.css --output assets/app.css --minify -w

gen:
	@tailwindcss --input web/input.css --output assets/app.css --minify
	@swag init -g internal/handlers/server.go
	@go generate ./...

test:
	@go test -v ./...

lint: gen
	@go mod tidy
	@templ fmt .
	@gofumpt -d -w .
	@golangci-lint run
	@swag fmt -d internal/handlers

docker: gen
	@docker-compose up --build

.PHONY: client.build client server.build server dev gen test lint
