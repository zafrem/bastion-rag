# Bastion-Navigator Federation Extension SRS

**Project:** Bastion-RAG - RAG Security Governance Framework
**Document Type:** Module Extension SRS (Tier 2.5)
**Document ID:** 22-navigator-federation-ext
**Module:** C - Navigator (extension)
**Version:** 1.0
**Date:** 2026-05-27
**Status:** Draft

**Base Module Reference:** 12-navigator-srs (v3.0)
**Foundation References:**
- 01-architecture-principles (v3)
- 02-event-schema-standard

---

## 1. Introduction

### 1.1 Purpose

This document specifies the **Federation** and **Agent** operational modes for Navigator. It enables distributed RAG across multiple Navigator instances operated by different teams, each with domain-specialized vector databases and local LLMs.

### 1.2 Problem

A team operating a domain-specialized Navigator excels within its own domain but lacks coverage in adjacent domains. Centralizing all data into one Navigator breaks data isolation and team autonomy. This extension allows teams to federate as peers: each Navigator can query others' vector databases to fill cross-domain knowledge gaps without moving or duplicating data.

### 1.3 Design Principle

**Retrieval is federated. Synthesis is always local.**

Peers are queried as additional search sources. The LLM that generates the final answer is always the originating Navigator's own LLM. In `federation` mode, peers return only search results — they never generate answers. In `agent` mode, peers may optionally invoke their local LLM, but this is explicit and opt-in.

```
User query → Navigator A
               ├── local search (Qdrant)
               ├── peer search → Navigator B → local search → results
               └── peer search → Navigator C → local search → results
                        ↓
               RRF merge: local + B results + C results
                        ↓
               Navigator A's local LLM synthesizes answer
                        ↓
               Answer to user
```

### 1.4 Relationship to Base SRS §1.5 ("Single-Direction")

The base SRS states Navigator is single-direction by nature (INPUT path search). This extension does not change that. Federation extends the retrieval fan-out horizontally across peers — all within the INPUT path search direction. Navigator remains a search module; it does not gain output-path responsibilities.

---

## 2. Operational Modes

A new top-level `mode` configuration field is added to Navigator:

| Mode | Description | Backward Compatible |
|---|---|---|
| `search` | Current behavior. Standalone, no peers. **Default.** | — |
| `federation` | Distributed vector search. Peers act as additional search sources. No peer LLM calls. | `search` unchanged |
| `agent` | Federation + peers may invoke their local LLM for domain answers. | `federation` unchanged |

All existing deployments use `mode: search` by default. No configuration change is required to preserve current behavior.

---

## 3. Federation Mode

### 3.1 Architecture

```
┌──────────────────────────────────────────────────────────┐
│           Navigator Service (Python) — federation mode    │
├──────────────────────────────────────────────────────────┤
│                                                           │
│  15-step pipeline (steps 1–4, 8–15 unchanged)             │
│                                                           │
│  Step 5 (extended):                                       │
│  ┌────────────────────────────────────────────────────┐   │
│  │  Vector Search                                     │   │
│  │  ├── local:  QdrantSearcher (unchanged)            │   │
│  │  └── peers:  FederationRouter → [PeerClient, ...]  │   │
│  │              parallel asyncio.gather               │   │
│  └────────────────────────────────────────────────────┘   │
│                                                           │
│  Step 7 (extended):                                       │
│  ┌────────────────────────────────────────────────────┐   │
│  │  RRF Fusion                                        │   │
│  │  local_results + peer_A_results + peer_B_results   │   │
│  │  → same formula: score(d) = Σ 1/(k + rank_i(d))   │   │
│  │  (N lists instead of 2; algorithm unchanged)       │   │
│  └────────────────────────────────────────────────────┘   │
│                                                           │
└──────────────────────────────────────────────────────────┘
```

### 3.2 New Components

**FederationRouter** — selects which peers to query for a given request:

```python
class FederationRouter:
    def __init__(self, peers: list[PeerConfig], embedder: Embedder):
        # Pre-compute affinity embeddings for each peer at startup
        self._peer_embeddings = {
            p.id: embedder.encode(" ".join(p.topic_affinity))
            for p in peers
        }

    def route(
        self,
        query_embedding: np.ndarray,
        local_confidence: float,
        config: FederationConfig,
    ) -> list[PeerConfig]:
        if local_confidence >= config.confidence_threshold:
            return []  # local results are sufficient; no fan-out

        scores = {
            peer_id: cosine_similarity(query_embedding, aff_emb)
            for peer_id, aff_emb in self._peer_embeddings.items()
        }
        return sorted(
            [p for p in self._peers if scores[p.id] >= config.routing_threshold],
            key=lambda p: scores[p.id],
            reverse=True,
        )[: config.max_peers_per_query]
```

**PeerClient** — gRPC client to a single peer Navigator:

