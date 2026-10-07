# Provenance — 0112-container

Every fact this story rests on, with the claim id it comes from and the
confidence that claim carries. Read at the submodule pin `54e154c`.

## Decoded, and built on

| Claim | Confidence | What this story takes |
|---|---|---|
| `ITEM-CONT-004` | High (shape, arithmetic) / **Medium** (nothing refuses) | The container is one class a unit and a sack both hold; `+0x1c` is the next-insert index, NOT a capacity; `+0x20` is the load; no slot count and no capacity exist anywhere. FR-1, FR-2, FR-3, D-4. |
| `ITEM-PICK-009` | High (routine, tick arm) / Unknown (the transition, since closed by `ITEM-PICK-016`) | A pick-up is all-or-nothing: gold to the owner's money, the sack's entire container poured over, the sack deleted. No capacity, distance, ownership or per-item selection. FR-6, FR-7, FR-8. |
| `ITEM-OWNED-028` | High (the arm, both destinations) / **Medium** (what the ids identify) | A type-8 record whose `+0x04` is non-zero stocks an existing actor and places no sack; its elements go to `actor+0x7c`. FR-11. |
| `ITEM-CODE-029` | High (masks, allocation sizes) / Unknown (ctor arguments) | Class is bits 8..11; a class outside 1..14 resolves to a null the map-load caller SKIPS. FR-4 takes the zero from this and nothing else. |
| `ITEM-DROP-008` | as published | Money is `Player+0x38` and is never an item. FR-7's purse, rather than gold in the container. |
| `AI-DIPLO-004`, `AI-DIPLO-085` | as published, via `relationSlots` | Fifty roster slots is a DECODED limit, already this package's constant. DD-3 indexes the purse by it. |

## Decoded, read, and deliberately NOT built on

| Claim | Why it is out |
|---|---|
| `ITEM-LOAD-005` (High) | The load's only consumer is the overload penalty, and half of `+0x20` is its input. `+0x20` needs a per-item weight this tree does not have. D-1. |
| `ITEM-STACK-003` (High / Unknown on `+0x44`) | Gives the load's sum and the stack split. Both need the same missing weight, and an authored element carries one code. D-1, D-2. |
| `HERO-EQUIP-017` | Establishes a WEAPON's weight (Weapons row slot 3) and no other class's. One class is not a column. D-1. |
| `ITEM-EQUIP-006` (High / Medium / Unknown) | Equipped is fourteen pointer fields, a different place from carried. `data.Equipment` already models the twelve slots; the container must not also. spec §Scope. |
| `ITEM-CMD-007` source 3 | The pick-up ORDER — the command, the walk, the destination slot. The later story's. |
| `ITEM-CARRY-015` | A hero's items crossing a mission boundary. Nothing here persists across missions. |
| `ITEM-SPAWN-026`, `ALM-SACK-065` | The type-8 section as authored loot, already consumed by 0103. |

## Ours by choice, not decoded

- The **carry section's and purse section's layout and their place in the byte
  form**. The original's save format is not what this form is; version 26's
  shape follows this package's own route and relation sections.
- **Version 26** itself: an allocation of this project's sequence.
- **Dropping a zero code in the constructor and refusing it in the decoder.**
  The DROP reproduces the original loader's skip; the REFUSAL is this package's
  own rule about what a decoded record may claim, and has no counterpart in the
  original, which has no such form.
- **The purse's 32-bit wrap.** The sack merge already wraps at that width (0103
  D-8); this follows it rather than inventing a saturation.
- **The eight pack cells** — 0110's authored count, unchanged here, and D-3 says
  plainly that it is the window's limit and not the container's.
- **The `G` key.** No claim names a keyboard.

## Open, and named as open

- **The id space of `actor+0x08`.** `ITEM-OWNED-028` grades it Medium and says
  nothing reads what writes it. This story joins on `alm.Unit.UnitID` because
  all 43 stock records on both roots match exactly one unit under it, with none
  unmatched and none ambiguous — a corpus bijection, not a decode. FR-12 keeps
  the cost of being wrong at "a stock record places nothing".
- **Whether anything refuses an add.** `ITEM-CONT-004`'s named blind spot is a
  caller that tests the load before calling. FR-3 is unbounded by disclosure.
- **The weight column for a non-weapon.** Nothing in the pin establishes it.
  Until it does, the container has no load and the overload penalty cannot be
  built — `pkg/data/hero.go`'s `Speed()` doc block says the gate is never
  satisfied because nothing carries anything, and that reason expires here even
  though the term still cannot be written.
- **Element `+0x04`.** Five of the corpus's 25 stock elements carry a non-zero
  one, which routes through the actor's own vtable rather than the plain append
  (`ITEM-OWNED-028`). What that arm does is not decoded; this build appends all
  25 alike and the divergence is recorded rather than guessed at.
