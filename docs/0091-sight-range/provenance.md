# 0091 — provenance

## Backing

| Spec anchor | Source | Confidence |
|---|---|---|
| FR-1 — a unit's sight range is its own, one byte wide | `AI-SIGHT-006` (`actor+0xa5`, `CMP AL,byte ptr [ESI+0xa5]`); `UNIT-STREAM-001` slot 10 -> `+0xa5`; `DAT-HUMANS-008` slot 8 -> `+0xa5`, both between the `u16`s at `+0xa4` and `+0xa6` | High for the byte and its address / **Medium** that the store is the `u8` helper `R0281` specifically: the two rows name the helper family and the destination, and the width is then fixed by the two `u16` neighbours rather than read off the call |
| FR-2 — the march is seeded from the observer's own range | `AI-SIGHT-006`: the seed `(1 << (k-1)) + (scanRange << k)` is written into the accumulator's centre cell, and the routine reads the observer's `+0xa5` at `L00263` | High |
| FR-3 — a group's stamp is one march per member at each member's own range | `AI-GROUPSEE-068` (one shared byte map, every member stamped into it, one sweep against it); `AI-SIGHT-006` as amended — `R0110` clears then stamps every member of a group | High |
| FR-4 — the notice radius is `max over members of (that member's distance from the centroid + that member's own range)` | `AI-RADIUS-014`: `grpAI+0x2c = max of (distance + that member's sight)`, stored at `L00357`/`L00358`/`L00359`; `grpAI+0x2a`/`+0x2b`/`+0x2c` are consecutive bytes | High |
| FR-5 — a placement's range is its band's own column | `UNIT-STREAM-001` (units slot 10), `DAT-HUMANS-008` (humans slot 8), `DAT-ACT-006` (the `-1` cell keeps the constructor default, per-cell) | High |
| FR-6 — an unresolved placement carries 5 | `AI-SIGHT-006`: the constructor default at `L00260` is 5; `DAT-ACT-006` for the empty-cell law that makes it the value a silent row keeps | High |
| FR-7 — a generated hero's range is `4 + (Mind + Reaction)/25` on capped statistics | `HERO-SIGHT-007`: `ftol(((mind + reaction)/25 + 4) x 256)` -> `actor+0xa4` as a `u16`, whole-cell radius = high byte | High for the arithmetic (five named instructions, the divisor and the addend read as constants) |
| FR-8 — a column value outside a byte is truncated, not clamped or refused | The streamer's own store is `MOV byte ptr [ECX],AL` after the `-1` compare (`DAT-HUMANS-008`, widths re-read); the narrowing precedent in this tree is `relationFrom`'s low-byte conversion | **Medium** — the truncation follows from the store's width, and *no shipped row exercises it*: over both roots every value of both columns is 4..12 (measured, `verification.md`) |
| FR-12 — the opening view spans 15 map columns | `SESS-VIEW-028`: rect `(0, 0, screenW - 0xa0, screenH)` -> `+0x64 = (right-left)/32`; the three spans re-read from both roots; the 640x480 arm is what a consumer selecting nothing reaches, pinned by two branch displacements and by `InitInstance`'s `"-640"` fallback | High |
| FR-13 — the view opens centred on the party's own cell | `MISSION-VIEW-019`: `ScrollTo(pkt[+0xa] - visCols/2, pkt[+0xe] - visRows/2)`, the packet carrying the hero's cell | High. This story moves nothing here; it is named because FR-12 changes the span the halving is over |

## Ours by choice

| Statement | Why |
|---|---|
| The field is named `ScanRange` on `sim.Entity`, matching the two decoded columns | `pkg/data` already carries an unrelated `UnitDef.Sight` (no column, no reader, default 0). One name for one byte, and the loader cannot be written `Sight: def.Sight` by mistake |
| `formatVersion` 18, and version 17 is skipped | Allocated by the orchestrator; 17 was taken by the concurrent decay story, which landed first and whose seven bytes this branch merged. Nothing in the numbering is a claim about the original |
| The byte sits at the entity record's own tail, behind version 17's | Every tail since version 8 went there for the same reason: anywhere earlier moves documented offsets and buys nothing. It is also what let two parallel lanes compose — 17 took +92…+98 and 18 took +99, and neither restated an offset of the other |
| Every byte value is legal and none is refused on decode | The field's source is a byte, so there is no value to fold and none to reject — the argument `Facing` already stands on. A refusal would make a state the constructor accepts a state the decoder will not read back |
| The march clamps nothing: a range of 0 lights the observer's own cell alone | The observer's cell is marked before any ring and outside every test, which is the law's own walk; a budget of half a step then fails every ring-1 cell |

## Open

| Question | Status |
|---|---|
| Whether the human recompute runs at spawn for an `.alm`-placed person, overwriting the streamed column | Undecided, and deliberately not decided here. `DAT-HUMANS-009` carries the same question at Medium for `Defence`. This build takes the streamed column for a placed person and the derive for a generated hero, so no path needs the answer |
| The second `disp:a5` writer, `L00261` — the per-actor state machine's arm `0x17` (`AI-GUARD-007`'s radius overwrite) | Located, and out of reach: the per-actor machine is not implemented at all (`0086` FR-20's territory). Disclosed as D-1 |
| The low byte of `actor+0xa4` — the sub-cell remainder of a hero's sight | No consumer decoded. Not carried. Disclosed as D-2 |
| How many ROWS the opening view spans | Still not fixed by this build. `SESS-VIEW-028` publishes 15/18/24 for the three resolutions and also that an open panel recomputes the row count; the view's height here follows the player's window. Disclosed as D-5 |
| `[View] X/Y` as a second opening-camera path | `MISSION-VIEW-020`, Medium, on inputs that are not facts about the image. Not modelled. Disclosed as D-6 |

## Removed

| Statement | Why it went |
|---|---|
| "The sight range is one constant for every unit" (`0090` D-1) | Discharged. The two writers the displacement sweep could not see are named in FR-5 and FR-7 |
| "`AuthoredStartColumns` is an upper bound and not a measurement" (`0088`) | Discharged by `SESS-VIEW-028`. The verdict AUTHORED is dropped with it — the number is now decoded, and the sentence that placed it "alone so that overruling it is one edit" has been spent |
| A per-band `Sight` field distinct from the scan range | Not written. `UnitDef.Sight` exists, decodes nothing, and is not `+0xa5`; adding a second entity field beside it would put two names on one byte |
