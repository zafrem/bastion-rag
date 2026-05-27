# Bastion Architecture & Design Principles

**Project:** Bastion - RAG Security Governance Framework  
**Document Type:** Foundation (Tier 1)  
**Document ID:** 01-architecture-principles  
**Version:** 1.0  
**Date:** 2026-05-17  
**Status:** Draft

---

## 1. Introduction

### 1.1 Purpose

This document establishes the foundational architecture and design principles for the Bastion RAG Security Governance Framework. It serves as the **single source of truth** for architectural decisions that all module and cross-cutting specifications must follow.

### 1.2 Audience

- Module implementation teams
- Cross-cutting feature developers
- System architects
- Integration engineers

### 1.3 Document Hierarchy

```
Tier 1: Foundation (THIS DOCUMENT + 2 others)
        ↓ defines principles for
Tier 2: Module SRS (5 modules)
        ↓ uses
Tier 3: Cross-cutting SRS (features)
        ↓ summarized in
Tier 4: Integration (master overview)
```

---

## 2. System Overview

### 2.1 What is Bastion?

Bastion is a **data governance framework** for RAG (Retrieval-Augmented Generation) systems. Despite the name suggesting a fortress, Bastion does not simply block data—it **governs the safe flow** of data through the RAG pipeline.

```
Philosophy:
"We don't block data; we govern its safe flow."

Bastion ensures data flows freely but safely,
transforming and protecting it at each stage
rather than simply blocking access.
```

### 2.2 The Five Modules

| Module | Name | Primary Responsibility |
|---|---|---|
| **A** | Sentinel | Input/Output validation gateway |
| **B** | Vault | Data isolation & anonymization |
| **C** | Navigator | Search & ranking |
| **D** | Tracker | Observability & audit |
| **E** | Anchor | Embedding security |

### 2.3 Pipeline Position

```
                    User Query
                       ↓
        ┌──────────────────────────────┐
        │  Input Pipeline               │
        │  A → B → C → E → LLM           │
        └──────────────────────────────┘
                       ↓
                      LLM
                       ↓
        ┌──────────────────────────────┐
        │  Output Pipeline              │
        │  LLM → E → B → A → User        │
        └──────────────────────────────┘
                       ↓
        ┌──────────────────────────────┐
        │  D (Tracker): cross-cutting   │
        │  observes entire flow         │
        └──────────────────────────────┘
```

---

## 3. Core Design Principle: Progressive Enhancement

### 3.1 The Central Idea

Bastion is built on **Progressive Security Enhancement**:

```
Each module provides value INDEPENDENTLY.
Combining modules ENHANCES security.
Full orchestration enables ADVANCED features.

Key property:
"Remove any module and the system still works,
 just with less security.
 Add any module and security increases."
```

### 3.2 Why This Matters

```
Traditional monolithic security:
- All-or-nothing
- One failure breaks everything
- Hard to adopt incrementally

Bastion's progressive approach:
- Each piece adds value
- Failures are isolated
- Incremental adoption
- Flexible deployment
```

### 3.3 The Litmus Test

Every module must pass this test:

```
"If I attach ONLY this module directly to an LLM,
 does it provide meaningful security?"

If YES → correct design
If NO → the module is too dependent
```

**Examples:**
```
Sentinel → LLM: ✅ Provides input validation
Vault → LLM: ✅ Provides PII protection
Navigator → LLM: ✅ Provides safe search
Anchor → LLM: ✅ Provides embedding protection
```

---

## 4. The Three-Layer Model

### 4.1 Layer Definitions

Every feature in Bastion belongs to one of three layers:

