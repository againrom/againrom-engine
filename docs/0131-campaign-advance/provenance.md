# 0131 — the campaign advances: provenance

## Backing — what the spec asserts because something established it

| Spec anchor | Source | Confidence |
|---|---|---|
| FR-1: the successor a mission declares is `[Mission<n>] AutoGetMission` | `REG-SCN-063` | High |
| FR-1: an absent key and a value of `-1` are the same state, and that state is "no successor" | `REG-SCN-063` | High |
| FR-1: any other value is a successor | `REG-SCN-063` | High |
| FR-2: a declared successor is entered directly, without passing through the campaign's home screen | `REG-SCN-063`, `SHOP-TOWN-023` | High |
| FR-4: where no successor is declared, the campaign goes home rather than onward | `REG-SCN-063`, `REG-GMAP-066` | High / Medium |
| Out of scope: what the home screen then is | `REG-SCN-064`, `SHOP-TOWN-022` | High |

`REG-SCN-063` is the load-bearing row and it is worth saying exactly what in it
is read rather than inferred. The reader stores the key at a fixed record offset
**with a default of -1**, and that offset has exactly one reader, itself called
from exactly one site; its two arms are `!= -1`, which loads that mission's
record and travels to its map object, and `== -1`, which sends the party to the
city. Both arms, both defaults and the store are read at instruction level, and
the call census is 1 hit with none in orphan code. So the equivalence of "absent"
and "-1" is not a reading we chose: it is the arithmetic of a default. The row is
**active** and carries one Unknown, about `LastMission`'s consumer, which this
story does not use.

`SHOP-TOWN-023` is what makes FR-2's "without passing through the home screen"
an established fact rather than a convenience: it establishes on three
independently read legs that the mission-10-to-20 stretch never reaches the town
at all.

## Ours by choice — what the spec fixes that no source asserts

| Spec anchor | The choice | Changeable later without contradicting anyone |
|---|---|---|
| FR-4 | Where no successor is declared, the sentence and the map list stay exactly as they are today, and the successor **named in that sentence** stays this tree's own ascending-by-tens order | Yes — it is scaffolding for a home screen that does not exist |
| FR-5 | A declared successor at or below zero is refused rather than attempted | Yes |
| FR-6 | A successor whose map will not read leaves the player on the map list holding the reason | Yes |
| FR-9 | That the headless check mode reports the advance at all, and the shape of the line | Yes |
| AC-8 | That the input which dismisses a banner reaches nothing inside the mission it opens | Yes, but it would be a defect to change |

FR-5 deserves its reason. The original's fork is `!= -1`, so a shipped `0` would
select the advance arm there; what its own mission loader then does with mission
0 was not read, and this tree has no address for a non-positive mission number.
Refusing is therefore **ours**, chosen because the alternative is composing an
address that cannot exist and reporting the failure as if the install were
broken. No shipped campaign reaches it: the key ships once, at 20.

## Open — undecoded, and deliberately given no meaning here

- **`LastMission`.** Ships once, `[Mission150] = 1`, stored beside
  `AutoGetMission` by the same routine. What reads it is `REG-SCN-063`'s own
  Unknown. This story assigns it nothing.
- **The announce flag** the original latches on the advance arm. Read as a call,
  not as a behaviour; this tree has nothing to announce into.
- **`globalmap.reg`'s travel.** The original walks the party to the successor's
  own map object before the mission opens (`REG-GMAP-066`). This tree has no
  global map, so the advance is a load and not a journey — an absence, not a
  contradiction.

## Removed — statements dropped, and why

- *"The campaign advances by tens."* True of the shipped data and false as a
  rule: it is `Campaign.NextMission`'s authored order, and `REG-SCN-063` shows
  the file has its own answer. Kept only where it already lives, as the
  scaffolding sentence of FR-4.
- *"Missions 10 and 20 are the prologue."* A conclusion about the campaign's
  shape, drawn in `REG-SCN-063` from the key's census. Nothing in this contract
  depends on which missions those are, and a spec that named them would be
  hard-coding a stock scenario.
