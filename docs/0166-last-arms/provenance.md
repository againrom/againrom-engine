# 0166-last-arms — provenance

The evidence this story is built on, by claim id. Research provenance lives here and not in
`spec.md` (S-2).

## Claims used

| Claim | Confidence | What this story takes from it |
|---|---|---|
| `AI-FORM-037` (amended) | High for the field and its writers; the corpus clause was retracted and re-read | Instant 7 writes `(u8)p0` into the named player's formation mode byte, raw and unremapped. The amendment is load-bearing: a reference-typed slot takes no `p` index (`TRIG-PARAM-030`), so the campaign's one node authors formation **0**, not 2. The constructor default is 2. |
| `MOVE-GATE-035` | — (cited through `AI-FORM-037`) | The byte's three behaviours: 0 never in formation, 2 in formation iff the spread test passes, any other nonzero in formation unconditionally. |
| `TRIG-DROPALL-024` | High for the arm and the sack chain; Medium for "at the unit's feet" | Instant 20 pours the unit's whole container onto the sack at the unit's own cell, merging into a sack already there, and re-seats the unit with a fresh empty container. Gold is 0. |
| `TRIG-CELLTAIL-035` | High for the write and the shipped no-effect result; Medium that no indirect consumer exists | Instant 25 packs the cell as `(u8(y) << 8) OR u8(x)`, stores the six bytes `{spell, power, 0, y, 0, y}` at the cell record's `+0x2c..+0x31`, and the authored `x` is not copied into the tail. The only behavioural reader tests `+0x2c == 26`; the two shipped nodes write spell 3 and 9, so they create persistent cell state and nothing else. |
| `TRIG-EFFECTTIME-034` | High | Instant 30 walks the referenced unit's attached-effect list and, for every effect whose id byte equals `(u8)p0`, writes `(u16)p1` to the duration field. Missing references and absent matches are no-ops. The instant-29 half landed in 0165 and is untouched here. |
| `TRIG-PROPERTY-036` | High for the selector and the stores; Medium for the field names | Instant 34 is a word-wide property setter selected by `p0`: 6 writes current health, 15 defence, 16 absorption. Any other selector stores nothing. No clamp, no maximum-health update, no derived-stat recomputation. |
| `AI-SCRIPTATTACK-120` (amended) | High for the helper's two branches; the dispatcher's completeness was retracted | Sub-command 10 stops every member, then per member scores against the named unit and compares with the veto sentinel. Off the sentinel the named unit acquires in place and every other member engages it. On the sentinel the member acquires in place and never engages. |
| `AI-COST-071`, `AI-PREF-070` | — (cited through the two rows above) | The sentinel `0xffffff` is returned only where the preference matrix cell is 0. Both are already in this tree as `scoreSeed` and `preference` (`pkg/sim/engage.go`). |
| `TRIG-GRPARM-047` | High for the arms, the gate and the escort writes; High for the veto's shipped unreachability; Medium for the census | Sub-commands 11 and 15 are one shape with two state constants, 8 for Defend and `0x11` for Follow. The named unit gets acquire-in-place; every other member gets the escort state with the named unit as its escort target. The veto is authored-unreachable in the shipped campaign, established positively. |
| `AI-FOLLOWSET-116` | High for the shape; Medium that the three sites are the only three | The split on `member == the named unit`, and the two writes each side takes. |
| `AI-FOLLOWRANGE-115` | High for the readers and the writers; Medium that the write population is complete | The escort range is the node's own third value, or **3** when that value is 0. Shipped ranges are 1..6 and never 0, so the coercion is authored-unreachable. |
| `AI-DEFEND-111` | High for the arm and cover's polarity; Medium for the block size | What state 8 does per tick. Read to establish that the per-tick arm is a different contract from the setter, and not built here (SC-1). |
| `TRIG-GRPLIMIT-048` | High for each limit; Medium for the shipped figures | The G2 limits: the escort range is a 32-bit authored field read through an 8-bit path, so an authored 256 behaves as 0 and coerces to 3; sub-command 10 has no authored parameter beyond its two references and its gate is a compile-time constant; the group walk is unbounded. |
| `TRIG-PARAM-030` | High | A reference-typed slot takes no `p` index. This is what makes instant 7's `p0` the `Formation` int and not the player index. |

## What is ours by choice

- This build holds **one** attached effect per entity (`SpellFX`, `SpellFXSpell`), not a list, and
  that predates this story (0154 DD-5). Instant 30's walk therefore reduces to one comparison. The
  reduction is disclosed in `spec.md` as SC-2.
- The cell-record tail is stored as the six bytes the decoded helper stores, and nothing else of the
  52-byte record is modelled. The dynamic-plane bit-0 rejection and the recomputation mark have no
  counterpart in this tree; both are named in `spec.md` as SC-3.
- The per-tick behaviour of the three escort and acquire states is not built (SC-1). The actor pass
  reaches no arm for a state it has no case for, which is that file's own stated seam.

## What is open

- Whether the original serializes the formation block: `AI-FORM-037` reads `Player::Serialize`
  handing the `0x20`-byte settings block to a raw archive read and write, so it does. Whether the
  attached-effect duration survives a save is `TRIG-EFFECTTIME-034`'s own sentence: both mutations
  survive through the effect serializers.
- `SAV-CELLREC-017` is cited by `TRIG-CELLTAIL-035` for the raw cell record being serialized. It was
  not read directly for this story; the tail is carried in this build's byte form because state a
  script writes and a later tick can read must survive a save, not because that row was re-read.

## Corpus, measured in this tree

Read with a throwaway node dumper over `pkg/mapload`, all 28 campaign maps of the EN root, and
removed before the first commit. It reproduces `TRIG-GRPARM-047`'s and `AI-FOLLOWRANGE-115`'s own
figures from a path that shares no code with either:

- instant 7: 1 node (map 110), `p0 = 0`, player 2 — the amended reading, not the retracted one.
- instant 20: 3 nodes (maps 10, 80, 131), every parameter 0.
- instant 25: 2 nodes (map 150), `(spell 3, power 20, x 59, y 54)` and `(spell 9, power 60, x 112, y 26)`.
- instant 30: 2 nodes (map 90), `(p0 = 20, p1 = 60000)` and `(p0 = 20, p1 = 1)`.
- instant 34: 5 nodes — selector 6 once with `p1 = 1` (map 100), selector 16 four times with `p1 = 200` (map 151).
- sub-command 10: 5 nodes (maps 40 ×2, 71, 120, 140), all four packed ints 0.
- sub-command 11: 6 nodes, ranges `1×2, 3×3, 4×1`.
- sub-command 15: 10 nodes, ranges `3×4, 5×5, 6×1`.

The two range distributions are `AI-FOLLOWRANGE-115`'s exactly, which is what establishes that the
node's `+0x0c` int is `Args[1]` in this tree's compiled record.