```
┌─────────────────────────────────────────────┐
│ Layer 3: ORCHESTRATION                      │
│ - Requires multiple modules + coordination  │
│ - Cross-cutting features                    │
│ - Optional (system works without)           │
│ - Examples: Honey-token, Lineage, Tenancy   │
├─────────────────────────────────────────────┤
│ Layer 2: COMPOSITION                        │
│ - Enhanced by combining 2+ modules          │
│ - Activated when modules are paired         │
│ - Optional enhancement                      │
│ - Examples: Indirect injection defense      │
├─────────────────────────────────────────────┤
│ Layer 1: STANDALONE                         │
│ - Each module's core function               │
│ - Works independently, no dependencies      │
│ - ALWAYS active                             │
│ - Examples: Injection defense, Anonymization│
└─────────────────────────────────────────────┘

Lower layers: always active
Upper layers: optionally added
```

### 4.2 Layer Characteristics

| Layer | Dependencies | When Active | If Removed |
|---|---|---|---|
| **Standalone** | None | Always | Module gone |
| **Composition** | 1-2 modules | When paired | Lose enhancement |
| **Orchestration** | Coordinator | When configured | Lose advanced feature |

### 4.3 Feature Classification

Each module's features are classified:

```
🟢 Core (Standalone)
   - Independent operation
   - No other module needed
   - Always functional

🟡 Enhanced (Composition)
   - Improved with other modules
   - Graceful without them
   - Optional synergy

🔴 Orchestrated (Cross-cutting)
   - Requires coordination
   - Hook-based integration
   - Optional advanced capability
```

---

## 5. Module Feature Classification

### 5.1 Sentinel

```
🟢 Core (Standalone):
   - Prompt injection defense
   - Metadata validation
   - Input/output sanitization

🟡 Enhanced (with Navigator):
   - Indirect injection defense
   - Search result validation

🔴 Orchestrated:
   - Honey-token detection (input/output)
   - Lineage event emission
```

### 5.2 Vault

```
🟢 Core (Standalone):
   - PII anonymization
   - Deterministic tokenization
   - Input/output masking

🟡 Enhanced (with Navigator):
   - Permission-based filtering
   - Category isolation

🔴 Orchestrated:
   - Honey-token creation/injection
   - Multi-tenancy full isolation
   - Lineage tracking
```

### 5.3 Navigator

```
🟢 Core (Standalone):
   - Hybrid search (vector + BM25)
   - Reranking

🟡 Enhanced (with Vault):
   - Permission-based filtering
   - Category partitioning

🔴 Orchestrated:
   - Honey-token search detection
   - Full tenant isolation
```

### 5.4 Anchor

```
🟢 Core (Standalone):
   - Embedding noise injection
   - Response analysis

🟡 Enhanced (with Navigator):
   - Search quality optimization

🔴 Orchestrated:
   - Pipeline-wide bias monitoring
```

### 5.5 Tracker

```
🟢 Core (Standalone):
   - Basic logging
   - Self-monitoring

🟡 Enhanced (with events):
   - Per-module metrics

🔴 Orchestrated:
   - Data lineage
   - Honey-token aggregation
   - Cross-module correlation
```

---

## 6. Independence Principles

### 6.1 Three Types of Independence

Bastion maintains three distinct types of independence:

```
1. Deployment Independence
   - Each module deploys separately
   - Independent containers
   - Independent scaling

2. Functional Independence
   - Each module owns its responsibility
   - Core features need no other module
   - Clear boundaries

3. Runtime Independence
   - One module's failure doesn't cascade
   - Graceful degradation
   - System continues with reduced capability
```

### 6.2 What Independence Does NOT Mean

```
Independence does NOT mean:
❌ Modules never communicate
❌ No shared concepts
❌ No cross-cutting features

Independence DOES mean:
✅ Core functions work alone
✅ Communication is loose (events)
✅ Failures are isolated
```

### 6.3 Loose Coupling Requirement

