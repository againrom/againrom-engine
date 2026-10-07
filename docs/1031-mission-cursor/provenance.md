# Story `1031` — provenance

Research pin at implementation: `753034d2d9f432a6126c7f6bb56c972dbac5d503`. Every row below was
read whole with `go run ./tools/claim <ID>` from `research/`, not summarized from `contract.md`.
`AI-CURSOR-205` was read first, per the contract's own instruction: the routine `L01256` that
`AI-CURSOR-190`, `AI-CURSOR-192` and `AI-CURSOR-193` cite does not exist and every reference
attributed to it belongs to `R0338`.

| ID | Grade | What it establishes | Where used in `spec.md` |
|---|---|---|---|
| `AI-CURSOR-190` | High | The eight edge arrows: the screen-position test, the four-global block, the compass direction map, the 1px/2px band asymmetry the original reads | B1 |
| `AI-CURSOR-191` | (superseded reading corrected below) | The armed-mode field and the eight-entry jump table's own dispatch shape | B2 |
| `AI-CURSOR-202` | High | The view's own count of currently selected objects, forcing `sdefault` when zero | B2 |
| `AI-CURSOR-203` | High | The mask forcing `sdefault`, resolving an `AI-CURSOR-191` Unknown | B2 |
| `AI-CURSOR-204` | High | The right-edge selector routine, its fallback and its held-item override | Precedence (B1 ahead of B2/B4 interaction) |
| `AI-CURSOR-206` | High | A second, separate routine repeating the same arrow selection | Corroborates B1's direction map |
| `AI-CURSOR-207` | High | The minimap selector's own arrow block, ahead of its armed-mode tree, bypassing the widget bounds-check and the `sdefault` load on a hit | Precedence order (B1 before B2) |
| `AI-CURSOR-208` | not established (the row's own subject) | A rectangle test whose widget was not identified | DIV-260 |
| `AI-CURSOR-209` | Medium | The `town` slot's own condition, read at instruction level at both selection sites, game-side meaning unresolved. Reads instructions inside the same `L01515`-`L13182` tree as the `attack` write below, but not the addresses that gate that write | DIV-263 (not implemented), DIV-270 (corrects the row's own "no published claim covers this tree" to a scoped statement) |
| `AI-CURSOR-052` | High (Medium cap lifted by `EXP-0110`/`AI-KEYMOD-059`) | The hostility test at hover: `R0219` turns the hit-test mask into a cursor; Ctrl forces attack/swarm on the unit-class bits, Alt forces move, and the hostility bit `0x4` (inside mask `0x24` with a structure-marker bit `0x20`) suppresses the select cursor | B3 (the `select` half only), DIV-262 (modifier latches not read by this build), DIV-263 (mask reduced to the hostility bit alone), DIV-270 (the claim does NOT establish `attack` for an unmodified hover) |
| `UNIT-HOVER-020` | High for five named bits, Medium that they are all the bits | The hover hit-test's own capability mask: `CUnit`/`CAirUnit`/`CStructure` class bits by exact-name `strcmp`, a structure sub-case, and the hostility bit set "for every class" | B3 |
| `UNIT-VPLAYER-021` | High for construction, Medium for the row's semantics matching the session matrix | The hostility bit is read from a cached VIEW-SIDE 32-entry row on a non-`Player` record, filled `{0, 1, 0xa}`, agreeing with `AI-DIPLO-005`'s session values `{0, 1, 2}` on bit 0 alone; no instruction linking the two records was read | B3, DIV-259 |
| `UNIT-VISBIT-044` | (cited alongside `UNIT-VPLAYER-021`) | The visibility bit the interface keeps beside the hostility row | B3 context |
| `AI-CLICK-050` | partially retracted (the drag-discard clause; the cursor-arms clause stands) | Not read as authority for anything this story builds; excluded per the contract's own note | Not used |
| `SPR256-CURSOR-046` | High | The five `.256` minimap mode cursors: construction, the overview draw and the physical handler agree these are the minimap's own set | B2 |
| `SPR16A-CURSOR-067` | High | Slot order and names, including the eight arrow slots and their compass hotspots, and the address each of the 28 slots is stored at | B1, DIV-270 (identifying `L00628` as slot 3 `attack`) |
| `AI-KEYMOD-059` | High for the mechanism and the vkeys | The three cursor gates are modifier-key-held latches: `[L00627]` vkey `0x11`, `[L00670]` vkey `0x10`, `[L00632]` vkey `0x12`, set on key-down and cleared on key-up and on focus loss | DIV-262 (all three, where the row previously named two) |

