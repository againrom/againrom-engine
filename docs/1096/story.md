# Original saved spellbooks

## Result

An original SAV supplies each retained player character's learned spell
membership. Loading must not substitute the Humans template, add equipment
teaching effects again, or share one character's book with another. The same
membership reaches native SAVE/load, mission carry, spell rows and casting.

## Authority and contract

`MAGIC-SPELL-001` identifies the serialized Spell ID, range, defensive flag
and mana cost. `SAV-SPELL-044` supplies its nine-byte shape.
`MAGIC-BOOK-002` and `SAV-SPELLBK-041` identify the sparse ID-indexed book and
the omitted slot zero. `SAV-HERO-059` and `SAV-GRPORD-058` keep ownership and
archive order separate.

The reader retains absent/empty books, sparse null slots, shared references
and all four named Spell values. Invalid IDs, wrong reference classes and
slot/ID disagreement reject the original load before session publication.
Membership uses the existing deterministic KnownSpells mask. Older native
saves retain their established defaults; no simulation byte-form change is
planned.

Later book use and newly equipped town items can still teach spells. Only the
new item contributes; loading and viewing do not replay the worn inventory.
An old native snapshot is not rewritten from provenance. Original export
refuses a native membership that differs from the source graph even when it
still equals an old template-derived baseline.

Original-city provenance version1 validates baselines with the exact previous
template-book import policy; version2 validates source-book baselines. Neither
policy overwrites current native party progress. Forging other baseline fields
still rejects LOAD transactionally. Both versions remain native-saveable.

## Proof

Hand-built archive records independently exercise sparse books, aliases,
empty books and malformed input. Production restoration, native round-trip,
mission carry and spell-row/use tests exercise the player route. Install-gated
EN/RU witnesses compare literal source field expectations, not the production
reader against itself. Final measurements are recorded in verification.md.

## Open debt

Per-character saved range, defensive and mana-cost overrides are decoded and
counted but are not modeled by the shared spell-rule table. Non-party books,
pending casts and a general world SAV writer remain separate work. This story
does not establish original-game acceptance of generated SAV files.

`DIV-650` retains instance parameters. `DIV-651` retains the unmodeled
container-presence/learning gate. `DIV-652` retains legacy native import loss
and its explicit export refusal rather than changing existing player progress.
