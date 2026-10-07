# 0129 — give all: provenance

## Claims this story is built on

| Claim | Confidence | What it settles here |
|---|---|---|
| `TRIG-GIVEALL-025` | High for the arm and the pour; High for the corpus figures; **Medium** for the notification's reach | Instant opcode 28 moves the **whole** container from the first reference to the second, `src->take(index 0, qty 1)` into `dst->add` until the source is empty, then destroys the source container and re-seats the giver with a fresh empty one. The giver keeps nothing and drops nothing. Corpus, both roots: three nodes, every one a map unit giving to `10001`; `10.alm` is 21 → 10001. |
| `TRIG-REC-011` | High for the record layouts and the three id bands | A compiled instant is 72 bytes with `+0x30` unit, `+0x34` group, `+0x38` player and `+0x3c` **the second reference of whichever kind came second** — so opcode 28's two references are the unit slot and that second slot, not two plain parameters. `Target_Unit` is three id spaces: `< 10001` the map's own unit table, `10001..11000` a hero ordinal, `> 11000` a static name table. |
| `ITEM-CONT-004` | High for the class shape; **Medium** for "nothing refuses" | The thing that holds items is one class with no slot count and no capacity, and a unit and a sack hold the same one. `+0x1c` is the next-insert index, not a limit, and an add lands at the tail whenever that index has reached the count. No insert path reads a maximum, so the receiving container cannot refuse a pour for being full. |

Read at the story's pin, `0908589`, with `go run ./tools/claim <ID>` from `research/`. All three are
active; none is retracted or amended.

## What is ours by choice

- **Refusing a node whose two references resolve to ONE entity.** The original's loop takes index 0
  and adds it at the tail of the same list, so the source never empties and the routine does not
  return. That is undefined behaviour, not behaviour to reproduce, and this build refuses it on the
  same ground `script.go` already refuses an out-of-range register subscript. No shipped node can
  reach it: all three name a map unit and the hero band, and no map unit is the hero.
- **The version number.** 38, allocated by the orchestrator. 36 is live and 37 is out to a lane
  running in parallel off the same master this one branched from; this tree carries none of its
  field.

## What is open, and deliberately not built

- **The notification packet.** `TRIG-GIVEALL-025` records `R0670` being called twice, once per
  owner, behind a `(u16)actor+0x0e` in `[0x21,0x40)` gate that the claim grades **Medium**: the gate
  is located and not interpreted, and what it excludes is not established. Nothing is built for it
  and no placeholder stands where it would go. A later story that wants it needs the gate decoded
  first, not this arm changed.
- **Gold.** The claim describes one container pointer moving and says nothing of a purse; `+0x7c` is
  the item list alone. So this arm moves no gold, and that is what the evidence supports rather than
  a simplification.
