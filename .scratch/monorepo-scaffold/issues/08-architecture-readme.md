# 08: architecture.md + README

**What to build:** The architecture write-up capturing the settled decisions (event sourcing, parity
contract, two-layer domain model, planned SQLite tables) and a README that gets a new contributor from
clone to green tests.

**Blocked by:** 04 — Full reducer + 24-fixture parity.

**Status:** resolved

- [x] `docs/architecture.md` reproduces the canonical `PointDerivedState` shape, fixture format, and equality rules
- [x] The planned SQLite tables reflect the settled model (seasons, two rosters, slugs)
- [x] The seven spec-gap defaults are documented
- [x] `README.md`'s command table matches the root `package.json` scripts

## Comments

Resolved by branch `ticket/08-architecture-readme` (cbd47bf), merged as 491e151.
