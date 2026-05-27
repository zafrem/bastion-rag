.PHONY: mock-llm test-all

mock-llm:
	go run tests/mock-llm/main.go

test-integration:
	go test -v tests/integration_test.go

test-flow:
	go test -v tests/flow_test.go

test-communication:
	go test -v ./navigator/internal/vault/...

run-tests:
	./scripts/run-tests.sh

demo:
	./scripts/start-demo.sh

submodule-update:
	git submodule update --init --recursive
