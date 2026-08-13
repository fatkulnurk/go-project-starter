.PHONY: build vet test test-integration lint arch check run-api run-worker fmt

build:
	go build ./...

vet:
	go vet ./...

test:
	go test ./...

test-integration:
	go test -tags=integration ./...

lint:
	golangci-lint run

arch:
	go-arch-lint check

fmt:
	go fmt ./...

check: build vet test fmt
	@echo "=== dependency direction check ==="
	@echo "--- domain must stay pure (no sql/http/framework) ---"
	@! rg -n "database/sql|net/http|go-chi|aws-sdk|asynq|go-redis" internal/modules -g '**/domain/**' || true
	@echo "--- modules only import each other through package roots ---"
	@! rg -n "internal/modules/(rbac|media)/(domain|application|infrastructure|adapters)" internal/modules/auth || true
	@! rg -n "internal/modules/(auth|media)/(domain|application|infrastructure|adapters)" internal/modules/rbac || true
	@! rg -n "internal/modules/(auth|rbac)/(domain|application|infrastructure|adapters)" internal/modules/media || true

run-api:
	go run ./cmd/api

run-worker:
	go run ./cmd/worker
