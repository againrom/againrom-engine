# 0162-wear-rule — plan

## Approach

One new file, `pkg/data/wearrule.go`, holds the predicate and the code-to-row read (FR-1, FR-2,
FR-2a). Two call sites consume it: `mapWorld.enqueueEquip` in `pkg/game/world.go` (FR-3) and the
three cell builders in `pkg/game/shopview.go` (FR-4). `pkg/ui/shopscreen.go` gains one background
state so a cell can be grey and occupied at once. `pkg/sim` is not touched (FR-5).

## Decisions

**DD-1 The predicate lives in `pkg/data`, not in `pkg/game`.** What an item's row says about who may
carry it is item knowledge, and `pkg/data` already owns the item code's field layout, the three
collection classes and the row resolvers. Both consumers are in `pkg/game`, so the alternative was a
`pkg/game` helper beside `EquipTarget`; it was rejected because `pkg/data` is where the next consumer
would look, and because the predicate can then be tested against a bare `Collection` with no front
end, no table and no world (P-2).

**DD-2 The column constant is spelled here as well as in `spawn.go`, and the two are not shared.**
`spawn.go`'s `weaponCarrySlot = 0xf` is the same column read for a different question — whether a
dead body leaves a weapon behind. Exporting one constant for both would join two readings of one
column that were established by different evidence and could be narrowed separately. Each site
carries its own named constant and its own doc naming what it reads the column for.

**DD-3 A row too short answers "usable by neither"; a row that is not there answers "unknown".**
The two are different facts. A short row has been read and says nothing in its sixteenth cell, which
in the original leaves the item's descriptor bits clear and refuses everyone. A missing row has not
been read at all, and refusing on it would turn every gap in this build's item resolution into a
silent equip refusal — including item class 14, which this build resolves to no row. Unknown
therefore permits, and the permission is the pre-story behaviour.

**DD-4 The value is bit tested as it stands, including negative values.** The producers in the
original clear the descriptor byte and then copy bit 0 and bit 1 of parameter 15 independently. A
value of -1 therefore sets both bits and reads as usable by both. This is the instruction-level rule
of `ITEM-WEAR-055` and it is what the implementation follows. `ITEM-WEAR-056`'s prose describes the
one row that carries -1 (`rem`, weapon row 23) as carrying no parameter array; this build's own
measurement of both preserved roots disagrees with that description and agrees with `spawn.go`'s
existing reading of the same cell. See verification.md; the disagreement changes no shipped
behaviour, because no code path resolves that row.

**DD-5 Enforcement is at `enqueueEquip`, not in `sim.World.Equip` and not in `ui`.** The original
enforces in the client at the equipment doll's drop handler and nowhere else; its simulation reads
neither the character's class bit nor the item's column. `enqueueEquip` is this build's equivalent
seam: it is the one place a player's equip intent becomes a command, it already holds both operands
(`invParty.mage` and `invParty.table`), and a refusal there is a bare return, which is the
already-established shape of its three existing refusals. Putting the rule in `pkg/sim` would add a
rule the original does not have to the one place this project keeps deterministic and hashed;
putting it in `pkg/ui` would need the table and the party in the drawing tier, which no seam there
carries.

**DD-6 A fourth `ShopCellBack` rather than reusing `ShopBackEmpty`.** The original draws an unusable
cell on `backinvg.bmp`, the same file an empty cell gets. Reusing the existing empty state would
have given the right picture and the wrong meaning: `ShopCell.Occupied` is `Back != ShopBackEmpty`,
and every click, price plaque and count in the shop screen is gated on it, so an unusable item would
have become unbuyable and unclickable. `ShopBackUnusable` is added as a fourth value drawn from the
same file, and `Occupied` is restated as "not empty" over the two occupied states.

**DD-7 The affordable arm is nested inside the usable arm.** The original's cell painter reaches the
affordable background only on a shelf item whose price the purse covers **and** whose usability test
passes; otherwise usable gives the item background and not usable gives grey. So usability is asked
first at all three cell builders and affordability only after it, which makes "an unusable shelf item
is never drawn affordable" true by construction rather than by a second test.

**DD-8 The shop's subject is the first party member.** The original passes the shop screen's shown
party member. This build's shop screen shows one member's pack — `shopPackItems` reads
`shopParty()[0]` — and has no member selector, so the same member is the subject of the usability
test. When a member selector is added, that one call is where it lands.

## Risks

**R-1 A background index that outruns its art array.** `ShopCellBack` indexes `ShopScreenArt.Back`
directly. Growing the enum without growing the array and the file list would have drawn nothing
under an unusable cell on a real install, and the existing bounds check at the blit would have hidden
it. Both grow in the same commit and the release-gated shop test covers the drawn screen.

**R-2 Refusing more than intended.** If the code-to-row read answered "unknown" as a refusal, every
item this build cannot resolve would stop being equippable. DD-3 settles the polarity and SC-3
witnesses it directly, with a code whose field B names no collection.

## Traceability

| FR | Decisions | Criteria |
|---|---|---|
| FR-1 | DD-1, DD-4 | AC-1, SC-1 |
| FR-2 | DD-1, DD-2 | AC-2, SC-2 |
| FR-2a | DD-3 | SC-3 |
| FR-3 | DD-5 | AC-3, AC-4, SC-4 |
| FR-4 | DD-6, DD-7, DD-8 | AC-5, SC-5 |
| FR-5 | DD-5 | AC-6, SC-6 |

## Success criteria

**SC-1** `go test ./pkg/data -run Suitab` witnesses the eight-answer table.

**SC-2** With `AGAINROM_ASSETS` set, the census test reads both preserved roots' collections and
matches the counts and the four named rows.

**SC-3** A code whose field B is item class 14 answers unknown, and the equip interaction's
behaviour for it is unchanged.

**SC-4** The equip refusal and the two permitted equips are witnessed through `enqueueEquip` with a
built world, asserting the pending command list and the container.

**SC-5** The three shop cell builders are witnessed for a fighter and a mage subject over the same
stock.

**SC-6** `go test -count=1 ./...` is green in both repos, `formatVersion` is unchanged, and the
mission 10 and mission 20 unsupported-node counts are re-measured against the milestone baseline.