```python
class PeerClient:
    async def search(
        self, req: SearchRequest, origin_id: str, hop_depth: int
    ) -> SearchResponse:
        metadata = [
            ("x-origin-id", origin_id),
            ("x-hop-depth", str(hop_depth)),
        ]
        # Calls peer's existing Search gRPC method — no new interface on peer
        return await self._stub.Search(req, metadata=metadata)
```

Peer Navigators receive a standard `Search` call with additional metadata headers. No changes are required on the peer side to support being queried in federation mode.

### 3.3 Topic Affinity Routing

Each peer declares its domain coverage via `topic_affinity` tags in config. At Navigator startup, the FederationRouter embeds each peer's concatenated topic tags using the same local embedder (bge-m3). At query time, peer selection is cosine similarity between the query embedding and each peer's affinity embedding.

This avoids broadcasting to all peers and routes only to those likely to have relevant results.

### 3.4 Confidence-Based Triggering

Fan-out only occurs when local results are insufficient:

```python
local_results = await self._searcher.search(query_vector, collections)
local_confidence = max((r.score for r in local_results), default=0.0)

peers_to_query = self._router.route(
    query_embedding, local_confidence, self._config.federation
)

if peers_to_query:
    peer_responses = await asyncio.gather(
        *[
            client.search(req, self._config.id, hop_depth=1)
            for client in [self._peer_clients[p.id] for p in peers_to_query]
        ],
        return_exceptions=True,  # timed-out peers are skipped, not fatal
    )
    all_lists = [local_results] + [
        r.results for r in peer_responses if not isinstance(r, Exception)
    ]
    return rrf_merge(all_lists, k=self._config.search.rrf_k)
else:
    return local_results
```

