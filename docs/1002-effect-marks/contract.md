# 1002 — effect marks: contract

## What will work after this story

A lasting effect that the original marks on its actor is drawn on that actor here.

The owner's request on 2026-08-16 is the subject: «Что насчет спрайтов защит от стихий и отдельный
броня щит мага, что с ними?» — the four elemental Protections and the mage's Shield draw nothing in
this build. They draw here.

Observable result, in `builds/1002-effect-marks/`:

1. Casting Protection from Fire, Water, Air or Earth on an actor puts a shipped sprite above that
   actor for as long as the effect stands. Four Protections at once form a diamond of half-width 6
   pixels.
2. Casting Shield puts a ring of shipped sprites around the actor, split by the actor's own sprite:
   the half above draws behind the actor, the half below draws in front. The ring's vertical
   envelope contracts to nothing and expands again on a 90-step cycle.
3. Casting Bless or Curse puts a 20-mark rotating ring on the actor, the two rotating in opposite
   senses.
4. Casting Stone Curse holds the target's animation frame. Casting Invisibility removes the target's
   own sprite for every participant that does not detect it.
5. The eighteen spells that mark nothing in the original mark nothing here.

## Claims

| Claim | Confidence | What it fixes |
|---|---|---|
| `MAGIC-MARK-059` | High | The 8-byte mark record, the array, the two-pass unit draw split on `depth`, and the position expression |
| `MAGIC-MARK-060` | High / Medium | The mark set is re-derived every rebuild from kind-and-countdown pairs; open at `0xffff`, close by message, decrement once per rebuild |
| `MAGIC-MARK-061` | High | `kind = 2*spellId + 8`; ten kinds reach seven builders, 35 in range reach none, five fall outside the range |
| `MAGIC-PROT-062` | High | The four Protections' one builder and its placement table |
| `MAGIC-SHIELD-063` | Medium | The Shield's rasteriser, envelope, two records per point and signed depth |
| `MAGIC-BLESS-064` | Medium | Bless and Curse: 20 marks, five steps of 18 degrees, opposite senses, frame `4 - step/18` |
| `MAGIC-CLOUD-065` | High (first arm) / Medium | Poison Cloud's single mark; Heal and Drain Life carry the previous rebuild's records forward |
| `MAGIC-ACTOR-066` | High / Medium | Stone Curse and Invisibility change the actor's own draw instead of adding a mark, tested by kind directly |
| `MAGIC-PIC-026` | High | `2*spellId + 8` is the record index and the `projectiles.reg` row the art resolves through |

## UNKNOWN

- The per-player bit `MAGIC-ACTOR-066` gates the Invisibility sprite arm on. Its owning table was
  not identified. This build authors the participant test in its place. → DIV-074.
- The Shield's rotation angle per phase step, its point count and the closed form of its horizontal
  extent (`MAGIC-SHIELD-063` states all three as not established), and the assignment of the four
  sign combinations to Bless and Curse's four records per step (`MAGIC-BLESS-064` read it from store
  order rather than proving it). → DIV-075.
- The spawn branch of Heal's and Drain Life's carry-forward arms (`MAGIC-CLOUD-065`), and with it
  `MAGIC-MARK-060`'s third element writer, the projectile driver. Poison Cloud's element is opened
  by that writer too, and this build delivers Poison Cloud as a cell effect rather than as an attach
  on an actor, so none of the three opens an element here. → DIV-076.

## Twelve aspects

| Aspect | Applies | Why |
|---|---|---|
| Data | Yes | The mark art resolves through `projectiles.reg` at the record index; `TileSize` scales every offset |
| Runtime state | Yes | The client mark clock: one countdown per (actor, kind), opened, decremented per rebuild, closed |
| Simulation | Yes | The effect set the clock is opened and closed from; the invisibility detection answer |
| Player input | No | Nothing here is commanded; the marks follow effects already castable |
| AI | No | No decision reads a mark |
| UI / HUD | Yes | The unit draw's two passes, the sprite gate and the animation hold |
| Triggers / scripts | No | No script node reads or writes a mark |
| Inventory / equipment | No | — |
| Persistence | Yes (N/A row expected) | The mark set is not stored: `MAGIC-MARK-060`. No byte-form change is expected |
| Campaign / session | Yes | The integration witness runs in a real campaign mission |
| Shipped content | Yes | All ten marking kinds must name a defined `projectiles.reg` row |
| Interactions | Yes | The existing authored `SpellFX` ring and the Heal burst both stand beside the new marks |

## Domains

**Client** (`pkg/render/terrain`, `pkg/ui`, the view half of `pkg/game`) is the domain that gains the
work: every builder is client geometry. **Combat & Magic** inside `pkg/sim` is read for the effect
set and gains one read-only query. **Assets** (`pkg/data`) is read for `TileSize` and the picture id.
**Persistence** is untouched.

## Out of scope

- `MAGIC-MARK-060`'s projectile-driver element writer, and with it the Heal, Drain Life and Poison
  Cloud marks (DIV-076). Heal already draws an authored target-local burst here
  (`pkg/ui/healstars.go`), so the actor is not unmarked on a heal.
- What stops a stone-cursed actor walking. `docs/DIVERGENCES.md` DIV-073 owns it and the movement
  half is undecoded.
- The `0154` authored `SpellFX` school ring. It stays as built; the decoded marks are drawn beside
  it.
- Any change to the serialized byte form. `formatVersion` is returned unused.

## Expected divergence rows

DIV-074 (UNKNOWN, invisibility participant test), DIV-075 (UNKNOWN, authored Shield rotation and
Bless/Curse sign assignment), DIV-076 (UNKNOWN, Heal and Drain Life carry-forward arms not built).
