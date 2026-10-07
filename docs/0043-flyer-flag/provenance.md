# Provenance — 0043 the mover's domain

Submodule pin: research `e47abfb`. Every row below was read at that pin.

## Claims this story is built on

| Claim | Confidence | What it settles here |
|---|---|---|
| `TERR-MOVE-057` | High | The domain is a **column**, not a flag: `movementType` reaches the actor's `+0x4a` from the placeable-definition database at spawn, and the shipped Units table carries **2 on Ghost/Bee, 3 on Bat_Sonic/Dragon** and nothing else — the streamer skips the store on an empty cell, so the constructor's 1 survives wherever the table is silent. Backs FR-1's three values and its default. |
| `TERR-PASS-051` | High | The mask the domain selects — `1` to `0x41`, `2` to `0x44`, `3` to `0x82` — and what the bits mean: **bit 0 blocks ground, bit 1 blocks air and is set by nothing but the border, bit 2 is a static object**. Its census is FR-2's own sentence measured: over 880 704 shipped cells a ground mover is blocked on 430 898 and an air mover on **158 976, exactly the border**. The mask constructor leaves `0x41`, so ground is the default (FR-1). |
| `TERR-PASS-073` | High | The same bit semantics from the other side, with both planes' complete writer list and the instrument's blind spots closed by hand. FR-8 rests on it: bit 1 has one writer, the border, which is what makes the landed margin read sound and what this story may not disturb. |
| `MOVE-PLANE-005` | High | Every mask carries its own occupancy half — bit 6 for `0x41`/`0x44`, bit 7 for `0x82` — set keyed by the same selector (`<3` to `0x40`, `==3` to `0x80`). Backs FR-3: the domain picks the plane, and a ghost contends with ground rather than with air. |
| `MOVE-COST-002` | High | `movementType 1` is *the ordinary ground domain* and every other mover is charged flat — the engine's own division into a default domain and the rest, which is the shape FR-1 takes. |
| `ANIM-DEATH-007` | High | Names the `movementType > 1` classes outright — Ghost, Bee, Bat_Sonic, Dragon — four, where the baseline's rule names five. |
| `HERO-DEATH-026` | High | The same four from the death path's own instructions: `(u8)actor+0x4a > 1` is "above 1 on exactly Ghost/Bee = 2 and Bat_Sonic/Dragon = 3". Independent of the row above, so the verdict rests on two readings and not one. |
| `MOVE-WAIT-008` | High | What the engine does when a mover's next cell is taken: it **waits facing it**, never pushes, swaps or interpenetrates. Cited for the divergence, not for a requirement. |

## Ours by choice

- **The soft collision itself.** Interpenetrate while moving, distinct at rest (FR-4, FR-5, FR-6) is
  the project owner's design requirement for this engine. Nothing above describes it and nothing was
  bent to make it look derived; see the divergence below.
- **Our own three codes, with ground as the zero value.** `Domain` is this package's enum, not the
  column's numbering, exactly as `Mode` is. Ground is zero because the engine's mask constructor
  defaults to the ground mask: a caller who says nothing gets the domain the engine gives a mover
  that says nothing, and every world already built keeps the behaviour it had.
- **The ghost's terrain term.** The engine's `0x44` reads bit 2, which the border literal also
  carries. This plane derives no bit 2, and the one border term it derives besides bit 0 is bit 1 —
  so a ghost is blocked here by the border and by nothing else.
- **The three sites the rest predicate is asked at** (FR-6). The engine has no soft collision, so no
  claim places them.
- **Reopening a settle for an occupied goal**, which the landed approach story put out of scope. It
  is reopened for the air domain alone: a flyer's search is blind to units by FR-5, so this is the
  only place occupancy can enter it.

## Divergence, disclosed

**Two flyers do not interpenetrate in the engine.** Bit 7 is a hard block in the dynamic plane and a
mover whose next cell is taken waits facing it (`MOVE-WAIT-008`, `MOVE-PLANE-005`). What ships here
is the owner's rule instead: a moving flyer is counted nowhere, so it neither blocks nor is blocked,
and the distinctness the engine gets from the plane is recovered at the cell an order ends on. A
deliberate departure from the reconstruction, not a reading of it.

**A ghost is stopped by less here than in the game.** `0x44` blocks that mover on 229 092 of 880 704
shipped cells; this plane's ghost is stopped by the border alone, bit 2 having no derivation here.
The gap is bit 2's, not the domain's, and it closes when a story derives one.

## Open — deliberately assigned no meaning

- **Which domain a placed unit is born in.** The column is `Data.bin`'s `movementType`
  (`DAT-SCHEMA-004` for the schema, `TERR-MOVE-057` for the values), and this tree has no reader for
  that file — `pkg/formats` holds `res`, `reg`, `alm`, `spr16`, `spr256`. So a map-built world spawns
  every entity ground (FR-8): right for 30 of the 34 classes, wrong for four. Nothing is guessed.
