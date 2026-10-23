# Spec: Repair USAU rules extraction

## Goal

Repair the reading-order defects in `docs/reference/usau-rules-2026-27.txt` (and the generated
markdown) where rule labels are stranded away from their rule text at page breaks — e.g. the
`17.C.2`–`17.C.5` labels interleaved into the 17.C.1 continuation, and similar patterns at
`3.J.`/`3.I.3.` and `7.D.`/`7.C.7.d.`. All fixes verified against the official PDF at
`docs/reference/usau-rules-2026-27.pdf`.

## Scope

- Fix the stranded labels in the txt and/or the markdown so every rule label sits with its text.
- Extend `scripts/verify-usau-rules-md.sh` with assertions covering the repaired rules.

Out of scope: any gameplay code changes.
