# Scoped Exceptions and User Notices

**Document Type:** Cross-module design
**Status:** Implemented for Tracker, Sentinel and Vault (first version)

## 1. User notices

When Sentinel or Vault blocks, masks or flags something, the response now carries a
`user_notice` that is safe to show the end user. The detailed reasons (rule IDs,
scores, matched values) stay in the response checks and the Tracker log.

```json
{
  "action": "masked",
  "code": "PII_REDACTED",
  "message": "Some personal information in this answer was hidden.",
  "reference_id": "req-123",
  "count": 2
}
```

| Code | Meaning |
|---|---|
| `INJECTION_SUSPECTED` | input looked like an attempt to override instructions |
| `METADATA_INVALID` | required request metadata missing or invalid |
| `PII_REDACTED` | personal data masked or withheld |
| `PERMISSION_DENIED` | answer needs a higher access level |
| `CONTENT_POLICY` | content policy violation |
| `NOT_GROUNDED` | answer could not be verified against sources |
| `FORMAT_INVALID` | answer not in a valid format |
| `REQUEST_BLOCKED` / `RESPONSE_MODIFIED` | generic codes used at `minimal` detail |

Rules:
- A notice names the **category**, never the rule, the matched text or the score, so it
  does not help an attacker tune around a detector.
- `count` is how many items were masked. Values are never included.
- `reference_id` is the request or audit ID so support can find the Tracker record.
- `user_notice.detail` (Sentinel and Vault config) is `category` (default) or `minimal`
  (generic message and code only, no count).

Where it appears: Sentinel REST `POST /v1/validate/input` (`user_notice`) and
`POST /v1/validate/output` (`Notice`); Vault anonymize responses (`user_notice`).

## 2. Scoped exceptions

An exception exempts **one rule** for **one tenant** (optionally one user) from
enforcement by named modules, until it expires or is revoked.

Fields: `rule`, `modules`, `tenant_id` (required), `user_id` (optional),
`justification` (required, 10+ chars), `expires_at` (required, max 90 days).
Wildcard rules are rejected, and a rule's module prefix must be in `modules`.

### Rule identifiers

| Rule | Exempts |
|---|---|
| `sentinel.in:<pattern_id>` | an input injection pattern (e.g. `pi-001`) |
| `sentinel.out.pii:<rule_id>` | an output PII rule, by its stable rule ID |
| `sentinel.out.permission` | the access-level boundary check |
| `sentinel.out.content:<violation>` | a content-filter violation (only if all violations are exempt) |
| `sentinel.out.grounding` | the hallucination / grounding check |
| `sentinel.industry:<filter_id>` | an industry-filter block |
| `vault.mask:<pii_type>` | masking of a PII type, e.g. `vault.mask:email` |
| `vault.view:<pii_type>` | transformation of a PII type in the output view (anonymized level only) |
| `navigator.purpose:<purpose>` | the purpose pre-filter, for one declared purpose |
| `anchor.bias:<category>` | one bias category (`gender`, `ethnicity`, `age`) in output verification |

Deliberately not exemptable: tenant isolation, Vault permission filtering, the Sentinel format
check, and the aggregated / audit views. Tracker refuses to create exceptions for them.

Navigator and Anchor use a Python copy of the client (`navigator/navigator/exceptions.py`,
`anchor/anchor/exceptions.py`, kept identical by the sync test). It is verified against an
envelope signed by the Go implementation.

### Flow

```
admin ──POST /v1/exceptions──▶ Tracker ──signs (Ed25519)──▶ NATS bastion.exceptions.updates
                                  │                              │
                 audit event      │ snapshot (request/reply)     ▼
                                  └─────────────────────▶ Sentinel / Vault
                                                          verify → persist → enforce
```

1. Only an **admin** can create or revoke, in the Tracker UI (**Exceptions** page) or via `/v1/exceptions`. Each change is written to the
   Tracker audit log as `exception_created` / `exception_revoked`.
2. Tracker signs every change with its private key. Modules hold only the **public** key,
   so a module (or anything on the bus) cannot forge an exception.
3. Modules verify the signature, merge by per-exception version, and persist the **signed
   envelope** to a local file. A tampered file is rejected on load.
4. When a module starts or reconnects it requests a snapshot from Tracker, so it catches
   up on anything it missed.
5. When an exception is actually used, the module reports it (`exceptions_applied`, and for
   Sentinel an `exception_applied` event) so the audit trail shows real use.

### Modules work independently

- Enforcement reads only the module's **local** store. If Tracker or NATS is down, the
  module keeps its last known exceptions; nothing blocks startup.
- With **no** public key configured the feature is off and every rule is enforced.
- With a key but no exceptions, every rule is enforced.
- Expiry is judged by each module itself; it does not trust Tracker for that.
- **No NATS at all:** export a signed snapshot (`GET /v1/exceptions/export`) and save it as
  the module's exceptions file. The file is polled every 30s.
- Revocations are ordered by version, so replaying an old message cannot revive a revoked
  exception. Sentinel flushes its result cache on any change so a revocation applies at once.

### Setup

```bash
tracker-cli exceptions keygen          # prints a private and a public key

# Tracker (secret)
TRACKER_EXCEPTIONS_SIGNING_KEY=<private>   # config: exceptions.signing_key / exceptions.file

# Sentinel / Vault (public)
SENTINEL_EXCEPTIONS_PUBLIC_KEY=<public>    # config: exceptions.public_key / exceptions.file
VAULT_EXCEPTIONS_PUBLIC_KEY=<public>
NAVIGATOR_EXCEPTIONS_PUBLIC_KEY=<public>
ANCHOR_EXCEPTIONS_PUBLIC_KEY=<public>
```

### Key rotation

Modules accept several comma-separated public keys. To rotate: give modules `new,old`; set
Tracker's `exceptions.signing_key` to the new key and `exceptions.previous_public_keys` to the old
public key. Tracker loads state saved under the old key and re-signs it with the new one. Once
everything is re-signed, drop the old key from the modules.

### Tracker state

Tracker's own state file is a signed envelope too. If its signature does not verify, Tracker
refuses to start (`ErrStateTampered`) rather than trusting an edited file.

## 3. Testing

- Unit tests cover signing, forgery, tampering, replay, scope, expiry and standalone restart.
- `tracker/internal/exceptions/e2e_test.go` runs against a real NATS server (live push, late-join
  snapshot, revocation, forged bus messages, standalone restart). It runs in CI (`exceptions-e2e`
  job, NATS service container) and locally with `EXCEPTIONS_E2E_NATS=localhost:4222`.
- `tests/exceptions_sync_test.go` fails if the shared `types.go` / `client.go` drift between
  tracker, sentinel and vault, or `exceptions.py` between navigator and anchor.
- `navigator/tests/test_exceptions.py` also runs a live NATS test with `EXCEPTIONS_E2E_NATS=host:port`.

## 4. Known limitations

- Sentinel's gRPC notice is sent as trailing metadata (`x-bastion-user-notice-bin`) on unary
  `Validate` only. Streaming and batch calls cannot carry a per-message trailer, and the proto is
  unchanged (no `protoc` in this environment).
- The Python live-NATS test is not yet a CI job (the Go one is).
- The Docker images have not been built or tested locally.
