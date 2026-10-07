# Spec — how far a unit sees is its own, and the view opens on fifteen columns

**Intensity:** spec-anchored / static. **Terrain:** brownfield.

## Why

Two numbers this tree wrote for itself are replaced by numbers the original carries. The two are
unrelated in code and share only that act, so they are stated as two groups of requirements and
nothing here couples them.

**The sight range.** The sight march is seeded with a range in whole cells, and this build seeds
every march with the same 5 — the value the definition tier's constructor default happens to hold.
Both shipped bands of the definition table carry that range as a column, and it is already decoded:
across the two bands the shipped rows take values from 4 to 12, so one constant is wrong for most
units on most maps. It is not a cosmetic wrongness. The range is the whole budget of a walk whose
reach also scales with the observer's altitude, so a unit given 5 where its row says 8 sees a
region several times too small from high ground, and a group's engagement decision reads exactly
that region.

**The opening view.** The view a mission opens on spans a fixed number of map columns. The count
this build uses was authored, and was disclosed as an upper bound rather than a measurement,
because the original's screen carried a panel whose geometry was undecoded. It is decoded now: the
map view's rectangle is the screen minus a 160-pixel right strip, and the shipped default screen is
640x480, which yields 15 columns and not 20.

## Requirements

### A unit's own sight range

- **FR-1** An entity carries its own sight range, in whole cells, as ONE BYTE. Every value the byte
  can hold is a legal range; none is refused, folded or clamped anywhere.
- **FR-2** The sight march is seeded from the observer's own range. Its budget is
  `(1 << (k-1)) + (range << k)` for the predicate's fixed-point shift `k`, exactly as before —
  what changes is where the range comes from and nothing about the walk.
- **FR-3** A group's visibility is the union of one march per living member, each at THAT member's
  own range, into one shared stamp. A candidate lit by any member is a candidate for every member.
- **FR-4** A guarding group's notice radius is the maximum, over its members, of that member's
  Chebyshev distance from the group centroid PLUS THAT MEMBER'S OWN RANGE; the maximum is taken over
  values already narrowed to a byte, then floored by the guard-range floor and widened by the arm's
  margin. A group of one member sees that member's own range added to zero distance.

### Where a range comes from

The definition tier carries a separate field named `Sight`, defaulting to 0, with no column and no
reader anywhere. It is **not** this range and nothing here reads it; the range is the scan-range
column, and the two must not be conflated by a later rename.

- **FR-5** A placement that resolves to a definition carries that definition's scan-range column:
  the creature band's slot for a creature, the person band's for a person. A cell the row leaves
  empty keeps the constructor's own value, which is what the definition tier already does with
  every other column.
- **FR-6** A placement that resolves to no definition at all carries the constructor's own range, 5.
- **FR-7** A generated hero's range is derived from his own statistics and not from any table:
  `4 + (Mind + Reaction) / 25`, in integer division, with both statistics capped first — the same
  cap, applied at the same point, as every other number derived from a hero.
- **FR-8** A definition column outside the range of a byte reaches an entity TRUNCATED to its low
  byte, never clamped and never refused. A negative or oversized column is therefore a value and
  not an error.

### The canonical form

- **FR-9** The sight range is canonical simulation state: it is carried by the byte form and it
  enters the digest.
- **FR-10** The byte form is at version 18. The range is one byte at the END of the entity record,
  so every offset the previous version fixed is unmoved and only the record's width changes. Every
  other version byte is refused, version 17 included — that version is a real one this tree wrote,
  in the concurrent story that landed while this one was open, and it is refused on the same terms
  as any other: there is no migration path.
- **FR-11** A decode carries the byte whole. There is no value the encoder can write that the
  decoder will not read back, and no two byte forms decode to one world.

### The opening view

- **FR-12** The view a mission opens on spans **15** map columns. The figure is what the decoded
  viewport rectangle yields at the shipped default screen resolution; it is no longer authored and
  carries no AUTHORED verdict.
- **FR-13** The view opens centred on the cell it is armed with, at the span FR-12 states. Nothing
  about the centring, the arming, or the one-shot application changes.

## Acceptance criteria

- **AC-1** An entity built naming a range carries it; one built naming none carries zero, and zero
  is a range and not an absence — such a unit's march lights its own cell and nothing else.
- **AC-2** Two units at the same cell on the same ground, with different ranges, light different
  regions, and the larger range's region contains the smaller's.
- **AC-3** A group of two members with different ranges lights the union of their two regions, and
  that union is a strict superset of either member's alone when the two stand apart.
- **AC-4** A group's notice radius follows the member whose distance plus range is greatest, not
  the member that is furthest out and not the member with the widest range, when those are
  different members.
- **AC-5** A creature placement whose row states a range carries that range; one whose row leaves
  the cell empty carries 5; a placement resolving to nothing carries 5.
