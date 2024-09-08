build.client:
	CGO_ENABLED=0 GOOS=linux go build -o client -ldflags "-s -w" ./cmd/client/main.go
build.server:
	CGO_ENABLED=0 GOOS=linux go build -o server -ldflags "-s -w" ./cmd/server/main.go
run.client: build.client
	./client
run.server: build.server
	./server
docker:
	make gen
	make lint
	@docker compose -f dev-docker-compose.yml up --build --remove-orphans

migrate:
	@goose -dir migrations sqlite ./build/db-data/throttlr.db up
migrate.fresh:
	@goose -dir migrations sqlite ./build/db-data/throttlr.db redo

gen:
	@tailwindcss --input web/input.css --output assets/app.css --minify
	@swag init -g internal/handlers/server.go
	@go generate ./...

lint:
	@go mod tidy
	@templ fmt .
	@gofumpt -d -w .
	@golangci-lint run
	@swag fmt -d internal/handlers

test:
	@go test -v ./...

.PHONY: docker migrate migrate.fresh gen lint test
