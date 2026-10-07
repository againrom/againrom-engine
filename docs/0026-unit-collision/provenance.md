# Provenance — deterministic unit cell occupancy

Pinned at research `e7602bb`, frozen for the story; `claims/retracted.md` and the registry's standing
corrections were read first, and nothing in either bears here.

## Backing

**No row.** Not one anchor of this contract is derived from a research claim. The search that
establishes it is on the record: `claims/terrain.md` and `claims/alm.md` hold the original's movement
machinery in detail, and everywhere it touches this story it *differs*. Those claims appear below as
divergences from a choice of ours, never as backing for one.

## Ours by choice

| What the spec fixes | What the evidence says |
|---|---|
| **At most one unit per cell, for every unit alike** (FR-1, P-1, P-5) | The original does not have this rule. `TERR-PASS-051` (High) makes the block byte a **bitmask**, with **bit 6 a ground occupant and bit 7 an air occupant**, set and cleared only on the dynamic plane, and a mover blocked iff `block[cell] & mover.mask`; the shipped masks are `0x41` ground, `0x44`, `0x82` air (`TERR-MOVE-057`, High — `Data.bin` streams `movementType` 2 on Ghost/Bee and 3 on Bat_Sonic/Dragon). So there a ground unit and an air unit share a cell freely. One occupancy class for all units is our simplification, with the divergence known |
| **A unit occupies exactly one cell** (FR-1, C-3) | `TERR-PASS-051` tests the mask over an `n x n` footprint, `TERR-MOVE-055` (High) puts `n` in a per-instance byte, and `TERR-MOVE-057` has the definition table storing footprint 2 on 11 entries and 3 on 4. Multi-cell footprints are live on shipped data, so 1x1 is a choice and not an absence |
| **All-or-nothing on the desired cell** (FR-2, AC-8) | Nothing decoded says what the original does with a blocked mover. **(A)** slide along whichever axis is free — cheap, and it turns a diagonal order into a wall-hugging drift no criterion can pin. **(B, chosen)** refuse the whole step: one outcome per unit per tick (P-4), and routing left to the story that owns it |
| **Ascending `EntityID` decides contention** (FR-3, AC-2) | 0019 already recorded the advancement order as ours: that the original advances its movers in any particular order is unknown, and it still is. **Answered since; appended 2026-07-31: `MOVE-TICK-009/013/014` and `MOVE-ID-016` at High, so ours is a divergence — argued in 0019.** **(A)** a draw from the world's own generator — fair-looking, but it consumes state nothing in the engine consumes yet and makes a contested outcome unreadable from the state alone. **(B)** neither unit moves — symmetric, and it deadlocks every convoy. **(C, chosen)** lowest id takes the cell, which is the one order already required to be observable |
| **A cell vacated earlier in a tick is available later in it** (FR-3, AC-4) | Undecoded. The original's dynamic plane *is* mutated by four named routines as occupants come and go (`TERR-PASS-051`), which is suggestive and is not a published claim about resolution order — so it is not cited as one. **(A)** a frozen snapshot with an arbitrating resolver: equally deterministic, and it stalls a convoy. **(B, chosen)** resolve in order against the state as it stands |
| **A blocked unit keeps its target** (FR-4, AC-3, P-2) | Nothing decoded covers it. The alternative — clear the order on contention — would make an obstacle indistinguishable from an arrival to every consumer of the target flag, which is the one distinction a caller cannot recover for itself |
| **Occupancy is derived while advancing and never stored** (C-1, FR-9) | The original persists it: `TERR-PASS-053` (High) has an MFC `Serialize` sweep the dynamic plane and emit one `u32` per cell whose byte exceeds `0x0f`. Ours is transient, which is exactly why the byte form and the digest do not move. Divergent, and cheaper to change later than a format bump would be |
| **One cell per axis per tick, integer only** (FR-6, C-2, C-3) | The original's step is sub-cell. `TERR-MOVE-056` (High) derives a per-step speed from the class speed, a registry multiplier, the two cells' height difference clamped to +/-32 and the mean cost byte, clamps it to `[1,63]`, and scales a diagonal step by the **double `0.707`**. Consuming that collides with our own determinism wall, so it is a story of its own and not a corner of this one |
| **A malformed start is not repaired** (FR-8, AC-11) | No evidence bears; the case is forced by this tree rather than preferred — see *Open* |

