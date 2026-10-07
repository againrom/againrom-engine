# 1105 — saved non-party spellbooks

Original mission LOAD restores the saved books of uniquely matched living
non-party actors from every Player, including temporary first-Player allies.
Absent, present-empty and sparse books remain distinct. Saved range, raw
Defensive and mana-cost words reach new book casts and native continuation.

## Contract and authority

- FR-1: Project Unit, Humanoid and Human books through the complete shared
  archive walk. Validate class, slot/ID, presence, count and truncation before
  publishing values. Null and repeated actor references add no actor copies.
- FR-2: Both mission LOAD doors join unique living, on-map, nonzero MapUnitIDs.
  Exclude persistent party and unsupported sources. Skip missing targets;
  reject ambiguous eligible identities before changing any book.
- FR-3: Install only KnownSpells and Book after stock/Rearm, without refreshing
  saved parameters. New book casts/AI and ordinary SAVE/fresh LOAD retain them.
- FR-4: Failed App LOAD preserves the previous Snapshot/hash. Reports name
  restored and skipped books without claiming incoming casts were restored.

`SAV-SPELLBK-041`, `SAV-SPELL-044`, `MAGIC-SPELL-001` and `MAGIC-BOOK-002`
establish Unit-owned sparse books and persistent fields. `MAGIC-CAST-003` and
`MAGIC-AI-012` establish signed cost and raw Defensive consumers. Amended
`SAV-DOC-053`, amended `SAV-DEATH-051`, `SAV-ROSTER-024` and `SAV-ID-015`
bound traversal, liveness and the authored map identity.

## Design

DD-1: A narrow SAV projection does not depend on Character, equipment or
first-Player party provenance. Shared Spell references yield detached values.

DD-2: A sim batch validates every resolved target and book before applying any
entry. The game layer owns the MapUnitID join and LOAD publication boundary.

DD-3: Existing form71 already persists all entities' book state. No format bump
or table refresh is required. Weapon, scroll, innate and script sources retain
their existing parameters.

## Exclusions

No missing/dynamic NPC creation, fallback name/definition/cell join, incoming
ROM1 order/cast/effect lifecycle, general world SAV writer, first-book
allocation (DIV-690), mutable shared-Spell alias propagation (DIV-691), or
complete ROM1 AI-context claim. The research pin remains unchanged.

Verification is recorded in `verification.md`.