```
REQUIRED: No direct module-to-module dependencies.
Communication uses two channels depending on context:

  Synchronous (in-request data path):
  - gRPC interface calls between modules
  - Data is PASSED IN the request by the caller
  - No module holds a direct reference to another

  Asynchronous (cross-cutting, observability):
  - Events via NATS (fire-and-forget)
  - Used for hooks, lineage, monitoring, alerts

❌ FORBIDDEN: Direct dependency or hidden calls
   class Sentinel {
       vault: Vault  // NO! Direct dependency
   }
   class Navigator {
       search() {
           perms = vault.getPermissions()  // NO! Hidden direct call
       }
   }

✅ REQUIRED: Data passed in request (sync) or events (async)
   // Sync: caller gets Vault permissions, passes to Navigator
   perms = vaultClient.GetPermissions(userId)        // caller fetches
   navigatorClient.SearchWithPermissions(query, perms) // passed in

   // Async: cross-cutting hooks publish events
   class Sentinel {
       eventBus: EventBus
       detect() {
           this.eventBus.publish("honey_token_found")
           // Sentinel doesn't know or care who listens
       }
   }
```

---

## 7. Cross-Cutting Concern Handling

### 7.1 The Hook Pattern

Cross-cutting features integrate via **hooks**, not direct dependencies:

```
Module exposes hooks (extension points):

class Sentinel {
    hooks: Hook[]  // Empty by default
    
    // Core function (always works)
    validateInput(query) {
        result = this.checkInjection(query)
        
        // Hooks run IF present (optional)
        for (hook of this.hooks) {
            hook.onValidate(query, result)
        }
        
        return result  // Core result regardless
    }
}

Key: Core works whether hooks exist or not
```

### 7.2 Cross-Cutting Coordinator

```
Cross-cutting features have a Coordinator:

┌────────────────────────────────────┐
│  Honey-Token Coordinator           │
│  - Registers hooks in modules      │
│  - Aggregates events               │
│  - Manages feature lifecycle       │
└────────────────────────────────────┘
         ↓ registers hooks ↓
   [Sentinel][Vault][Navigator]
   (each still independent)

If Coordinator absent:
- Hooks not registered
- Core functions unaffected
- Feature simply inactive
```

### 7.3 Cross-Cutting Features

Identified cross-cutting features (Layer 3):

| Feature | Modules Involved | Coordinator |
|---|---|---|
| **Honey-token** | Vault, Sentinel, Navigator, Tracker | Vault-led |
| **Multi-tenancy** | Vault, Navigator, Sentinel | Shared |
| **Data Lineage** | All modules | Tracker-led |

---

## 8. Deployment Configurations

### 8.1 Progressive Deployment

Bastion supports incremental deployment:

```
Minimal (1 module):
[Sentinel] → LLM
- Input validation only

Basic (2 modules):
[Sentinel] → [Vault] → LLM
- Input validation + PII protection

Standard (3-4 modules):
[Sentinel] → [Vault] → [Navigator] → LLM
- + Safe search

Full (5 modules):
[Sentinel] → [Vault] → [Navigator] → [Anchor] → LLM
+ [Tracker] observing
- Complete security

Orchestrated (Full + Cross-cutting):
- All modules + Honey-token + Lineage + Tenancy
- Maximum security
```

### 8.2 Configuration Matrix

| Config | Modules | Security Level | Use Case |
|---|---|---|---|
| Minimal | Sentinel | Basic | Dev/test |
| Basic | A+B | PII-safe | Simple RAG |
| Standard | A+B+C | Search-safe | Production RAG |
| Full | A+B+C+E | Complete | Sensitive data |
| Orchestrated | All + CC | Maximum | Enterprise |

### 8.3 Pipeline Variations

```
Full Pipeline:
A → B → C → E → LLM (all security)

Lite Pipeline (public data):
A → C → E → LLM (Vault bypassed)

Minimal Pipeline (dev):
A → C → LLM (minimal security)

Each variation is valid and supported.
```

---

## 9. Design Constraints

### 9.1 Technical Constraints