## `AI-CURSOR-052` does not establish the `attack` cursor for an unmodified hover

Read whole again at the return of adversarial pass 1. The row covers ONE block of `R0219`,
`L00633`-`L07916`, and its own sentence about the hostility bit is that bit `0x4` "appears
only inside `mask & 0x24`, whose effect is to suppress the select cursor". Suppressing `select` is
not selecting `attack`. `spec.md`'s B3 and `hoverHostilityCursor`'s doc comment had cited the row
as the source of `attack`; both are corrected.

The write this build implements is in a second selection tree in the same routine,
`L01515`-`L13182`, which `EXP-0216` prints in `evidence/disasm-listings.txt`. `go run ./tools/claim
-k` returns no row for the five addresses that gate the write itself - `L01524`, `L01525`,
`L13183`, `L01527`, `L01526` - but a claim covers a different part of the same tree: see the
correction below. Read from that listing:

- `L01609` calls the hit test `R0218` and stores the mask in `[EBP-0x60]`.
- `L01524` tests `view+0x140`: with nothing selected the routine answers `select`
  (`L06217`) when `mask & 0x23` and `default` (`L01211`) otherwise, and exits.
- `L01525` tests `view+0x144 & 0x24` and answers the same pair.
- `L00635` tests bit `0x4`. Inside it: Alt (`[EBP-0x20]`) forces `move`; mask bit `0x20`
  answers `select`; otherwise `L01527` writes `L00628`, slot 3 `attack`.
- `L01694` is the non-hostile arm on `mask & 0x23`, ending in `select` at `L01411`.

`SPR16A-CURSOR-067` supplies every slot-address-to-name step above. `DIV-270` carries the reading
and the gates this build does not reproduce.

## `AI-CURSOR-209` covers part of the second tree, corrected at adversarial pass 2

`spec.md`, `missioncursor.go`'s doc comment, `closure.md` (three sites) and `DIV-270`'s own
research-basis cell all stated "no published claim covers this tree" as an unscoped fact about
`L01515`-`L13182`. The scoped search above (five addresses, no row) is true; the unscoped
sentence drawn from it is false. `AI-CURSOR-209` (Medium, from the same `EXP-0216`) reads
`L01682`-`L01410` and `L01412`-`L01411`, inside the same range, for the `town`
pick's own entry test (`[view+0x140]==1` at `L01405`, `[view+0x144]&0x1` at `L01406`,
both established fields per `AI-PANEL-061`) and its refinement (`+0x18c` bit `0x1`, hit-test bit
`0x800`, both Unknown in game terms).

Those instructions do not gate the `attack` write. Tracing the listing: `L13184` (`JZ L13183`)
and the fall-through from `L01410` both land at `L13183`, so the hostility test the `attack`
write depends on is reached whether the `town` entry test at `L01405`/`L01406` passed or
failed. The value it computes (`[EBP-0x64]`) is read again only later, at `L01412`, inside the
NON-hostile arm, to choose the `town` pick over the fallback - a branch this build's hostile-hover
write never reaches. `AI-CURSOR-209` therefore reads real instructions inside the tree without
covering the write this story implements; the five addresses that gate the write remain uncited
by any claim.

