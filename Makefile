.PHONY: help setup teardown test build run dev clean install demo populate-demo

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'



teardown: ## Stop and remove NATS test environment
	docker compose -f test/docker/docker-compose.yml down -v

test: setup ## Run tests against local NATS
	go test -v ./...
	$(MAKE) teardown

build: ## Build the binary
	go build -o bin/n2s ./cmd/n2s

run: build ## Build and run against local NATS
	./bin/n2s --server nats://localhost:4222

dev: setup run ## Setup environment and run (for development)

clean: teardown ## Clean build artifacts and stop containers
	rm -rf bin/
	go clean

install: build ## Install binary to $GOPATH/bin
	go install ./cmd/n2s

demo: ## Start NATS and populate with demo data for screenshots/videos
	@echo "🚀 Setting up demo environment..."
	@docker compose -f test/docker/docker-compose.yml up -d
	@echo "Waiting for NATS to be healthy..."
	@timeout 30 sh -c 'until docker exec n2s-test-nats wget -q -O- http://localhost:8222/healthz > /dev/null 2>&1; do sleep 1; done' || true
	@sleep 2
	@echo "📝 Populating demo data..."
	@chmod +x test/scripts/populate-demo.sh
	@./test/scripts/populate-demo.sh
	@echo ""
	@echo "✨ Demo environment ready!"
	@echo "Run 'make run' to view the data with n9s"

populate-demo: ## Populate demo data (assumes NATS is already running)
	@chmod +x test/scripts/populate-demo.sh
	@./test/scripts/populate-demo.sh

