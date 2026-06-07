# Integration Testing Guide: Bastion-RAG

This document outlines the strategy and procedures for performing full integration testing of the Bastion-RAG framework. Since the framework consists of multiple distributed modules, integration testing focuses on verifying the bidirectional data flow and the security handshakes between them.

## 1. Testing Architecture

The integration test environment uses a **Mock LLM** to eliminate external dependencies and provide deterministic results.

```
[ Test Suite ] <--> [ Sentinel ] <--> [ Vault ] <--> [ Navigator ] <--> [ Anchor ] <--> [ Mock LLM ]
       |                                                                                  ^
       +---------------------------- Logs & Verifies --------------------------------------+
```

### The Mock LLM (`tests/mock-llm`)
The Mock LLM acts as a high-fidelity simulator of an Ollama-compatible API.
- **Port:** `11435`
- **Capabilities:**
    - Logs all incoming requests (Method, Path, and Body).
    - Simulates `/api/generate`, `/api/chat`, and `/api/embeddings`.
    - Returns predictable, hardcoded responses for verification.

## 2. Test Scenarios

### A. Input Path (Sentinel-IN -> Vault -> Anchor)
- **Goal:** Verify that a user query is correctly intercepted, anonymized, and embedded with noise.
- **Verification:**
    - Check Sentinel logs for "injection blocked" or "metadata validated".
    - Check Vault logs for PII tokenization.
    - Check Mock LLM logs to see the final "processed" query received.

### B. Output Path (Anchor -> Navigator -> Vault -> Sentinel-OUT)
- **Goal:** Verify that LLM responses are de-anonymized, checked for hallucinations, and filtered for PII leaks.
- **Verification:**
    - Check Sentinel logs for "pii_incidents" or "grounding_score".
    - Verify the final response received by the test suite is correctly sanitized.

## 3. Running the Tests

### Prerequisite: Start the Mock LLM
The Mock LLM must be running to receive and log traffic from the modules.
```bash
make mock-llm
```

### 3. Run Integration Tests
There are two levels of integration tests:

#### A. Automated Communication Test
This test verifies real network communication between the `Navigator` and `Vault` modules.
```bash
make test-communication
```

#### B. Full Bidirectional Flow Demonstration
This test traces a prompt through all 10 steps of the Bastion-RAG security lifecycle.
```bash
make test-flow
```

#### C. Full Integration Package (recommended)
Runs the entire `tests/` package end-to-end: it starts the Mock LLM automatically,
then exercises the Sentinel-IN → Vault → Navigator → Vault → Sentinel-OUT pipeline.
Tests that spawn the Vault/Navigator subprocesses run when the tooling is present and
skip themselves otherwise.
```bash
make integration            # → scripts/run-integration.sh
# Force the in-process-only subset (no subprocesses):
BASTION_SKIP_SUBPROCESS=1 make integration
```

## 4. Manual Verification Flow

To manually verify the flow, you can point your module configurations to the Mock LLM (`http://localhost:11435`) and observe the terminal output of the `mock-llm` process.

**Example Mock LLM Log Output:**
```text
2026/05/23 10:32:34 RECEIVED: POST /api/generate | Body: {"model":"llama3","prompt":"Hello, mock LLM!","stream":false}
2026/05/23 10:32:34 COMPLETED: POST /api/generate in 925.092µs
```

## 5. Module-Specific Test Commands
Each submodule contains its own internal test suite. You can run them individually:

- **Sentinel:** `cd sentinel && go test ./...`
- **Vault:** `cd vault && go test ./...`
- **Navigator:** `cd navigator && go test ./...`
- **Anchor:** `cd anchor && go test ./...`
- **Tracker:** `cd tracker && go test ./...`

## 6. Running the Modules Separately

Each module runs as its own independent process. Start them one at a time (each
script runs in the foreground and owns its terminal) or launch the whole stack in
the background.

```bash
make start-sentinel     # or: ./scripts/start-sentinel.sh
make start-vault
make start-navigator
make start-anchor
make start-tracker

make start-all          # background-launch all five; PIDs tracked under run/
./scripts/status.sh     # show what is running
make stop-all           # stop everything start-all launched
```

Logs from `start-all` land in `logs/<module>.log`; PID files live in `run/`
(both git-ignored).

### Default ports

| Module    | REST | gRPC | Other |
|-----------|------|------|-------|
| Sentinel  | 8080 | 9090 | |
| Vault     | 8081 | 9091 | |
| Navigator | 8082 | 9092 | |
| Anchor    | 8083 | 9093 | |
| Tracker   | 8080 | 9090 | WebSocket 8081 · metrics 9091 · dashboard served at REST `:8080/` |

> **Port conflict:** in the default configs Tracker's REST (`8080`) collides with
> Sentinel (`8080`), and Tracker's WebSocket (`8081`) collides with Vault (`8081`).
> To run the full stack natively at once, override Tracker's `server.rest_port` /
> `server.websocket_port` (in `tracker/config/config.yaml`) — or run Tracker on a
> separate host/container. `make start-all` starts Tracker last so the clash is
> obvious in `logs/tracker.log`.

The operator dashboard (audit, metrics, live flow) is served entirely by **Tracker**
at http://localhost:8080 (the same port as its REST API) — there is no separate UI
module. Note: `tracker/config/config.yaml` still defines an unused `ui_port: 3000`;
the dashboard is actually served by the REST server on `server.rest_port`.
