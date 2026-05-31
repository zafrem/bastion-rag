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
