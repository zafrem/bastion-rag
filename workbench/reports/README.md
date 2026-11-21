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

## After honouring `report` rules (`after-report-action.md`)

Same dataset, profiles and services, with Sentinel now honouring each pattern's declared
`action_on_match` (see `sentinel` config `honor_report_action`) and with explicit separated-card patterns.

| Measure | Baseline | After |
|---|---|---|
| Pass rate | 49% (54/110) | **75%** (82/110) |
| Harmless answers rewritten | 28 of 38 | **0 of 38** |
| False-positive rate | 78% | **17%** |
| Precision | 53% | **83%** |
| Leaks | 0 of 22 | **0 of 22** |
| Recall / missed attacks | 69% / 10 of 24 | 69% / 10 of 24 (unchanged) |

What remains is detection coverage, not false positives: encoded and indirect injections, the bias word
lists (age, ethnicity, and a neutral mention of gender flagged), the content filter, grounding of
unrelated answers, harmless prompts that merely discuss prompt injection, and the 10-character minimum
refusing a correct one-word answer.