## Open / undecoded

- **The original's resolution order**, and whether it has one. Carried from 0019 unchanged.
- **What the original does with a mover whose next cell is taken** — wait, re-path, or displace the
  occupant. Nothing decoded. Here it waits and keeps its order, and that is disclosed rather than
  presented as recovered behaviour.
- **Terrain and object passability are decoded, and deliberately not consumed.** `ALM-GRID-012`,
  `TERR-PASS-049` and `TERR-PASS-050` give the tile-word bit-13 arm, terrain class 8, the raw water
  test, the type-3 object arm writing block value 5, and the 8-cell `0x1f` border — High for the
  arms. This contract blocks on units alone, so a unit still walks onto water and off the grid. The
  baseline's reason, *not decoded yet*, is wrong at this pin, and which reason holds matters:
  undecoded would make this a request to research, a later story makes it ours to schedule.
- **The sub-cell fraction the loader drops.** `ALM-UNIT-018` records the low byte as usually `0x80`,
  so a fraction genuinely exists. Restoring it would remove the malformed-start case at its source;
  FR-8 pays for it instead.

## Confidence, and the High threshold this story cannot meet by citing

Policy sets a **High** threshold for anything reaching hashed simulation state, and trajectories are
hashed here. There is nothing to grade: no claim is cited, so no confidence attaches to any anchor,
and a graded row would be a fabrication rather than an assurance. What the threshold exists to buy is
delivered by mechanism instead — `internal/archtest`'s two-part determinism wall over `pkg/sim`, and
the digest-equality criteria, which fail on a divergence rather than reporting a confidence about it.

## Removed from the baseline and why

- Its provenance-basis preamble, and its closing section that exists to report having no research
  item — both refused by the doc-budget content bans wherever they appear.
- **`MoveTo(id, x, y)` as the operation that sets a target.** No such symbol exists in this tree.
- **`openrom/` as the determinism wall.** Not a path in this module, and a different shape of check
  from the single test the baseline implies.
- **"Terrain / tile passability — not decoded yet."** False at this pin; restated as scheduling.
- **The precondition that every world from the normal construction and command path is well-formed.**
  Refuted in this tree, so the contract gained FR-8 rather than resting on a guarantee it does not
  have.
- **"A descending-ID convoy stalls one cell per tick."** Measured; the line stretches until gaps
  open and then flows again, and the disclosure now says that instead.

## Appended 2026-08-01 — pin `130bb79`: three cited rows gained an overturn, none moves a requirement

**`TERR-MOVE-057` — SUPERSEDED.** "2 = Ghost/Bee (`0x44`), 3 = Bat_Sonic/Dragon (`0x82`, air)"
reads as two grades of one thing and they are not a ladder: `0x44` crosses water and mountain and
is stopped by every object, building and ground occupant, while `0x82` is stopped only by the
8-cell border and another air occupant — they disagree on **83 203 of 880 704** shipped cells
(`MOVE-DOM-026`). No value, mask or address moved. This story cites the row for its **footprint**
half — the definition table's 2 on eleven entries and 3 on four — which the overturn does not
touch, so FR-1's one-cell model and the reason it is a *choice* both stand.

**`ALM-GRID-012` — SUPERSEDED.** "bit 13 = impassable flag" is one of three tile-word arms that
block a cell, and the smallest: 7 464 cells against 136 622 for the strip-pair class and 87 584 for
the raw water test, over a denominator of 880 704 rather than the 880 552 first published. The
*deliberately not consumed* row above already names all three arms and is right as written; what
would mislead is the claim id read on its own.

**`TERR-MOVE-055` carries no Kind at all.** It is `● active (amended)` and its `retracted.md` row's
Kind column is `—`, which at this pin means *nobody has looked*, not *neither applies*. The clause
it lost is "every `.alm` mover is 1x1 / `0x41`, masks 2 and 3 unreachable" — which this story cites
the row **against**: the per-instance footprint byte survives, that consequence does not. Recorded
so an unclassified row is not mistaken for an unexamined one.

`TERR-PASS-051`'s cell counts are also re-scoped at this pin (the ingest plane against the plane
after the structure pass). They are not cited here; named only so a reader comparing this file with
`0043`'s sees why the two quote different numbers.
