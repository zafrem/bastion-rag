.PHONY: mock-llm test-integration test-flow test-communication run-tests integration \
        submodule-update start-all stop-all \
        start-sentinel start-vault start-navigator start-anchor start-tracker

mock-llm:
	go run tests/mock-llm/main.go

# ── Start modules individually ─────────────────────────────────────────────
start-sentinel:
	./scripts/start-sentinel.sh
start-vault:
	./scripts/start-vault.sh
start-navigator:
	./scripts/start-navigator.sh
start-anchor:
	./scripts/start-anchor.sh
start-tracker:
	./scripts/start-tracker.sh

# ── Start / stop the whole stack ───────────────────────────────────────────
start-all:
	./scripts/start-all.sh
stop-all:
	./scripts/stop-all.sh

# ── Tests ──────────────────────────────────────────────────────────────────
test-integration:
	go test -v tests/integration_test.go

test-flow:
	go test -v tests/flow_test.go

test-communication:
	go test -v ./navigator/internal/vault/...

run-tests:
	./scripts/run-tests.sh

integration:
	./scripts/run-integration.sh

submodule-update:
	git submodule update --init --recursive
