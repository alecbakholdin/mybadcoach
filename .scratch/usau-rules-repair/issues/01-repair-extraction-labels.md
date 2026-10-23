# 01: Repair stranded extraction labels

**What to build:** Fix the rule labels the txt extraction stranded at page breaks (17.C.2–17.C.5,
3.J./3.I.3., 7.D./7.C.7.d.), verified against the official PDF.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

- [ ] Every stranded label sits with its rule text in `docs/reference/usau-rules-2026-27.md`
- [ ] Fixes verified against `docs/reference/usau-rules-2026-27.pdf` (page references noted in a comment)
- [ ] `npm run verify:rules` passes with new assertions covering the repaired rules
