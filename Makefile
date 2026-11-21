.PHONY: workbench test-workbench mock-llm test-integration test-flow test-communication run-tests integration \
        submodule-update start-all stop-all \
        start-sentinel start-vault start-navigator start-anchor start-tracker

mock-llm:
	go run tests/mock-llm/main.go

# ── Workbench (demo UI + pipeline tests; talks to modules over HTTP only) ──
workbench:
	cd workbench && go run .
test-workbench:
	cd workbench && go vet ./... && go test ./...

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
	go test -v ./tests/ -run "TestIntegration|TestFullPipeline|TestCase|TestHealth"

test-flow:
	go test -v ./tests/ -run "TestBidirectionalFlow|TestDemoScenarios|TestPipeline"

test-communication:
	go test -v ./navigator/internal/vault/...

run-tests:
	./scripts/run-tests.sh

integration:
	./scripts/run-integration.sh

submodule-update:
	git submodule update --init --recursive