```
Language: Go (all modules)
Communication: Event-driven (NATS)
Deployment: Docker/Kubernetes
Interfaces: gRPC, REST, CLI (all modules)
Standards: OpenTelemetry, W3C Trace Context
```

### 9.2 Architectural Constraints

```
1. Every module MUST work standalone
2. Inter-module communication MUST NOT use direct dependencies;
   synchronous data-path calls use gRPC (data passed in request),
   cross-cutting/observability uses events (NATS)
3. Cross-cutting MUST use hooks
4. Core functions MUST NOT depend on other modules
5. Failures MUST degrade gracefully
```

### 9.3 Quality Attributes

```
Priority order:
1. Independence (modules don't break each other)
2. Security (defense in depth)
3. Observability (everything traceable)
4. Performance (minimal overhead)
5. Flexibility (configurable deployment)
```

---

## 10. SRS Authoring Guidelines

### 10.1 Module SRS Structure

Every module SRS must include:

```
1. Core Functions (🟢)
   - Standalone capabilities
   - No dependencies
   - Full detail

2. Enhanced Functions (🟡)
   - Composition capabilities
   - Optional module pairings
   - Full detail

3. Hooks (🔴)
   - Cross-cutting extension points
   - BRIEF definition only
   - Reference cross-cutting SRS for detail

4. Standalone Operation
   - How to run independently
   - Mock dependencies

5. Interfaces
   - gRPC, REST, CLI
```

### 10.2 Cross-Cutting SRS Structure

Every cross-cutting SRS must include:

```
1. Feature Overview
2. Module Responsibilities
   - Which modules participate
   - What each contributes
3. Hook Usage
   - How each module's hooks are used
   - DETAILED here (not in module SRS)
4. Data Flow
   - End-to-end feature flow
5. Coordinator Design
   - How feature is orchestrated
```

### 10.3 Avoiding Duplication

```
RULE: State once, reference elsewhere

Module SRS:
"Exposes honey-token hook (see Honey-Token SRS)"
[3 lines]

Cross-Cutting SRS:
"Sentinel's honey-token hook works as follows..."
[detailed]

NEVER duplicate the detailed logic.
```

---

## 11. Glossary

| Term | Definition |
|---|---|
| **Standalone** | Module operating independently |
| **Composition** | Two+ modules working together |
| **Orchestration** | Cross-cutting coordination |
| **Hook** | Extension point for cross-cutting |
| **Coordinator** | Manager of a cross-cutting feature |
| **Core function** | Standalone capability |
| **Enhanced function** | Composition capability |
| **Orchestrated function** | Cross-cutting capability |
| **Loose coupling** | Event-based, indirect communication |
| **Graceful degradation** | Continuing with reduced capability |
| **Progressive enhancement** | Adding value incrementally |

---

## 12. Change History

| Version | Date | Changes |
|---|---|---|
| 1.0 | 2026-05-17 | Initial foundation document |

---

## Appendix A: Decision Records

### A.1 Why Progressive Enhancement?

```
Decision: Adopt progressive enhancement architecture

Rationale:
- Enables incremental adoption
- Isolates failures
- Each module independently valuable
- Flexible deployment

Alternatives considered:
- Monolithic (rejected: all-or-nothing)
- Microservices without standalone value (rejected: 
  modules useless alone)
```

### A.2 Why Event-Driven?

```
Decision: All inter-module communication via events

Rationale:
- Loose coupling
- Independence preserved
- Graceful degradation
- Observability

Alternatives considered:
- Direct calls (rejected: tight coupling)
- Shared database (rejected: hidden coupling)
```

### A.3 Why Three Layers?

```
Decision: Standalone / Composition / Orchestration

Rationale:
- Clear feature classification
- Maps to deployment options
- Maps to document structure
- Explains independence + cross-cutting

Alternatives considered:
- Two layers (rejected: insufficient nuance)
- Per-feature (rejected: too granular)
```

---

**End of Document**
