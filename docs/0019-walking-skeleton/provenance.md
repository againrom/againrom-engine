# Provenance — the headless deterministic core

Pinned at research `909b330`, frozen for the story; `claims/retracted.md` was read first and nothing
in it bears here. This story is our own engine architecture: exactly one row below rests on a decoded
game fact, and everything else is recorded as ours rather than given a citation it does not have.

## Backing

| `spec.md` anchor | Claim | Confidence |
|---|---|---|
| *Loader* — a placed unit's cell is its record position shifted right by 8; the record's `X`/`Y` are `u32` fixed-point in 1/256 cell | `ALM-UNIT-018` (amended) | High — case 6 of the loader reads exactly 70 B per record and stores the first two dwords into the unit struct's `+0x00`/`+0x04`; `70 · count == payloadSize` on 38/38 corpus maps and 8094/8094 records land inside `[0,W) x [0,H)`, which a wrong stride or offset would scatter out of bounds. Already consumed in the tree: `alm.Unit.X`/`Y` carry the `/256` note |
| *Loader* — a world's bounds are the decoded map's width and height | 0003's container contract, carried unchanged | High as decoded there; this story re-derives nothing about the container |

## Ours by choice

| What the spec fixes | What the evidence says |
|---|---|
| **An entity id is the map's unit-slice index.** | The type-6 record carries a real unique unit id and `alm.Unit` exposes neither it nor the group id beside it. Slice order is therefore *our* identity, not the game's, and the two are not known to agree. A later story wanting the original's identity takes it from the record, not from this order — **at `+0x40`**: `ALM-UNIT-040` had the two labels the wrong way round and `ALM-UNIT-048` corrects them, `+0x40` the unit id and `+0x42` the group id that a trigger's `Target_Group` resolves against |
| The whole world / command / step contract: the tick and its ordering, the single write path, id-ascending iteration, the seeded RNG, the digest, the versioned canonical byte form, record and replay, and the runner | No claim bears on any of it and none could — it is our own engine, not the original's. Nothing in it was reverse-engineered and none of it is a fidelity statement |
| `pkg/mapload` owns the transform and `pkg/sim` imports no `againrom` package | Our own architecture (`docs/ARCHITECTURE.md`), enforced by `internal/archtest`. The baseline placed the transform in `pkg/sim`; see below |
| One cell per tick — no speed, no path, no collision, no clamp to the bounds | The original's movement rules are undecoded, and this story does not need them: the behaviour is deliberately trivial and what is being proven is the boundaries around it |
| The 1/256 fraction is dropped rather than carried | `ALM-UNIT-018` records the low byte as *usually* `0x80`, the tile centre, so a sub-cell fraction genuinely exists in the data and the skeleton discards it. Re-admitting it is a later fidelity slice, not a correction to this one |
| The seed is a fixed named constant in the loader | Nothing decoded says what the original seeds from, or when. A constant keeps two loads of one map identical, which is what the digest contract needs |

## Open / undecoded

- **How long a tick is.** Nothing decoded gives the original's tick rate. Here the tick is an index
  and not a duration, so the story needs no answer — and does not supply one.
- **Update order.** That the original advances its units in any particular order is unknown.
  Ascending entity id is ours, chosen because it is observable and stable, not because it matches.
  **Answered since, and appended 2026-07-31 rather than substituted, because this file is a dated
  record.** True at this story's pin, where `claims/move.md` did not yet exist — EXP-0054 created it.
  At the pin current on that date, `MOVE-TICK-009`/`013`/`014` and `MOVE-ID-016` answer it at **High**:
  the tick loop has no sort and no priority key, the walk order is pure `AddTail` insertion history,
  and the runtime id is *"never the walk order."* `MOVE-TICK-017` adds that the original's order does
  not even survive a save/load, which ours does. So ascending entity id is a **divergence** we can
  now name, not a choice made in a vacuum — and a consumer of this contract should read it that way.
- **A unit ordered outside the map.** No claim covers what the engine does there. Nothing clamps
  here, and the contract discloses it rather than inventing a limit.

## Removed from the baseline and why

- **FR-9's "`pkg/sim` imports only `formats/alm` + stdlib".** Refused mechanically in this tree:
  `internal/archtest` holds `pkg/sim` to an empty intra-module allow-list, tests included, and
  carries a negative test named for this exact edge. The transform moved to `pkg/mapload`.
- **Its provenance-basis preamble, and its closing section that exists only to report having no
  research item** — both refused by the doc-budget content bans wherever they appear.
- **"any observable state difference changes the hash".** A 64-bit digest cannot carry that promise.
  What the contract asserts instead is that every field the canonical byte form carries enters the
  digest, with the single-field changes witnessed one at a time.
