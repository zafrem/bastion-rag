# Design Document: Bastion-RAG Interactive Demo

## Overview
A visual demonstration platform for the Bastion-RAG security framework. It allows users to see exactly how their prompts are handled across the multi-layered security pipeline.

## System Architecture

### 1. Backend: Demo Orchestrator (`cmd/demo-api`)
- **Language**: Go
- **Function**: Receives a prompt and executes a simulated bidirectional flow through the core modules (Sentinel, Vault, Navigator, Anchor, Mock LLM).
- **Endpoint**: `POST /api/process`
- **Response Format**:
  ```json
  {
    "steps": [
      { "id": "sentinel-in", "name": "Sentinel (In)", "info": "No injection detected.", "status": "success" },
      { "id": "vault-p1", "name": "Vault (Phase 1)", "info": "Anonymized PII: [TOKEN_PERSON_1]", "status": "success" },
      ...
    ],
    "final_output": "The response text..."
  }
  ```

### 2. Frontend: Interactive Dashboard (`ui/`)
- **Framework**: Next.js (React) + TailwindCSS
- **Animation**: Framer Motion
- **Features**:
  - Central input field for user prompts.
  - Vertical/Vertical flow visualization with "nodes" representing each module.
  - Sequential highlighting: nodes "glow" as data passes through them.
  - Detail tooltips: clicking/hovering a node shows the specific input/output for that stage.
  - Dark-mode optimized high-tech aesthetic (cybersecurity theme).

## Flow Diagram
1. User enters: "What is John Doe's balance?"
2. UI calls `/api/process`.
3. Backend processes:
   - Sentinel: Safe.
   - Vault: Anonymizes "John Doe" -> "[USER_1]".
   - Navigator: Retrieves "Balance: $500".
   - Anchor: Adds noise.
   - LLM: Generates "[USER_1] has $500".
   - ...
4. UI receives all steps and starts the animation sequence.
5. Sentinel In lights up -> Vault lights up -> ... -> User Output appears.

## Execution Plan
1.  **Phase 1: Backend Implementation**
    *   Create `cmd/demo-api/main.go`.
    *   Refactor `tests/flow_test.go` logic into a reusable service.
2.  **Phase 2: Frontend Setup**
    *   Initialize Next.js project.
    *   Implement the base layout and "Node" components.
3.  **Phase 3: Animation & Integration**
    *   Implement the sequential highlight logic.
    *   Connect the frontend to the Go backend.
