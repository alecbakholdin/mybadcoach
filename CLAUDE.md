# My Bad, Coach

Real-time, collaborative, offline-first Ultimate Frisbee live stat tracker. See `GLOSSARY.md` for domain vocabulary.

## Reference

- USAU rules: `docs/reference/usau-rules-2026-27.pdf` (official PDF) and `docs/reference/usau-rules-2026-27.txt` (text extraction, for grep/reading).

## Agent skills

### Issue tracker

Issues and specs live as markdown files under `.scratch/<feature-slug>/`. See `docs/agents/issue-tracker.md`.

### Triage labels

Default label vocabulary: `needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`, recorded as `Status:` lines in issues. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: one `GLOSSARY.md` + `docs/adr/` at the repo root. See `docs/agents/domain.md`.
