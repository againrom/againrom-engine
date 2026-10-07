# 0156 — provenance

## Claims this story builds on

| Claim | Confidence used | What it settles here |
|---|---|---|
| `TRIG-ADDITEM-027` | High for the arm and the creation-vs-transfer distinction | Instant 12 creates an item from the record's own code and adds it to the named unit's container. It reads no source. FR-2. |
| `TRIG-TAKEITEM-038` | High for the arm, the free and the detach | Instant 13 removes one unit by code: the element is unlinked at a count of 1 or below and one unit is split off above it. FR-3. |
| `TRIG-XFERITEM-039` | High for the builder's three stores | The compiled record's item field is `0xe18 + V` as a word. This is the whole of FR-1's arithmetic and the only place it comes from. |
| `ITEM-CODE-029` | High for the masks | A code's class is bits 8..11; class 0 resolves to nothing. FR-4's zero refusal. |
| `MISSION-M30-024` | High | `30.alm`'s win is `trig[6]`: check 6 between unit 10001 and unit 56 compared `<= 3`, running message, two instant-13 nodes and the win. `trig[7]` is always true and runs instant 12 with `Unit=10001 Item=6`. |
| `MISSION-CURE-026` | High for the mechanism | The map creates the item, hands it to hero 1, destroys it at the destination, and **tests for it nowhere**. This is why the build must not make the item a precondition of the win. |
| `MISSION-LOSE-025` | Medium, scoped to the script | `30.alm` authors no reachable lose path. |
| `ALM-TRIG-046` | Medium / Unknown for `Target_Item` | The nine parameter type codes; type 8 is `Target_Item`. Its **domain** is graded Unknown there, which is why FR-1 rests on `TRIG-XFERITEM-039`'s builder store instead. |
| `TRIG-BIND-010` | — | The zero-left rule that drops `trig[0]`; already implemented. |
| `TRIG-REC-011` | — | The `10001..11000` hero-ordinal band; already implemented. |

## Cross-check performed here

`TRIG-XFERITEM-039` gives `0xe18 + V`. `MISSION-CURE-026` states independently
that mission 30's node produces class 14 index 30. `ITEM-CODE-029` gives class as
bits 8..11 and, for class 14 only, the index as the whole low byte. With `V = 6`,
`0x0e18 + 6 = 0x0e1e`, whose class nibble is 14 and whose low byte is 30. The two
claims agree, and `DAT-DOC-021`'s `MagicItems[28]` at packed code `0x0e1c`
(`0x0e18 + 4`) is a third value in the same block.

`ALM-TRIG-046` records `Target_Item` values 2..36. Added to `0x0e18` those give
`0x0e1a..0x0e3c`, every one of class 14 and none of them zero, so no shipped node
reaches FR-4's zero refusal.

## Ours by choice

- The **presence flag** on the compiled item reference. The original's builder
  leaves the record's field at its constructed value when a node names no item,
  and a code of zero already resolves to nothing, so the flag is not a fact about
  the original. It is here for the reason the player flag is: consistency of one
  record's five other reference slots.
- **Where the `0xe18` addition happens.** The original does it in the builder and
  so does this build, but nothing forces a reimplementation to; the alternative
  would carry the authored value on the record and add in the arm.
- **`formatVersion = 46`.** See `plan.md` DD-4. This lane could not reach the
  orchestrator mid-run and allocated 46 from the observed state of every branch
  and worktree on this machine: master 44, `0154-spells` 45, nothing above.

## Open

- **What the created object is** beyond class 14 index 30. `ITEM-CODE-029` grades
  the three constructor arguments Unknown, and `MISSION-CURE-026` grades the
  object's identity Medium because the only other evidence is the map author's
  own label text. This build creates an element with the packed code and nothing
  else; no weight, price, name or effect is attached, and none is needed by
  anything in `30.alm`.
- **Check opcode 17**, `Item in inventory`, `TRIG-ITEMTEST-040`. Published, 23
  authored nodes, not built here — `30.alm` authors none.
- **Instant opcode 11**, the transfer, `TRIG-XFERITEM-039`. Published with 0
  authored nodes on either root, so building it would move no census number.
