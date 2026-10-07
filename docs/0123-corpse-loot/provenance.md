# 0123-corpse-loot — provenance

Research pin: submodule `research` at `87a256d`. Every row below was read through
`tools/claim` at that pin, never from a ledger by hand.

## What the contract is built on

| Claim | Confidence | What this story takes from it |
|---|---|---|
| `ITEM-DEATH-012` | High | The whole shape of FR-1: on death the corpse's container object **becomes** the sack, receiving the same object the actor was carrying — identity, not a copy — and the corpse is then given a fresh empty container. Also FR-2: a sack is built **iff** the container is non-empty or gold is non-zero; with gold declined here (below), that is "iff non-empty". |
| `ITEM-SACK-010` | High (contested) | FR-3, the merge. Creation goes through one routine that **looks the cell up first** and, when a sack is already there, pours the incoming container into it and adds the incoming gold rather than building a second. One sack per cell is the engine's own first act, not a normalisation of ours. Also the fact that a sack holds a container **of the same class a unit holds**, which is why a drop is a move and not a conversion. |
| `ITEM-SACK-011` | High / Medium | Why nothing here schedules anything: a sack does not tick and does not expire — its two tick slots are prologue-and-`RET` stubs and it never enters the actor list. A dropped sack lies until it is picked up. |
| `ITEM-CONT-004` | High / Medium | Why the drop refuses nothing on size: the container class has no slot count and no capacity, and no routine in it compares either field against a limit. A corpse with a hundred codes drops all of them. |
| `HERO-KILL-027` | High / Unknown | The gold clause, read and **declined**: `treasureMin + U[0, treasureMax]` when the treasure chance exceeds `U[0,100]`, for `typeID > 0x40`, from parameters `0x26`/`0x27`/`0x28`. Recorded here so the story that builds it does not have to find it again. |
| `UNIT-STREAM-001` | High / Medium | Why the gold columns are not reachable from here: the units spawn streamer reads slots 0–37 and **slots 38–54 (treasure, Power, the spell pairs) are not streamed**. `pkg/data`'s `UnitDef` mirrors that boundary exactly, at `lastUnitSlot = 37`. The death routine fetches them from the template on demand instead — a path this tree has no equivalent of. |
| `AI-RAND-058` | High (identification) | FR-7: the original's generator is MSVC's CRT `rand()`, `seed*214013+2531011`, returning `(seed>>16)&0x7fff`. The row is **amended**, and `claims/retracted.md` carries an overturn of its arm-`0xb` *consequence* only — the identification and both constants stand. |
| `AI-RANGE-102` | High / Medium | The divisor: `AImgr+0x00` is `0x8000` = `RAND_MAX + 1`, so `n × rand() / AImgr[0]` is uniform on `0…n−1`. Together with the row above this is what makes `pkg/sim/rng.go`'s doc block stale. |

## What is ours by choice

- **The drop site.** `ITEM-DEATH-012` names one routine that does the unequip, the
  suppression test, the gold roll and the sack build together. This tree splits a death
  across `clearFelled` and `decayPass`; putting the drop in `clearFelled`, under the
  existing `Decay == DecayNone` guard, is our own placement, chosen because that guard
  already carries "this fires once per death" for the stage, the dwell and the defence.
- **Merge by binary search into an ordered list.** The original keeps a per-cell registry.
  Ours reproduces the observable rule (one sack per cell, incoming poured into the
  standing one) over the (Y, X)-ascending list 0111 already built. Not observable through
  anything this build exposes.
- **The bounds guard (FR-5).** No claim covers a corpse standing off the map, because the
  original has no such state to cover. It is a refusal of ours, chosen over a clamp on
  `sackFault`'s own ground: a world this package could marshal and then not read back is
  worse than a sack that is not built.
- **The generator itself.** `pkg/sim`'s RNG is SplitMix64 and stays so. `AI-RAND-058`
  describes the original's; adopting it is a separate decision with a digest cost, and
  this story only corrects the claim that nothing describes it.

## What is open

- **Which frame a sack draws.** `pkg/game/sacks.go:77` answers the literal `0` for every
  sack, and the sacks this story creates use it unchanged. That is an open research
  question, not a decode we skipped — see 0111's own spec. Not touched here.
- **Parameter 15 of a Weapons row.** `ITEM-DEATH-012`'s own Unknown cell: what the column
  is *called* is not established; the behaviour is "a weapon whose column 15 is 0 is never
  dropped". Nothing here names it.
- **Which of the fourteen equipment slots a death touches.** `ITEM-DEATH-012` says
  exactly two, `+0x74` and `+0x78`. If that holds, a body leaves at most two things and
  not a full kit — worth confirming from the game side before the equipment fold is built
  on top of it.

## What was removed

Nothing. No claim cited here was retracted or amended against this story's reading, and
`AI-RAND-058`'s overturn touches a consequence this story does not use.
