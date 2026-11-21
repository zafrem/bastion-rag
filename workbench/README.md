# Bastion Workbench

A demo UI and test bench that sends a prompt through the Bastion pipeline and shows
what each stage did. It lives in the superproject, outside every module, and talks to
them over HTTP only: nothing here imports module code, and any service whose URL is
empty is skipped.

```
prompt ─► Sentinel (input) ─► context ─► model (Ollama) ─► Sentinel (output) ─► Anchor ─► user
```

## Run it

Start the services you want, then the workbench. For local development:

```bash
# Sentinel (default config, REST on :8080)
cd sentinel && go run ./cmd/sentinel server

# Anchor (REST on :8083). Dev only: needs a token, or allow_unauthenticated: true in the config.
cd anchor && python -m anchor.main --config config/config.yaml

# Ollama with a model
ollama pull qwen3.5:4b

# the workbench
make workbench            # http://127.0.0.1:8090
```

| Variable | Default |
|---|---|
| `WORKBENCH_ADDR` | `127.0.0.1:8090` (keep it on loopback) |
| `SENTINEL_URL` / `SENTINEL_TOKEN` | `http://localhost:8080` |
| `ANCHOR_URL` / `ANCHOR_TOKEN` | `http://localhost:8083` |
| `OLLAMA_URL` / `OLLAMA_MODEL` | `http://localhost:11434` / `qwen3.5:4b` |
| `OLLAMA_MAX_TOKENS` | `300` |
| `WORKBENCH_PROFILES` | optional JSON file of profiles |

Set a URL to empty to leave that stage out.

## What the UI does

- **Run**: one prompt under one caller profile, with every stage laid out. Click a stage for its
  detail and the notice the end user would see. The two answers are diffed, so you see exactly which
  words a control changed. *Observe mode* keeps going after a block to show what would have been
  prevented.
- **Compare profiles**: the same prompt under 2 to 4 profiles with **one shared model answer**, so a
  difference comes from the controls and not from the model's randomness. The first stage where the
  profiles diverge is outlined.
- **Suite**: eight built-in scenarios (harmless answers, injection, PII, bias, ungrounded) across the
  chosen profiles, as a matrix. Fixed answers make it fast and repeatable; the live-model switch uses
  the real model.
- **Profiles**: edit the callers (tenant, user, access level, metadata quality).

Notes on behaviour: if Sentinel is configured but fails, nothing is delivered (fail closed). Anchor is
advisory, so its failure is shown but the answer still goes out. Reasoning models such as `qwen3.5`
are called with `think: false`; otherwise they spend the whole token budget on hidden reasoning and
return an empty answer.

## Tests

```bash
make test-workbench                     # fake Sentinel, Anchor and Ollama; runs anywhere
cd workbench
WORKBENCH_LIVE=1 go test -v ./...                      # real Sentinel and Anchor (canned model answers)
WORKBENCH_LIVE=1 WORKBENCH_LIVE_LLM=1 go test -v ./... # also the real model
WORKBENCH_FAIL_ON_KNOWN=1 ...                          # turn documented known issues into failures
```

## Known issues it surfaced

- Sentinel output rules `latitude_01` and `longitude_01` redact ordinary integers, so
  "within 14 days" is delivered as "within [DECIMAL LATITUDE …] days". Harmless answers differ from
  the raw model for every profile.
- Sentinel-IN rejects metadata with a user id under 3 characters, a non-UUID-v4 context id, or a
  timestamp outside ±1 hour. A careless client is blocked regardless of the prompt.
- With the default configuration, the internal and external profiles behave identically: the
  controls are global, so the only difference between callers is metadata quality.
