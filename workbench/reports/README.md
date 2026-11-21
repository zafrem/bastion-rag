# Reports

`baseline.md` / `baseline.json` score the built-in dataset (`../data/cases.jsonl`) against the
modules as they shipped, so later changes (fixes, new profiles, tuned thresholds) can be compared
with a fixed starting point.

Conditions of the baseline (2026-11-21):

- Sentinel: `config.Default()` (no config file), no scoped exceptions
- Anchor: repository `config/config.yaml` defaults, no scoped exceptions
- Profiles: `internal-staff` and `external-customer`, model answers fixed by the dataset
- Run with: `workbench score -profiles internal-staff,external-customer -out reports`

Timings are from a laptop with everything on localhost and say little about production latency;
the findings (what was blocked, rewritten or missed) are the useful part. To compare a new run
against this one, run the same command into another directory and diff the two `report.md` files.