- **Bit 2 of the block plane.** Derived nowhere, here or in `pkg/mapload`. It stays reserved and
  refused by the world constructor, unchanged.
- **Bits 3 and 4, and the dynamic plane's bit 5.** `TERR-PASS-073` grades their origin **Unknown**.
  Nothing here reads them.

## Removed — what the baselines asserted and this story does not

| Dropped | Why |
|---|---|
| `IdlePhases > 0` as the flying test | A correlate of the decoded column, not the mechanism, and it disagrees with it on membership and on shape alike — a boolean where the engine has a three-valued domain. Death Star carries an idle animation and `movementType` 1; Ghost and Bee carry neither the ground mask nor the air one. |
| `Flying()` on `data.UnitClass` | Removed with the rule it would have read: the struct is pinned field-for-field against `units.reg`'s 37-key inventory, `movementType` is not one of those keys, and `classes.go` states that what a key does is not established — so the method would assert a meaning for `IdlePhases` that no source gives it. |
| `Z` as the flying test this story replaces | Nothing here reads `Z` for movement, or reads a flying test at all, so there is nothing to withdraw. |
| A flyer must never linger as a stationary targeted holder | Closed already, for every domain, by the landed approach story: a far search that cannot reach its goal settles and the order **takes the settled cell**, so no mover holds an unreachable target across ticks. |
| The path preview must seed occupancy exactly as `Step` does | There is no second seeding: `Route(id)` reads stored route state, so the defect this was written against cannot occur. |
| The occupancy-erasure repro | It reproduces a defect of a presence-only set this tree does not have: occupancy is a **count** here, and a moving flyer is never counted, so no entry exists for a departure to erase. |

## Not consulted

No third-party reimplementation, port, decompilation, format schema or web source was read for any
fact here. Both baselines folded into this story are our own earlier clean-room work, treated as
hypotheses: where one disagreed with a decoded row, the row was taken and the disagreement written
down above.

## Appended 2026-08-01 — the pin moved to `130bb79`, and one figure above moved with it

A provenance is a dated record. Every row above says what was read at research `e47abfb` and is
left exactly as written; what follows is the correction beside it.

**`229 092` is superseded by `242 179`, and the reason is a plane, not an error.** The *Divergence*
row gives the engine's `0x44` mover 229 092 blocked cells of 880 704. `TERR-PASS-051`'s census is
re-scoped by `MOVE-DOM-026`: the published set `430 898 / 229 092 / 158 976` counts the **ingest**
block plane, before the structure pass, and after that pass the same three masks read
**`441 540 / 242 179 / 158 976`** — domain 3 unchanged, because no structure arm touches bit 1.
`retracted.md` classes this **SUPERSEDED**: nothing this story asserted is wrong, the plane the
figure counted was named later. What the correction costs the *Divergence* row is that the gap it
sizes is bigger than it says — `242 179 − 158 976 = 83 203` cells, the two non-ground domains'
entire disagreement, against the 70 116 the old figure implies. It is still bit 2's gap and it
still closes when a story derives one.

**The layer split holds, and it was cited here rather than reasoned out.** Ground and ghost share
occupancy **bit 6**, the flyer uses **bit 7** — now also a row of `MOVE-DOM-026`'s table, and
already carried by the `MOVE-PLANE-005` row above at this story's own pin. Recorded because
`builds/0043-flyer-flag/README.md` says the split is "exactly the layer split this story chose on
its own reasoning": that is the one sentence of the owner artifact this sweep contradicts, and the
correction runs in the story's favour — the reading was evidenced when it was made.

**Two grades on rows above have moved, and neither changes a requirement.** `TERR-PASS-051`'s
*naming* clause — that `vt+0x20`'s codes are movement domains — now reads **"Medium, was Unknown,
was Medium"**, restored by EXP-0078 on three witnesses that do not depend on the mask; the
predicate, the mask setter and the occupancy writers keep their High. The row above grades the
claim flat High, which is right for what it cites it *for* and too strong for the name. And
`TERR-MOVE-057` has gained a **SUPERSEDED** row: "2 = Ghost/Bee (`0x44`), 3 = Bat_Sonic/Dragon
(`0x82`, air)" reads as two grades of one thing, and they are not a ladder — `0x44` crosses water
and mountain and is stopped by every object, building and ground occupant, while `0x82` is stopped
only by the border and another air occupant. No value, mask or address moved. FR-1's three values
and FR-2's masks are untouched; the word that must not travel with them is *weaker*.

**The domain still has no consumer in `pkg/mapload`.** Re-checked at this pin, not taken on report:
no file under `pkg/mapload` names `movementType` or a domain, so `FromALM` still spawns every
entity ground — right for 30 of the 34 classes, wrong for four, exactly as the *Open* row says.
