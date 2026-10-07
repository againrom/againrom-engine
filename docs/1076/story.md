# 1076 — Original ground-loot continuation

## Intent and delivered boundary

The owner's objective is faithful SAV continuation, including an original-format
writer driven by live state. This candidate closes one complete supported mission
axis: original ground Sack list → live ground loot → pickup → native save/load.
It does **not** complete that larger objective or broaden story 1073's original
city writer. Mission saves remain explicit `.ags`; no imported world is passed
through as if it represented current play.

Based on implementation master `f136861b8f8ce338f06c9a36afe1f17c6f9f7f18`, with
research pin `26c755b2be9513b9ec90edba6527db4fdd613dc4`. The earlier 1076 checkout
contained no tracked or ignored Phase-A artifacts to recover.

## As built

`sav.File.GroundSacks` follows archive indices through Players, dead actors,
Buildings, SpellEffects, counted terrain, session, Sacks and the common trailer.
It does not scan for Sack strings or substitute a corpse's inventory. The walk
includes the published spell subclasses, Outpost's counted records, Tavern and
Shop. Null/wrong-class references, shared ground-item ownership, duplicate Sack
identity, unsupported schemas and malformed extents fail closed.

Both `ResumeOriginalSave` and the production `FrontEnd.RestoreOriginal` path
replace fresh-map sacks after mission construction and before visibility/tick.
An empty saved list clears fresh loot. Packed cell, gold, item code, kind, signed
price, full stack count and ordered state-0 effects reach canonical state;
same-cell sacks merge in source order. Imported codes enter the weight table.
The atomic simulation seam deep-copies input and leaves the world unchanged on
invalid bounds. No native format bump is needed.

`ITEM-STACK-003` makes Item+42 the count. The Item+08 write in
`SAV-POSTLOAD-223` changes flags, not that count: quantity-one extraction repeats
until the source stack is drained, deep-copying effects. Native item instances
therefore expand the full count, not one value per source object.

Ground import refuses non-centered fine coordinates, zero code/count,
non-state-0 effects and expansion past 1048576 item/effect values. Cached load,
insertion cursor, archive identities and opaque Token fields are not claimed as
canonical Sack state. `DIV-026` is narrowed; `DIV-594` records the boundary.

## Evidence and proof

| Owner source | Exact SHA-256 | Measured ground state |
|---|---|---|
| preserved RU `game0000.sav` | `acc9a35bfb7fdcc7b234447024d1e7a45d4400d2c4bc47156fd93405f20803c9` | Four centered Sack objects, one item each |
| corpus `2026-08-12/game0012.sav` | `12b05b4eb3ff6921cdddfcf89965f525c40dc169331ee28c5f39b9a63c6c8f44` | Three sacks; `(20,65)` is absent, while fresh map 10 has four |
| corpus `2026-08-15/game0018.sav` | `1e2eb21f47082ab05a8f7810148fdf722fe8aa4f69a57d79953bd8fe0025eb6b` | Six sacks, seven item units; stack two at `(77,110)` with 78 gold; effect `(11,0,1)` on code 13377 at `(13,130)` |

Independent synthetic serializers test intervening variable-size graphs,
corpse exclusion, empty/no-world distinction, stack/ordered-effect projection,
every truncation, malformed framing, unsupported-state refusal and bounded
expansion. Simulation tests cover atomic replacement, alias isolation,
same-cell merge, empty replacement and native hash equality.

The two source-continuation `TestReleaseOriginalGround1076*` witnesses use real
installed content:
App LOAD demonstrates fresh four → source three; loading the close pre-pickup
`game0011.sav`, issuing the production walk/pickup order and using menu SAVE then
App LOAD retains the picked weapon and three sacks. The mission-40 witness
transfers gold and enchanted/stacked items, then requires exact canonical hash
equality after native continuation. EN/RU are two materializations of the same
owner source, not independent original observations. `savtool sacks FILE...`
provides a read-only source census.

Separate `savecheck` processes restored source `game0012.sav`, saved through the
production seam and reloaded `.ags` on each root. Both before/after hashes were
`3eb7d4f42dfe0a29`; both 58446-byte output files have SHA-256
`d0d63fbc33f1ee7f83bc7209042e81fd156b131a661bae48f8f5eb0265836d20`.
They are outside Git under `gameversions/saves/2026-08-30/story-1076-ground-native/`:
`en/save-20260830-154731.ags` and `ru/save-20260830-154743.ags`.

Initial gates: `gofmt`, `go test -trimpath -count=1 ./...`, asset guard and
divergence-claim check passed. The release gate ran all 110 declared tests on EN
and RU, with no missing subjects. Its existing hidden cutscene helper required
an approved run outside the sandbox and `GOFLAGS=-buildvcs=false`; earlier
sandbox attempts failed on VCS stamping and access, not on SAV assertions.
The relevant `0152-save666` scenario passed on both roots. The preserved-install
check verified all 181 recorded files unchanged. Original-format SAV generation
and ROM1 GUI acceptance remain with the seat's separate city-writer witness.

## Sole review correction

F1 in the seat's `pipeline/reviews/1076-adversarial-review.md` found that an
out-of-map original Sack was rejected after the FrontEnd had already discarded
the previous session. Original mission LOAD now prepares its Town, body cache,
map and live driver separately through the existing native prepare/commit seam.
All fallible work finishes before the returned opener adopts the new session
once. The fresh-party fallback cannot borrow the previous game's carried party.

`TestReleaseOriginalGround1076RejectedLoadPreservesLiveGame` reproduces the
original failure through App LOAD with a centered Sack at `(255,255)` on map 10.
After the fix, EN and RU retain the prior live pointer, Town, merchant, original
city provenance, body cache, town UI and complete Snapshot; the world hash stays
`2d0993936ef7d9a5`. Production SAVE after refusal decodes to that exact Snapshot.
Separate synthetic tests cover a missing mission map and successful preparation
which leaves the previous session intact until its one-time commit.

Correction gates: `gofmt`, full uncached Go tests, asset guard and diff whitespace
check passed. One release invocation ran all 111 declared tests on each EN/RU
root with no missing subjects. `0152-save666` passed on both roots. The existing
cutscene helper again required the approved outside-sandbox release run.

## Remaining research frontier

`SAV-WORLDFRONT-432` still leaves eleven world-head dwords, Position+06..07,
unnamed session spans/scalars, campaign+114/+128, the final global/trailer and
external map reconstruction without a safe source-free live projection.
`SAV-LIVEPROD-413` distinguishes named/derived fields from 35 Unknown Human,
Player and Group differences; installed initial values are not live-current
values. The source-faithful diagnostic in EXP-0263 had structural acceptance,
not original runtime acceptance of an authored current mission.

The next falsifiable actor question sent to research is: which live writers and
first post-load consumers define each byte in Human raw-a6 tail, raw-be tail and
raw-d4, and which bytes change under controlled equip/unequip, health/mana and
skill-progress mutations? Terrain, corpses, actor orders, casts, area effects,
trigger results and unnamed session state remain outside this candidate.