Most queries (within a team's own domain) will have high local confidence and never fan out. Cross-domain queries trigger federation automatically.

### 3.5 Loop Prevention

Every peer call includes these gRPC metadata headers:

| Header | Value | Rule enforced by receiver |
|---|---|---|
| `x-origin-id` | ID of the Navigator that initiated the query | Peer does not fan out if this matches its own ID |
| `x-hop-depth` | Integer, starts at 1 for first peer hop | Peer does not fan out if value `>= max_depth` |

Receiving Navigators check these headers before calling FederationRouter. If either condition is met, they perform local search only and return results without further fan-out. This prevents infinite loops in any P2P topology.

---

## 4. P2P Topology

### 4.1 No Central Coordinator

Each Navigator is simultaneously a client (queries peers) and a server (receives peer queries). There is no routing authority or global registry. The full topology is defined by each Navigator's local `peers` config.

### 4.2 Service Discovery

| Method | When to use |
|---|---|
| Static config | Default. Peer endpoints listed in `config.yaml`. Simple and predictable. |
| DNS-SD | Dynamic environments. Peers register `_navigator._grpc.local` SRV records. |

### 4.3 Example Topology

Three teams, fully connected. Any team can query either other.

```
Team A (customer)  ←──────────────→  Team B (manufacturing)
        ↑                                       ↑
        └───────────────────────────────────────┘
                Team C (HR/compliance)
```

Team A peers: [B, C]  
Team B peers: [A, C]  
Team C peers: [A, B]

Loop prevention handles the cycle: if A queries B, B will not re-query A because `x-origin-id == B's own ID` check blocks it.

---

## 5. Agent Mode

Agent mode extends federation: peers with a local LLM can optionally generate a domain-specific answer rather than returning only search results. Use when a domain's knowledge is primarily in a specialist LLM's weights, not fully indexed in its vector DB.

### 5.1 New Endpoint (Agent Peers Only)

Peers running in `agent` mode expose:

```
POST /v1/navigator/agent/generate
```

**Request:**
```json
{
  "query": "What is the required maintenance interval for the CNC-200?",
  "context": [
    {"content": "CNC-200 spec sheet excerpt...", "score": 0.81}
  ],
  "max_tokens": 500,
  "tenant_id": "tenant-acme"
}
```

**Response:**
```json
{
  "answer": "The CNC-200 requires preventive maintenance every 500 operating hours, per section 4.2 of the maintenance manual.",
  "sources": ["mfg_docs/cnc200_maintenance_v3"],
  "model": "llama3.2:8b",
  "confidence": 0.89
}
```

### 5.2 Capability Declaration

Each peer declares whether it accepts `Search` only or can also generate answers:

```yaml
peers:
  - id: team-manufacturing
    endpoint: navigator-mfg:9092
    topic_affinity: [manufacturing, equipment maintenance]
    capability: search          # default — standard Search gRPC only

  - id: team-specialist
    endpoint: navigator-spec:9092
    topic_affinity: [niche_regulatory_domain]
    capability: agent           # can generate answers via local LLM
```

The originating FederationRouter calls `Search` for `capability: search` peers and `/v1/navigator/agent/generate` for `capability: agent` peers. Agent responses are merged into the context alongside vector search results before local LLM synthesis.

### 5.3 Local LLM Configuration (Agent Mode)

```yaml
navigator:
  agent:
    local_llm:
      provider: ollama           # ollama | llamacpp | custom_http
      endpoint: http://localhost:11434
      model: llama3.2:8b
      max_tokens: 2048
      timeout_seconds: 30
      context_window: 8192
```

---

## 6. Governance

### 6.1 Permission Enforcement (No New Protocol)

The originating Navigator passes `user.allowed_categories` in the standard `SearchRequest`. Peer Navigators enforce their own Vault permission filter (FR-ENH-PF-001) against these categories before returning results. No cross-team permission protocol is required — the existing `SearchWithPermissions` interface handles this.

If a user does not have `manufacturing` permission, Team B returns empty results for manufacturing documents. The originating Navigator receives what the user is allowed to receive.

### 6.2 Tenant Isolation

`tenant_id` is forwarded in gRPC metadata on every peer call. Peers enforce the multi-tenancy pre-filter hook (`navigator.tenant.prefilter`) independently before performing any search. Cross-tenant data leakage is prevented at the peer level, not the originating Navigator.

### 6.3 Response Transform

Federated results are subject to the same Vault Phase 2 transform as local results. If the user holds `anonymized` access level, peer-sourced results are returned in anonymized form (applied by the peer before returning). The originating Navigator does not re-apply transforms to peer results.

---

## 7. Configuration

```yaml
navigator:
  mode: federation              # search | federation | agent

  federation:
    confidence_threshold: 0.70  # fan-out if local max score < this
    routing_threshold: 0.40     # route to peer only if affinity score > this
    max_peers_per_query: 3      # max simultaneous outbound peer queries
    max_depth: 2                # max federation hops (loop prevention)
    peer_timeout_ms: 2000       # timeout per peer; timed-out peers are skipped
    rrf_k: 60                   # same constant as local RRF

    peers:
      - id: team-manufacturing
        endpoint: navigator-mfg:9092
        topic_affinity:
          - manufacturing processes
          - quality control
          - supply chain
          - equipment maintenance
        capability: search       # search | agent

      - id: team-hr
        endpoint: navigator-hr:9092
        topic_affinity:
          - human resources
          - employee benefits
          - payroll
          - compliance training
        capability: search

  agent:
    local_llm:
      provider: ollama
      endpoint: http://localhost:11434
      model: llama3.2:8b
      max_tokens: 2048
      timeout_seconds: 30
```

---

## 8. Performance Targets

| Scenario | Latency target (p95) |
|---|---|
| Local confidence sufficient (no fan-out) | Same as standalone: <150ms |
| Federation, 2 peers in parallel | <300ms |
| Federation, 3 peers in parallel | <350ms |
| One peer times out (skipped) | +`peer_timeout_ms` worst case, then continues |
| Agent mode, 1 peer LLM call | <5s (LLM latency dominates) |

Peer queries are always executed in parallel via `asyncio.gather`. The added latency is bounded by the slowest responding peer (or `peer_timeout_ms`, whichever comes first). Timed-out peers are omitted from results but do not fail the request.

---

## 9. Events

All use Foundation BastionEvent schema (`module: "navigator"`, `module_version: "3.0.0"`).

```
bastion-rag.events.navigator.federation_started
  → query_id, peers_queried, local_confidence

bastion-rag.events.navigator.federation_completed
  → query_id, total_results, local_count, remote_count, latency_ms

bastion-rag.events.navigator.peer_timeout
  → peer_id, timeout_ms, query_id

bastion-rag.events.navigator.peer_skipped_loop
  → peer_id, reason (origin_id_match | max_depth_reached)

bastion-rag.events.navigator.agent_generated
  → peer_id, model, confidence, latency_ms
```

---

## 10. Non-Functional Requirements

| ID | Requirement |
|---|---|
| NFR-FED-001 | `mode: search` behavior is identical before and after this extension is deployed |
| NFR-FED-002 | Peer timeout must not fail the overall request; timed-out peer results are omitted |
| NFR-FED-003 | Loop prevention (x-origin-id + x-hop-depth) is enforced on every incoming peer request |
| NFR-FED-004 | Each peer enforces its own Vault permissions; the originating Navigator does not override them |
| NFR-FED-005 | Peer queries are always parallel (asyncio.gather); sequential fan-out is not permitted |
| NFR-FED-006 | FederationRouter routing decisions (peers selected, reason) are logged per query |
| NFR-FED-007 | Affinity embeddings are pre-computed at startup; no per-query embedding of peer tags |
| NFR-FED-008 | Agent mode endpoint `/v1/navigator/agent/generate` is not exposed in `search` or `federation` mode |

---

## 11. Change History

| Version | Date | Changes |
|---|---|---|
| 1.0 | 2026-05-27 | Initial draft |

---

**End of Document**
