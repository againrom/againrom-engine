# 1000-v2-baseline — contract

Preparatory story. Owner directive, 2026-08-15: pipeline v2 is in force from this date; a
preparatory story numbered 1000 takes ownership of the game-level documents the transition
introduced. Meta-level process changes (checker retirement, workflow text, doc-set conventions)
are excluded: a spec is about the game, and those changes are about how we work.

## Result

Three documents on `master`, all game-level:

1. `docs/DOMAINS.md` — the game's nine vertical domains, their ownership and coupling rules
   (introduced in commit `9ab5a86`).
2. `docs/DIVERGENCES.md` — the single typed divergence ledger, nine rows at introduction
   (introduced in commit `26c231e`).
3. `docs/1000-v2-baseline/spec.md` — the top-level description of the game as built at the v2
   epoch, per domain. Later contracts name domains against `DOMAINS.md` and record gaps against
   `DIVERGENCES.md`; this spec is the baseline both are read against.

## Story numbering

1000 opens the v2 numbering (owner, 2026-08-15): every new story continues from it — 1001, 1002,
and so on. Numbers below 1000 are the pre-v2 era and are never reused. A new round of an existing
story keeps that story's number (the shop's next round stays `0157-r3`, per the owner's «land все
в 0157»).

## Research claims

None consumed. This story asserts nothing about ROM1 behaviour. The divergence ledger's rows
cite their own claims.

## Aspects

All twelve aspects are N/A: no behaviour changes. The closure records the completeness check
that replaces them for a documentation story.

## Domains touched

All nine, by naming them. No domain's code is touched.

## Out of scope

- Process and checker changes (`AGENTS.md`, `scripts/`, `SDD/`): meta-level, owned by the
  pipeline's own records above this repo.
- Any change to game behaviour, tests, or serialized state.