All five sites are corrected: `missioncursor.go`'s doc comment above `hoverHostilityCursor` and
`mapCursorPresent`, `spec.md`'s B3 section, this paragraph, `closure.md`, and `DIV-270`'s
research-basis cell, which now reads "a claim covers part of this tree" rather than "no published
claim covers this tree", and whose Notes cell no longer gives "the reading rests on an experiment
file, not a claim" as the reason the selection gate was not built - that reason did not hold once a
claim was found to cover part of the same tree. The reason it stays unbuilt this cycle is the
adversarial ceiling: building it now would be new behaviour at the story's last review pass, and
which of the two selection trees an ordinary hover reaches is a question open in a research lane.

## `AI-CURSOR-052` does not name the Shift latch that gates its own `select` write

In the same block, the `select` write at `L00637` is reached only when `[L00670]` is non-zero:
`L01534 CMP [L00670],0x0` and `L13185 JZ L07916` jump past the write when the latch is
clear. `AI-KEYMOD-059` gives `[L00670]` as vkey `0x10`. The row names `[L00627]` and
`[L00632]` and not this one, so the block's no-modifier outcome is "no cursor written" rather
than `select`. `DIV-262` carries this; `DIV-273`, held for it, was returned unused because it is
that row's own subject.

## `AI-CURSOR-191`'s own headline is corrected by the pin-bump rows

`AI-CURSOR-191`'s prose names `[EBP+0x140]` ambiguously between "armed mode" and "selection
count". `AI-CURSOR-202` (High) resolves the count reading: `[EBP+0x140]==0` gates on nothing
selected, forcing `sdefault`. The armed-mode value itself is a separately loaded field
(`AI-CURSOR-178`, not cited in `contract.md`'s own table but read for this reconciliation),
driving the eight-entry jump table proper. `spec.md`'s B2 uses the resolved reading.

## `UNIT-VPLAYER-021`'s retracted first reading

The contract's own warning — "will mislead you if you read only its headline" — refers to the
row's SUPERSEDED first version (`claims/retracted.md:277`), whose headline reads "the record is
NOT a `Player`" and whose own Unknown ("the two objects are held to be interchangeable... this
experiment did not resolve how") was doing real load-bearing work. The ACTIVE amended row
(`claims/unit.md:38`) is what `spec.md` cites: the record IS a `CPlayer` by `CRuntimeClass`, a
different, unregistered class from the simulation's own player object, and the `L01780`
comparison this project's earlier reading flagged Unknown is closed by `EXP-0110`
(`UNIT-VPLAYER-022`). Neither the retraction nor its closure changes this story's own conclusion
(the view-side row is Medium-linked to the session matrix on bit 0 alone); both were read before
`DIV-259` was written.

## Reading discipline followed

Every row above was read through the pinned reader. The two highest-risk rows — the one the
contract names as misleading by headline (`UNIT-VPLAYER-021`) and the one carrying the mask
arithmetic B3 depends on (`AI-CURSOR-052`) — were re-read in full at the landing, after the
implementation was written from an earlier reading, specifically to check the code comments and
`spec.md` against the claims' own text rather than against memory of them.

THAT RE-READ DID NOT CATCH THE `AI-CURSOR-052` DEFECT, and the reason is worth recording: the
row's text was quoted correctly in both places and the error was in what was concluded from it.
Reading a claim whole answers "does the document say this"; it does not answer "does this follow".
What caught it was the second instrument — opening the evidence file the claim's own experiment
ships and reading the listing under the sentence — which is `AGENTS.md`'s standing rule for a row
that states a branch, a count or a polarity, applied here to a row that states a scope. No confidence grade
above was rounded up; `AI-CURSOR-209`'s Medium grade and `UNIT-VPLAYER-021`'s Medium semantic
link are carried into `docs/DIVERGENCES.md` (`DIV-259`, `DIV-263`) rather than resolved by
assertion.