- **AC-6** A person placement whose row states a range carries that range.
- **AC-7** A hero of capped statistics carries `4 + (Mind + Reaction)/25`; a hero of zero
  statistics carries 4; statistics above the cap give the capped answer.
- **AC-8** A column of 300 reaches an entity as 44 and a column of -2 as 254.
- **AC-9** The entity record is one byte wider than the version this story's branch was merged
  onto, the form this build writes opens at the version this build defines, and a buffer at any
  other version is refused with its version named. **No test states the version as a literal**: the
  number is a project-wide allocation that any lane may move, so a literal would hold when a story
  added a field and forgot to bump and would fail when another lane legitimately took the next one.
- **AC-10** Two worlds differing only in one entity's range marshal to different bytes and hash to
  different digests.
- **AC-11** A world marshalled and decoded again holds the ranges it held, over every byte value
  including 0 and 255.
- **AC-12** The pinned world's bytes and digest are those of a hand transcription that carries the
  new byte, and the transcription with that byte removed from every record is the previous
  version's pinned form.
- **AC-13** The opening view spans 15 columns, and the figure a test compares against is derived
  from the decoded viewport rectangle and the shipped default resolution — not read back off the
  constant it is pinning.
- **AC-14** A viewer armed with a cell opens with that cell centred and 15 columns across the view.

## Properties

- **P-1** *Invariant.* The march's region for range `r` is contained in its region for range `r+1`
  on the same ground from the same cell.
- **P-2** *Invariant.* A group's stamp contains every member's own stamp.
- **P-3** *Completeness.* Every entity reaching a world through the map loader has a range from
  exactly one source: its band's column, or the constructor's default. There is no path that leaves
  a placed unit's range unset.
- **P-4** *Idempotence.* Marshalling a decoded world reproduces the bytes it was decoded from.
- **P-5** *Negative invariant.* No byte value is refused on decode, and no range is normalised at
  construction — the constructor and the decoder accept exactly the same set of ranges.

## Divergence from the published law

- **D-1** *The second writer of the sight byte is not modelled.* Beside the constructor's default
  and the two streamers, one arm of the PER-ACTOR state machine overwrites it. That machine is not
  implemented in this build at all — it governs about one creature in a hundred and fifty of the
  shipped corpus — so the arm is out of reach rather than skipped, and the byte here is never
  written after a unit is placed.
- **D-2** *A hero's sight is carried in whole cells only.* The original recomputes it as a
  sixteen-bit value in sub-cell units, whose high byte is the whole-cell radius; the low byte is a
  remainder no decoded consumer reads. This build carries the whole-cell byte and drops the
  remainder.
- **D-3** *Whether a placed person's streamed range survives spawn is not decided.* The original's
  recompute, if it runs for such a unit, would replace the streamed column with the derivation FR-7
  states. A placed person here keeps its row's column and a generated hero takes the derivation, so
  no path in this build depends on the answer.
- **D-4** *The notice radius is still recomputed rather than frozen.* The original computes it on
  every guard tick and then uses the value frozen when guard was last issued. Freezing needs a group
  record and a guard setter, and this build has neither, so FR-4 states what the original computes
  and discards. Unchanged by this story, and restated because FR-4 changes what that computation
  reads.
- **D-5** *The number of view ROWS is not fixed.* The decoded viewport is a column span AND a row
  span per resolution, and an open side panel recomputes the row span at run time. This build fixes
  the column span alone and lets the rows follow the player's window.
- **D-6** *The persisted view origin is not modelled.* Mission entry also restores a saved view
  position, on a path that runs beside the centring FR-13 keeps; which of the two lands last is not
  established. This build has only the centring.

## Out of scope

- The per-actor state machine, and therefore its radius-overwriting arm (D-1).
- Any change to the march itself: the window tables, the recurrence, the ring order, the inset test,
  the unpruned blocker and the signed height plane are all as they stand.
- Any change to how a definition column is decoded. The scan-range column is already decoded on both
  bands; nothing here re-reads the table.
- The reach, which is the other number the definition tier carries with no column, and the sub-cell
  remainder of D-2.
- The view's row count, its persisted origin, and the resolution selection that would let a consumer
  ask for a span other than 15 (D-5, D-6).

## Traceability

| Requirement | Criteria | Properties |
|---|---|---|
| FR-1 | AC-1, AC-11 | P-5 |
| FR-2 | AC-1, AC-2 | P-1 |
| FR-3 | AC-3 | P-2 |
| FR-4 | AC-4 | — |
| FR-5 | AC-5, AC-6 | P-3 |
| FR-6 | AC-5 | P-3 |
| FR-7 | AC-7 | — |
| FR-8 | AC-8 | — |
| FR-9 | AC-10 | — |
| FR-10 | AC-9, AC-12 | — |
| FR-11 | AC-11, AC-12 | P-4, P-5 |
| FR-12 | AC-13 | — |
| FR-13 | AC-14 | — |
