# 1005 — the interactive doll — spec

This document is canonicalized to what the story actually builds, both rounds: FR-1 through FR-7
are the mission map screen (round 1), FR-8 is the ground drop and FR-9 is the town shop screen
(round 2). `contract.md` states the whole result and the claims it rests on.

## FR-1 — the per-pixel slot mask

The doll answers which equipment slot a pixel belongs to through a `SlotMask` (`pkg/ui/inventory.go`):

```go
type SlotMask struct {
	W, H int
	Slot []uint8
}
func (m *SlotMask) At(x, y int) (int, bool)
```

`Slot` is one byte per pixel, row-major (`y*W+x`), holding the 1..12 equipment slot whose own layer
painted the topmost non-transparent pixel there, or 0 for a pixel no layer touched — the base body,
or the fully transparent ground around the figure. `At` reports `(0, false)` for a nil mask, a point
outside `W`×`H`, or a zero-valued pixel, so a subject built with no mask at all (an enemy's flat
portrait, a world sprite) is not a special case for a caller.

The mask is built BESIDE the composed canvas, never by reading the finished picture back: both
compositors already know, at the instant they paint a layer, which slot that layer belongs to, and
that is the one fact a flattened picture's own pixels cannot recover.

- `composeInventorySubject` (`pkg/game/inventory.go`) and `composeUnitFigure` (`pkg/game/figures.go`)
  both build one, through a shared `paintFigureLayerMasked(dst, layer, mask, slot)` — a thin wrapper
  around the pre-existing `paintFigureLayer` that also stamps `mask.Slot[y*W+x] = slot` for every
  pixel it paints as opaque. `paintFigureLayer` itself, and the pixels it produces, are unchanged: it
  now calls the masked form with a nil mask.
- A fresh canvas earns a fresh mask, same bounds and all zero. The mage-cloak recombination
  (`figure directory .Mage()`, slot 8) rebuilds one when it rebuilds the canvas: the cloak paints
  first, then the body repaint marks its own pixels slot 0 — clearing the cloak's mark wherever the
  body covers it, since the body is what a viewer actually sees there.
- A paired slot's second sheet (`data.HasItemFigureSecondaryLayer`) is marked under the SAME slot
  number as its primary: one equipment slot, not two.
- `composeUnitFigure`'s signature widened to `(*image.RGBA, *ui.SlotMask)`. Every call site that
  never needed the mask (`composeDollEvidence`'s four digests, `chargen.go`'s preview, `speakers.go`'s
  dialogue faces) discards it with `_`. `mapWorld.unitFigure`'s own cache (`figurePics`) gained a
  sibling, `figureMasks`, keyed identically — populated but not yet consumed by any round-1 caller,
  since only the party's own subject (via `composeInventorySubject`) is interactive this round.
- `InventorySubject.SlotMask` carries the party subject's own mask. `buildInventorySubject` populates
  it at mission open (through `composeInventorySubject`); `mapWorld.refreshEquipment` propagates the
  recomposed mask alongside the figure and the slot icons it already copies, on every equipment
  change — a mask left standing from a stale composition would misname every pixel the moment the
  figure and the mask disagree.

**Not one drawn pixel changes.** `paintFigureLayerMasked` performs the identical copy
`paintFigureLayer` always did; the mask write is a second, parallel buffer. The existing figure
composition tests (draw order, mage-cloak layering, icon assignment) are unmodified and pass
unchanged.

## FR-2 — the doll's own hit test

`dollFigureSlotAt(x, y int) (int, bool)` (`pkg/ui/inventory.go`) is the doll box's hit test. It
centres the mask inside `dollFigureRect()` with the SAME arithmetic `drawInventoryPicture` uses to
centre the figure itself, so a hit test and a drawn pixel can never disagree about where the figure
stands. The answer is zero-based (0..11), `wornSlotRects`' own indexing, one below the mask's own
1..12.

**It is gated on `dollSubject`'s own arm-1 test, not on `inventoryEligible`** (round-2 adversarial
review, 2026-08-16, "the doll is drawn under one condition and hit-tested under a stricter one"):
`dollFigureSlotAt` calls `v.dollSubject()` directly and refuses unless it answers `true` with a
non-nil `figure` field — exactly the branch `dollSubject` takes to draw the composed subject picture
(`v.invHasSubject && v.invSubject.ID == present[0].ID && v.invSubject.Figure != nil`, `present` the
selection's own present-and-alive members in selection order), the ordinary figure or, under FR-6's
suppression, the lifted one. `dollBox` — and so `inventoryCaptures`, which decides whether a press on
this pixel belongs to the inventory window at all rather than the map underneath it — already follows
this same wide rule: ANY selection whose first present member is the subject, not only a selection of
that one member alone. Before this fix `dollFigureSlotAt` (and so the hover popup, FR-3, and the
press/drag origin, FR-4/FR-5, both of which call it) was gated on `inventoryEligible()` instead — the
worn box's and the pack bar's own narrower test, `len(present) == 1 && present[0].ID ==
v.invSubject.ID`. An ordinary box-select that caught the subject FIRST, alongside any other unit, left
the doll drawing the subject's own composed figure exactly as before while the hit test refused every
point on it: the press still landed inside `dollBox`'s own rectangle (so it was swallowed away from
the map) and then answered nothing there, `DIV-085`'s stated property broken again — the picture on
screen correct, only the gate wrong.
`TestDollFigureSlotAtAnswersUnderAMultiUnitSelectionLedByTheSubject` (`pkg/ui/doll_test.go`) drives a
two-unit selection (`v.sel = selection{5, 6}`, entity 5 the subject and first) through
`dollFigureSlotAt`, `hoveredItemInfoAt` and a full press-release gesture
(`dollPress`/`tapAt`/`TakeInventoryDollUnequip`) — the same production entry points every other doll
test in that file drives, not a hand-set eligibility flag — and a companion case
(`v.sel = selection{6, 5}`, the subject second) confirms the fix answers only when the SUBJECT leads,
not merely when the subject rides anywhere in the selection. No shipped test asserted the narrow
gate's own behaviour under a multi-unit selection specifically before this fix: the full `pkg/ui`
suite passed unchanged both before and after the gate was widened, since every pre-existing doll test
used either a single-unit selection or a selection moved to one different unit entirely, never a
multi-unit selection led by the subject.

**The worn box and the pack bar keep `inventoryEligible`'s narrower gate, deliberately** — unchanged
by this fix. `wornBox`'s own doc (`pkg/ui/inventory.go`) states the reason directly: both compose
PER-SLOT ICONS this build only ever builds for the party's own subject alone (`shopPackItems`' own
precedent one screen over), so a multi-unit selection leaves neither box anything correct to draw,
where the doll's own composed FIGURE is exactly present[0]'s own subject picture whether or not anyone
else rides beside it in the selection. Only the doll's own hit test needed widening; `dollBox`'s own
visibility test was already this wide, which is the asymmetry the review found.

**Which mask it reads follows which picture the doll box is actually drawing.** For an ordinary
figure it is `v.invSubject.SlotMask`; while FR-6's suppression is active for the same entity — a
drag holding one of the doll's own slots — it is `v.dollSuppressMask`, the mask
`composeInventorySubject` built beside the suppressed picture, never the unsuppressed one still held
on `invSubject`. This closes a round-1 review counterexample (2026-08-16): the suppressed picture and
`invSubject.SlotMask` disagree for the whole slot the drag lifted, and reading the wrong one during a
drag named a slot the drawn picture no longer shows there.

A press or a hover over a doll showing an enemy's flat portrait or its own world sprite — neither of
which carries a mask — names no slot, on `SlotMask.At`'s own nil handling.

**The mission doll's own composed figure and its raw-array readers agreed on nothing for slot 1**
(round-2 adversarial review, second pass, 2026-08-16, counterexample 2). `buildInventorySubject`
(`pkg/game/inventory.go`) composes the FIGURE with a slot-1 fallback: a member whose worn array
leaves slot 1 empty but whose `PartyMember.Weapon` is set — his starting weapon reaching him only
through that field, never through `Worn[1]` — is drawn wearing it. `SlotInfo` (`openMission`), the
same field's seed at `switchInventorySubject`, and `refreshEquipment`'s change-guard all read
`mw.currentEquipment()` directly, the raw array with no fallback, so a companion in exactly this
state showed the weapon on the figure while its own hover text and drag-origin lookup answered
empty for slot 1, and `refreshEquipment` treated an equipment change TO that state as no change at
all — its own comment there previously read "composed from this exact live equipment above," which
was false whenever the fallback fired.

`mw.currentFigureEquipment()` (`pkg/game/world.go`) is `currentEquipment()` widened by the SAME
fallback rule `buildInventorySubject` already applies to the figure — the raw equipment cloned,
`SetCode(1, startWeapon.Code)` applied only when slot 1 reads unoccupied and
`mw.invParty.startWeapon` is set. `openMission`, `switchInventorySubject` and `refreshEquipment`'s
guard now build `SlotInfo` and compare equipment through it; `mw.invEquipment` (`rearm`'s own
tracker) and `mw.bodyEquipment` (`refreshAppearance`'s own tracker) keep reading the raw,
un-widened `currentEquipment()` deliberately — `rearm`'s stat recompute already credits the
fallback weapon through its own `everEquipped`-gated rule and would re-arm a deliberately
unequipped hero if seeded from the wider one, and `invWeaponEverEquipped`'s opening seed has to
read the raw array so the tracker still starts false for a member who has never touched slot 1.

**"Worn array" above meant `member.Worn` literally, and for a member who had crossed a mission
boundary that array was already stale** (round-2 adversarial review, twelfth pass, 2026-08-17, C1).
`member.Worn` is the `PartyMember`'s equipment array as read at its last assembly and never
rewritten again; `member.Carry.Equipped`, once set by `CarryParty`/`CarryRoster` at a previous
mission's finish or by an original-save restore, is what a prior mission actually left the member
wearing. `buildInventorySubject` read `member.Worn` directly for every slot but 1 (the slot-1
fallback above is layered on top of whichever array feeds it), so a member's mission-doll figure and
slot mask showed the equipment he opened the CURRENT mission carrying at assembly rather than what a
prior mission left him wearing. The shop and the town screen already preferred `Carry.Equipped`
through `mapload.EquipmentFromParty` (`pkg/mapload/loadout.go`).

**Preferring `Carry.Equipped` over `Worn` closed the cross-mission case and left a narrower,
same-mission one open, found by the twelfth pass's own scenario-level mutation-kill of its first
correction.** `member.Carry` is nil for the whole of the CURRENT mission — `CarryParty`/`CarryRoster`
only set it at that mission's own finish — so a party-record read, Carry-preferred or not, falls
back to `member.Worn` for any member still inside the mission that assembled him, and `member.Worn`
does not track a live `sim.KindEquip`/`KindUnequip` issued during that same mission. A save taken
mid-mission, after the doll changed a slot through the ordinary drag machine, then reloaded, opened
showing the SLOT THE MISSION STARTED WITH: the resumed world (`resumeWorld`, `resume.go`, replaces
the mission's own entity data before `openMission` runs) already held the correct, current
equipment, but a party-record read — Carry-preferred or raw — could not see it.
`scenarios/1005-doll-and-shop.json` step 87 (existing, pre-round-2) witnesses this: it saves and
reloads mid-mission and asserts a retired slot stays retired, and it failed under the
Carry-preferred-only correction with `figure slot 1 = 265, want empty`.

**Fixed by having the mission-doll compositor read the live world first**, not the party record:
`missionDollEquipment(w *sim.World, id sim.EntityID, member mapload.PartyMember) (eq data.Equipment,
fallback bool)` (`pkg/game/inventory.go`) reads `w.Equipped(id)` when the world already holds the
entity — which it does by the time this runs, at both a fresh mission open and a resume, since the
world is built (and, for a resume, overwritten with the saved binary state) before this composition
runs — and falls back to `mapload.EquipmentFromParty(member)` only when it does not. This agrees
with the Carry-preferred answer at a true fresh mission open, because `mapload/start.go`'s own
entity mint already applies the identical Carry-then-Worn preference to seed the world in the first
place; it additionally reflects a resumed save's own live state and any live in-mission change
regardless of which member is being composed. `buildInventorySubject` and `world.go`'s own
`mw.invComposedEquipment` seed (C1b, below) both call it with the same three arguments — the
mission's world, the subject's entity id, and the party record — rather than each restating the
read. `EquipmentFromParty`'s Carry preference remains unconditional per member as the fallback path,
so `CarryRoster`'s mid-mission-joiner branch — a joining unit's `Worn` comes from its roster template
while its `Carry.Equipped` is read from the live world at the same finish call — is closed by the
fallback alone for a caller with no live entity yet, and by the live-world read directly for one
already in the world.

**The mission doll's own dedicated tracker could not have witnessed the C1 defect above, and a
first attempt at a headless witness read the wrong field** (round-2 adversarial review, twelfth
pass, 2026-08-17, C1b). `headlessInventory`'s `Figure` field previously read `mw.invFigureEquipment`
— `refreshEquipment`'s own guard seed, re-derived from `mw.currentFigureEquipment()` (a live read
off the running `sim.World`) at every mission open and member switch. That field agrees with the
live world by construction and so can never disagree with a composition that itself drew the figure
from something OTHER than the live world, which is exactly the C1 defect's own shape. `Figure` now
reads a new tracker, `mw.invComposedEquipment` (`pkg/game/world.go`), seeded from
`missionDollEquipment`'s own return at `openMission` and `switchInventorySubject` — the same call,
over the same three arguments, `buildInventorySubject` itself makes — and moved forward inside
`refreshEquipment`'s guarded recompose branch together with `invFigureEquipment`, so the two agree
again once a live change is genuinely recomposed. Since `missionDollEquipment` itself now prefers
the live world too (above), this tracker and `invFigureEquipment` typically hold the same value in
practice; the two remain structurally independent trackers rather than one alias for the other,
because `invFigureEquipment` is defined by a DIFFERENT call (`currentFigureEquipment`, gated on
`mw.invWeaponEverEquipped`) that would keep answering correctly even if `buildInventorySubject`
stopped calling `missionDollEquipment` for its own figure — the one case `invComposedEquipment`
exists to catch. `HeadlessInventoryState.Equipment` is unchanged: it still reads
`mw.currentEquipment()`, the raw live array with no fallback, for a caller that wants that instead.

**Reachable in shipped content, not only in a fixture.** `rosterTemplate` (`pkg/mapload/spawn.go`)
sets `PartyMember.Weapon` from the joining unit's own weapon row unconditionally; `startingLoadout`
(same file) separately clears `worn[cellWeapon]` for any row whose name contains the substring
`NPC` (`ITEM-DEATH-012`'s own death-time rule, applied at spawn instead — `0132`'s own reasoning,
restated at `DIV-070`). `DIV-070` names mission 151's own companion row, `NPC_Scrakan`, a
persistent, playable ally who joins mid-mission: his `Worn[0]` is cleared by that rule while his
`Weapon` field is set by `rosterTemplate`, the exact "slot 1 empty, `Weapon != nil`" state this
fix addresses. `switchInventorySubject` — the unit-selection-driven doll subject switch, the
mission's own counterpart to the shop's `shopMemberIndex`/`shopStepMember` — reaches any present
party member, Scrakan included once he has joined, so the disagreement was reachable through
ordinary play and not only through a constructed fixture.

`TestSwitchInventorySubjectAppliesTheSameWeaponFallbackAsTheFigure` (`pkg/game/world_test.go`)
builds a one-member party whose `Worn` leaves slot 1 empty and whose `Weapon` is set, and checks,
independently, `openMission`'s own construction, `switchInventorySubject`'s `SlotInfo` and
`invFigureEquipment`, that `invEquipment` (the raw tracker) stays un-widened, and that
`refreshEquipment`'s guard does not erase `SlotInfo` over an unchanged fallback-worn slot 1.

**The fallback above could fire a second time for a member who had already unequipped the starting
weapon for real during the mission**, the same duplication the shop's own `shopSlot1Code` fix
(above) closed for the shop, reopened here for the mission doll (round-2 adversarial review, third
pass, 2026-08-16, counterexample 4). `currentFigureEquipment`'s own `!occupied && startWeapon != nil`
test says nothing about whether the fallback has already been spent: a live `sim.unequip` on slot 1
leaves the array cell empty again, which reads identically to "never equipped" unless something
else remembers the difference. Fixed: `mw.invWeaponEverEquipped` — the sticky per-mission tracker
FR-2 already carries for `rearm`'s own gate — is seeded through `weaponFallbackSpent(eq)`
(`world.go`), which answers true when slot 1 is occupied, and `currentWeaponFallbackActive` (the
shared condition `currentFigureEquipment`, `ui.InventorySubject.WeaponFallback`, and
`refreshEquipment`'s own change guard all read) adds `&& !mw.invWeaponEverEquipped` to its test.
Once the tracker has latched true it does not revert on a later unequip. The persisted latch
`PartyMember.WeaponMaterialized`, below, is what carries the same fact across a mission boundary and
into the shop.
`TestSwitchInventorySubjectAppliesTheSameWeaponFallbackAsTheFigure` extends past its four original
assertions with a genuine live-equipment scenario: `sim.Step` equips the SAME code the fallback would
have supplied into the real slot (`WeaponFallback` must go false, and does — the figure's own drawn
code is unchanged across this transition, so `refreshEquipment`'s guard has to compare
`WeaponFallback` as well as the equipment value, not only the equipment value, to notice), then
`sim.Step` unequips it again (`WeaponFallback` must stay false, `invFigureEquipment` must not
reintroduce the starting weapon, `SlotInfo[0]` must go empty) — reverting either the tracker's own
seed or the guard's added `WeaponFallback` comparison reddens this scenario, where the earlier four
assertions alone could not, since a simple fixture with no live equip/unequip leaves the value
correct through more than one path.

**The suppressed doll picture — the figure a drag draws once armed, FR-6 below — read the same raw,
un-widened `currentEquipment()` while the ordinary figure (`refreshEquipment`) had already moved to
`currentFigureEquipment()`** (round-2 adversarial review, third pass, 2026-08-16, counterexample 3):
a drag held on any slot OTHER than 1, for a member whose own slot 1 is fallback-only, suppressed a
figure missing the weapon layer the ordinary figure still drew the tick before — the same
hit-test-vs-picture disagreement this file's own round-1 review fix (above) closed for the mask,
reopened for the drag-held figure. `refreshDollDrag` now clones `currentFigureEquipment()` rather
than `currentEquipment()` before clearing the dragged slot and recomposing. `Viewer`'s own drag
state (`dragActive`, `dragCandKind`, and the rest) is unexported and set only from real gesture
frames through `command()`, which `pkg/game` cannot drive directly — no exported API arms a doll
drag from outside `pkg/ui`. `refreshDollDrag` delegates its composition to the production seam
`suppressedDollSubject`. `TestRefreshDollDragEquipmentSourceAgreesWithTheOrdinaryFigure`
(`pkg/game/world_test.go`) calls that seam over a live `mapWorld` and checks the fallback weapon's
independent mask pixel and blue figure pixel remain while slot 2 is suppressed. Replacing the
seam's `currentFigureEquipment()` read with `currentEquipment()` reddens both assertions. The test
does not reproduce the source choice in test code, and no test-only pointer-state setter is
exported.

**A fallback-only slot 1 drew a weapon layer indistinguishable from a real one once the picture
above widened, but nothing in the live entity's equipment array or any container backs it, so
`sim.unequip` and `dropFromEquipment` both refuse it outright** (counterexample 4, same review). A
press-drag or a tap-to-unequip on that slot armed exactly as if the weapon were real, then resolved
into a command the simulation silently ignores. Fixed on both sides: `ui.InventorySubject` gained
`WeaponFallback bool` (`pkg/ui/inventory.go`), set from `mw.currentWeaponFallbackActive()` at the
same three sites that already build `SlotInfo` from `currentFigureEquipment` (`openMission`,
`switchInventorySubject`, `refreshEquipment`'s guarded recompute) — and, self-contained, at
`buildInventorySubject`'s own construction (`pkg/game/inventory.go`), so a caller that opens a
mission before `mw.invParty` exists still gets a subject that correctly refuses to arm a drag it
cannot act on. `buildInventorySubject` reads `!member.WeaponMaterialized` (below) rather than a pack scan of its
own; the inline scan this paragraph originally described was replaced at the fifth pass, next.
**A fallback-only slot 1 is taken off like any other slot, and the item is materialized rather than
refused** (owner directive, 2026-08-17: pressing an item on the doll takes it off). The earlier
behaviour — `command.go`'s press-arm block refusing `dragCandKind, dragCandIdx = dragFromDoll, slot`
for `slot == 0 && v.invSubject.WeaponFallback` — is removed: `pkg/ui` arms a drag and a tap on that
slot exactly as it does on a real one, and the mission side answers. `mw.materializeFallbackWeapon()
(int, bool)` (`pkg/game/world.go`) is that answer, called from `enqueueUnequip(0)` and from
`enqueueDrop`'s own `worn` arm before either issues its simulation command: while
`currentWeaponFallbackActive()` holds, it appends the starting weapon's own code to the entity's
CONTAINER through `ReplaceStock`, raises the latch through `materializeStartingWeapon`, sets
`mw.invWeaponEverEquipped`, and returns the container element index. An unequip then needs no
simulation command at all, since the item is already where an unequip would have put it; a ground
drop issues `KindDropCarried` for the element just appended. It writes the container and not the
equipment array deliberately: filling slot 1 would re-arm the member's stats through `rearm` and
would restore the corpse drop that `startingLoadout`'s own NPC rule exists to suppress (`DIV-070`).
This is the mission-side restatement of the shop's own rule, where taking a fallback weapon off has
always produced a real item. Witnessed by `TestAFallbackOnlySlotOneIsTakenOffLikeAnyOther`
(`pkg/ui/doll_test.go`, a tap and a drag-to-pack both raise `TakeInventoryDollUnequip`),
`TestTakingTheDrawnStartingWeaponOffPutsARealOneInThePack` and
`TestDroppingTheDrawnStartingWeaponReachesTheGround` (`pkg/game/weaponlatch_test.go`).

**The fallback's own materialization is HISTORY, not a value the array's own present contents can
re-derive, and every consumption site before this pass re-derived it anyway** (counterexamples A
and B, round-2 adversarial review, fifth pass, 2026-08-16). `weaponFallbackSpent`,
`shopMemberPack`'s own pack scan, and `buildInventorySubject`'s own inline pack scan each asked
"does the entity's present pack or array already hold the starting weapon's code" as a stand-in for
"has the fallback ever materialized," which is a different question: once a materialized starting
weapon is later carried out of the party's own pack entirely — sold, dropped, or moved off the
member across a mission boundary — the present-state scan again reads "not present," so the SAME
member the fallback had already spent could re-trigger it and hand back a second copy on the next
composition, a cross-mission and mid-mission gap none of the three call sites' own static fixtures
exercised. Fixed with a persisted latch: `PartyMember.WeaponMaterialized bool`
(`pkg/mapload/start.go`) survives `clonePartyMember`'s plain value copy and `Snapshot.Party`'s gob
encoding with no added plumbing — the same field `OwnParty` (`pkg/mapload/party.go`) carries through
its per-member copy loop, breaking no cross-member alias, exactly as every other scalar field on
`PartyMember` already does (`TestOwnPartyCarriesWeaponMaterializedAsAPlainValue`,
`pkg/mapload/party_test.go`).

**No pack scan seeds the latch.** A scan of the container was the seed at the fifth pass and it is
removed (seventh pass, 2026-08-17): a container holds CODES and not identities, so a second unit of
the same code — a shipped duplicate, loot, a purchase — read as "the fallback has already
materialized" and retired a fallback that had never fired. `weaponFallbackSpent(eq data.Equipment)
bool` (`pkg/game/world.go`) now has one term, whether slot 1 is occupied, and takes no item stacks.
`resolveWeaponMaterialized(member *mapload.PartyMember, eq data.Equipment) bool` (`pkg/game/world.go`)
is where the persisted bit and that present-state observation combine: it returns the persisted bit,
having first raised it when slot 1 is occupied for real. `missionPartyMember(id sim.EntityID)
*mapload.PartyMember` (`pkg/game/world.go`) resolves a live entity id to its `*mapload.PartyMember`
inside `mw.mission.party` by scanning `mw.mission.ids`. Three consumption sites read the persisted
bit rather than re-deriving it: `currentWeaponFallbackActive` (`pkg/game/world.go`,
`&& !mw.invWeaponEverEquipped` from the third pass, above, generalizes into this bit's own
per-mission latch), `shopWeaponFallbackCode` (`pkg/game/shoproom.go`), and `buildInventorySubject`
(`pkg/game/inventory.go`).

**One writer, and a test that keeps the enumeration true.** `materializeStartingWeapon(member
*mapload.PartyMember)` (`pkg/game/weaponlatch.go`) is the only assignment to the latch in package
`game`. `TestWeaponMaterializedHasOneWriter` (`pkg/game/weaponlatch_scan_test.go`) parses the
package's own non-test source and fails on an assignment anywhere else, on a mention inside a
function its table does not name, and on a table row no longer touching the latch. The table and
`closure.md`'s write/read enumeration are therefore edited together, which a comment could not
enforce: four adversarial passes were spent on sites that each set this bit their own way.

**A mid-mission write reaches the save.** `FrontEnd.liveDriver` (`pkg/game/resume.go`) previously
stored a CLONE of the party beside the live mission, so `Snapshot` wrote the clone and every
mid-mission write to a `*PartyMember` — the latch included — was absent from a save taken during the
mission. It now aliases the caller's own slice and names it in place through
`mapload.NameParty(party []PartyMember)` (`pkg/mapload/party.go`), the identity assignment lifted out
of `OwnParty`, which is now `CloneParty` plus `NameParty`. Both production callers pass a slice they
own: `frontend.go:1032` passes none, and `frontend.go:1293` passes `ms.Party`, the same slice the
mission writes. Witnessed by `TestAMidMissionLatchReachesTheSave` (`pkg/game/weaponlatch_test.go`).
`TestBuildInventorySubjectWeaponFallbackStaysOffOnceMaterialized` (`pkg/game/inventory_test.go`)
builds a member with `WeaponMaterialized: true` and no code in the pack, and asserts the composed
subject shows no fallback and no unread layer — mutation-witnessed by reverting the read back to
`member.Weapon != nil` alone, which reddens with an `unread` list naming the weapon's own layer and
icon addresses.

**`refreshAppearance` composed the doll's own map sprite from the raw, un-widened
`currentEquipment()` while the inventory doll had already moved to `currentFigureEquipment()`**
(counterexample I, round-2 adversarial review, fifth pass, 2026-08-16): a member whose slot 1 is
fallback-only showed the starting weapon on the inventory doll (FR-2, above) and not on his own map
sprite, the DIV-070 companion population this file already names reachable in shipped content.
`currentFigureEquipment`'s own doc comment undercounted its narrow callers at two
(`rearm`, `currentWeaponFallbackActive`'s own internal check) where four remained: `openMission`'s
and `switchInventorySubject`'s own seeds are narrow for the same reason `invEquipment` is (they feed
`resolveWeaponMaterialized`'s own present-state observation, which must read the raw array), and
`rearm` stays narrow because it resolves the fallback through its own internal `everEquipped`
parameter — widening its input would double the fallback into `Rearm`'s stat computation. Fixed:
`refreshAppearance` (`pkg/game/world.go`) now reads `mw.currentFigureEquipment()`, matching
`HeroBodyFor` (`pkg/data/equip.go:206`)'s own precedent for composing a body from widened equipment.
`mw.bodyEquipment`, `refreshAppearance`'s own change-tracker, is seeded from the same widened source
at both of its seed sites (`openMission`'s conditional seed, `switchInventorySubject`'s unconditional
seed) for consistency with the comparison `refreshAppearance` itself now makes.
`TestRefreshAppearanceAgreesWithTheDollOverAFallbackWeapon` (`pkg/game/world_test.go`) builds a
fallback-only companion, calls `refreshAppearance`, and asserts `mw.bodyEquipment` reports the
fallback code occupying slot 1 and equals `mw.invFigureEquipment` exactly — mutation-witnessed by
reverting `refreshAppearance`'s read back to `currentEquipment()`.

**`rearm`'s own `everEquipped` parameter (above) is not the only caller of
`ResolveEquipmentLoadout`, and the other one read the latch incorrectly** (adversarial review,
2026-08-17, C3): `mapload.PartyLoadout` (`pkg/mapload/loadout.go`) resolves a member's opening
`Loadout` at every mission's own construction (`PartySpawnWithTable`, `pkg/mapload/start.go`),
feeding the entity's `CombatBlock` at the mint that follows — a SEPARATE call from `rearm`'s own
mid-mission recompute, over a `PartyMember` value rather than a live `sim.World`. It passed
`everEquipped` as an unconditional `false`, on the premise that mission construction is always a
member's first resolution — true before `WeaponMaterialized` existed, false once the latch can
ride in from an earlier mission through `mapload.CarryParty`. A member who took the starting
weapon off and sold it kept arriving at his next mission's own mint armed with its damage: an
empty, latched slot 1 still read as "not yet materialized" instead of "taken off." Fixed:
`PartyLoadout` now passes `p.WeaponMaterialized`, the same bit `rearm`'s own everEquipped already
reads correctly, so mission construction and a live mid-mission recompute apply the identical rule.
`TestWeaponMaterializedHasOneWriter` (`pkg/game/weaponlatch_scan_test.go`) now parses
`pkg/mapload` as well as package `game`, so `PartyLoadout` is a catalogued reader rather than an
invisible one. `TestPartyLoadoutRespectsTheMaterializedLatch` (`pkg/mapload/loadout_test.go`)
derives two members through `PartySpawnWithTable` — one never latched, one latched with an empty
slot 1 — and asserts their `Combat.DamageBase`/`DamageSpread` differ; mutation-witnessed by
reverting `PartyLoadout`'s `everEquipped` argument back to the literal `false`.

## FR-3 — hover shows the item's popup

`hoveredItemInfoAt` (`pkg/ui/itempopup.go`) asks the doll box between the worn box and the pack bar,
through `dollFigureSlotAt`, and shows the same `InventorySubject.SlotInfo[slot]` line the worn box
already shows for that slot. A point over the figure's transparent pixels or its background shows
nothing, matching every other point outside a populated cell.

## FR-4 — a plain press takes an item off

A press released on the SAME slot it began on, without the gesture ever crossing `TapSlop`, raises
`TakeInventoryDollUnequip()` — a third one-shot request field (`invDollUnequipRequest`), the same
"index+1 so zero means nothing pending" shape as the existing `invUnequipRequest`. `pkg/game`'s
`unequipFromDoll()` drains it every `frameAdvance`, beside `unequipFromWorn()`, into the SAME
`mw.enqueueUnequip(idx)` both drains already call — no second rule for what an unequip means.

## FR-5 — the drag machine

A press on a pack-bar cell or a doll slot is a drag candidate (`v.dragCandKind`,
`dragFromPack`/`dragFromDoll`, `pkg/ui/command.go`), captured at the press and reset to `dragNone` at
the top of every new press. The worn box is deliberately NOT a drag source in round 1: it has no
per-cell picture of its own beyond the icon the doll's own slot already carries.

The drag arms — `v.dragActive = true`, the candidate's own icon copied into `v.dragIcon`
(`Pack[idx]` or `Slots[idx]`) — once the gesture crosses `TapSlop`, read off `v.dragMoved`, the SAME
accumulator `dragIntent` (`viewer.go`) already raises for the map's own box-select, every tick
`PrimaryDown` holds, regardless of where the press landed. No second distance measurement exists.

While armed, the carried icon is drawn on the cursor: `dragItemPresent()` returns the icon centred on
`(v.cursorX, v.cursorY)`, and `Viewer.Draw` blits it as the last pass, over everything else.

**Release resolves into exactly one command, or none, and the world is never mid-drag** — nothing is
removed from the pack array or the world's equipment before a release, so "return to origin" costs no
statement:

- `dragFromPack` released with the cursor inside `dollBox()` raises `invEquipRequest` — the SAME
  request a double-click on that pack cell already raises, drained by the existing `equipFromPack`,
  wear rule included.
- `dragFromDoll` released with the cursor inside `packBar()` raises `invDollUnequipRequest`.
- Released anywhere else: neither request is raised. `v.dragActive`, `v.dragCandKind` and
  `v.dragIcon` are cleared unconditionally at every release, so the next frame's ordinary
  presentation already shows the item where it started.

**"Every release" had one exception: a release over the HUD toggle bar** (round-2 adversarial
review, third pass, 2026-08-16, counterexample 5). `hudToggleCaptures`'s own `PrimaryReleased`
branch, the one early-return path in `command()` that answers before the inventory's own release
handling above runs at all, cleared `v.held` and returned without touching `dragActive`,
`dragCandKind`, `dragIcon` or `invGrab`. An armed drag released over the bar stayed armed on every
later frame: `invGrab || ...` kept reading true regardless of where the cursor went next, so the
NEXT primary release anywhere off the inventory reached the drag-release switch above with a stale
origin and resolved as a ground drop at that unrelated release's own cell — a cell the player never
aimed the original drag at. Fixed: the same four fields are reset, unconditionally, in the toggle
bar's own branch too, matching "released anywhere else" for a HUD surface the inventory's own boxes
do not claim. `TestReleasingAnArmedDollDragOverTheHudToggleBarCancelsIt` (`pkg/ui/doll_test.go`) arms
a drag off the bar, releases on the bar, and asserts a later, unrelated release raises no stray drop
— mutation-witnessed by removing the four reset lines.

Ground drop (a release outside every inventory box, during a mission) is FR-8, below.

## FR-6 — the doll redraws without the dragged layer

Per `DLG-FIGURE-021`, the compositor takes no layer selector, so suppression is done by composing a
DIFFERENT equipment set — the dragged slot's own code cleared — never by teaching the compositor a
layer to skip.

- `Viewer.DraggedDollSlot() (int, bool)` is a level (not one-shot) read of the doll-sourced drag's own
  slot, for as long as one stands.
- `mapWorld.refreshDollDrag()` (`pkg/game/world.go`), run every `frameAdvance` after
  `refreshEquipment`, tracks the last slot it told the viewer about (`invDollSuppressSlot`,
  `refreshEquipment`'s own guard-then-skip shape restated for one integer): unchanged since last
  frame costs one compare and no recompose. On a change, it clones `currentFigureEquipment()` — the
  fallback-widened source `refreshEquipment` itself composes the ordinary figure from, not the raw
  `currentEquipment()` (round-2 adversarial review, third pass, 2026-08-16, counterexample 3, FR-2
  above), `SetCode(slot, 0)`, recomposes through `composeInventorySubject`, and pushes BOTH the composed
  picture and `composed.SlotMask` through `Viewer.SetDollSuppressedFigure(owner, slot, pic, mask)` —
  the mask travels with the picture it was built beside, the same pairing `refreshEquipment` already
  keeps for the ordinary figure.
- `dollSubject()` (`pkg/ui/inventory.go`) is the one place the substitution happens: for the SAME
  entity the suppression was pushed for, with a non-zero slot and a non-nil picture, it returns the
  suppressed figure instead of the ordinary one. `dollSource.suppressSlot` is part of the doll's own
  comparable cache key, so a suppressed composition and the ordinary one are distinct cache entries —
  a drag beginning or ending on the exact frame the doll would otherwise treat as unchanged still
  forces a rebuild. `dollFigureSlotAt` (FR-2) makes the SAME three-field check against
  `v.dollSuppressOwner`/`v.dollSuppressSlot`/`v.dollSuppressPic` to decide whether to read
  `v.dollSuppressMask` in place of `v.invSubject.SlotMask`, so the hit test and the hover popup agree
  with whichever picture `dollSubject` is drawing this frame.

## FR-7 — nothing else changes

- Worn-box double-click unequip, pack-cell double-click equip, and pack-bar scrolling are
  unmodified and their own pre-existing tests are unchanged and pass.
- A drag beginning inside an inventory box and carried, past `TapSlop`, over the map does not start a
  box-select: `v.invGrab` — set at press when the press lands inside `inventoryCaptures`, and read at
  the TOP of `command()` before `decide()` (the map's own box-select build) is ever reached — keeps
  the WHOLE gesture inside the swallow branch for as long as it began there, regardless of where the
  cursor later moves. This is pre-existing behaviour (story 0140's own hotfix), unmodified by this
  story; it is witnessed here because the drag machine is the first round-1 gesture whose cursor
  routinely leaves the box it began in.
- **`inventoryCaptures` reserves the worn box's and the pack bar's own screen ground when a
  multi-unit selection led by the subject leaves neither drawn** (round-2 adversarial review, third
  pass, 2026-08-16, counterexample 6). `dollBox` follows `dollSubject`'s own wide rule (any selection
  whose first present member is the subject draws the doll, FR-2's own round-2 review fix, above);
  `wornBox` and `packBar` follow `inventoryEligible`'s narrower rule (exactly one selected entity)
  and draw nothing under a multi-unit selection. `inventoryCaptures` asked `wornBox`/`packBar`
  directly, so the pixels those two boxes occupy on every OTHER selection stopped capturing input
  the moment a multi-unit selection left them undrawn: a release there read as "outside every
  inventory box" and fell through to FR-8's ground drop at a cell the player was never looking at,
  while a doll-origin drag could still be armed in the same selection state. Fixed, narrowly rather
  than by widening the check unconditionally: `wornBoxArea`/`packBarArea` (`pkg/ui/inventory.go`) are
  `wornBox`/`packBar`'s own rectangles with the `inventoryEligible` half of the gate dropped;
  `inventoryCaptures` still asks `dollBox`, `wornBox` and `packBar` first, exactly as before, and
  only when none of the three answers does it ALSO ask the two area rectangles — gated behind
  `dollInventoryActive()`, which is true only when the doll box is on AND `dollSubject` is actually
  drawing the inventory subject's own composed figure (not an enemy portrait, not an empty
  selection, not a different leading member). Every selection state that is not this specific
  subject-led multi-selection keeps `inventoryEligible`'s existing contract unchanged: the
  invisible worn box and pack bar stay map ground, exactly as `wornBox`/`packBar` already say they
  should. `wornBox`/`packBar` themselves — and every caller that composes their CONTENT rather than
  only asking whether a press lands on them — keep the narrow gate unchanged: neither box has
  anything correct to draw for a multi-unit selection, only the ground under it stays the
  inventory's own, and only while the doll itself is genuinely interactive there.
  `TestInventoryCapturesTheReservedGroundOnlyWhileADollDragIsInFlight` (`pkg/ui/doll_test.go`) sets
  a two-unit selection led by the subject, confirms `dollBox` true and `wornBox`/`packBar` false,
  confirms `packBarArea` true, asserts that with no drag armed the same point is NOT captured (a
  plain press there is an ordinary map gesture), and asserts a real drag release over that ground
  raises no stray ground drop — mutation-witnessed by reverting `inventoryCaptures` to ask
  `wornBox`/`packBar` directly, which reproduces the stray drop.
  `TestInventoryDoesNotCaptureInvisibleAreasWithoutASelection` is the negative control: with no
  selection at all, `packBarArea` still answers a rectangle (the switch alone), but
  `inventoryCaptures` over that same ground answers false — `dollInventoryActive` refuses, since
  `dollSubject` draws nothing to press. This is the scope boundary the naive "always ask the area"
  fix would have missed: an invisible pack bar is map ground everywhere except the one selection
  state this counterexample names.

- **The gate above still over-captured: `dollInventoryActive()` alone reserves the wide areas for
  EVERY press under a subject-led multi-selection, not only while a doll-origin drag is actually in
  flight** (counterexample C, round-2 adversarial review, fifth pass, 2026-08-16). A plain map move
  order — a press-and-release with no drag ever armed — under that same selection state landed on
  `wornBoxArea`/`packBarArea`'s own pixels, was swallowed by `inventoryCaptures`, and never reached
  the map's own order handling: the third pass's own fix widened the SURFACE correctly but the
  GATE too far, since the counterexample it answered was a drag release, not an ordinary press.
  Fixed: `inventoryCaptures`' wide-area branch now also requires `v.dragCandKind == dragFromDoll` — a
  doll-origin drag candidate captured at THIS press, not merely a selection state that could carry
  one — narrowing the reservation to the one interaction it exists for.
  `TestInventoryCapturesTheReservedGroundOnlyWhileADollDragIsInFlight` (`pkg/ui/doll_test.go`,
  replacing the third pass's own test of the same area) asserts both halves: no drag in flight, the
  wide areas do not capture and a plain move order reaches the map; a doll-origin drag armed first,
  the wide areas capture and the release resolves into `invDollUnequipRequest` — mutation-witnessed
  by reverting the added `dragCandKind` clause, which reddens the first half.

- **`dragFromDoll`'s own release arm asked the NARROW `packBar()` rather than the WIDE
  `packBarArea()` it needs under the same subject-led multi-selection**
  (counterexample D, same review pass). `command.go`'s release switch resolved a doll-origin drag by
  checking whether the release point falls inside `v.packBar()` — the box-drawing rectangle, which
  `wornBox`/`packBar`'s own narrow `inventoryEligible` gate leaves empty under a multi-unit
  selection — so a recognised, correctly-captured gesture (counterexample C's own fix reaches this
  release at all) still resolved to nothing: `invDollUnequipRequest` was never raised, and the item
  stayed on the doll with no ground drop either, since the release point was never "outside every
  inventory box" in the first place. Fixed: the release arm now asks `v.packBarArea()`, the same wide
  rectangle `inventoryCaptures` itself now gates on. `TestInventoryCapturesTheReservedGroundOnlyWhileADollDragIsInFlight`'s
  own drag-in-flight half above is this counterexample's own witness, since a release that reaches
  the switch at all but resolves to nothing is indistinguishable from one `inventoryCaptures` never
  captured without checking which specific request came out — mutation-witnessed by reverting the
  release arm to `packBar()`, which reddens the `invDollUnequipRequest` assertion while leaving
  `inventoryCaptures` itself green.

- **A tremor inside the SAME cell or slot a gesture began on could still arm a drag, and an armed
  drag's release switch tried `v.dragActive` before checking whether the release named the drag's
  own origin** (counterexamples E and F, same review pass). `TapSlop` is a 4-pixel Manhattan
  threshold accumulated over the whole press; a pack cell is `invCellSize` (48) pixels wide and a
  doll slot's own mask region is comparably large, so a hand's own tremor crosses `TapSlop` while the
  gesture never leaves the cell or slot it began on. Before this fix, `command.go`'s release switch
  tried `v.dragActive` first: once the tremor armed the drag, the release fell into the drag-release
  resolution above, which does not recognise "released back on its own origin" as a case (the
  origin is neither `dollBox` nor outside every box), and raised neither an equip, an unequip nor a
  drop — losing a double-click's second press (E) or a doll tap-to-unequip (F) outright, the same
  `dest == origin` tremor guard the shop's own drag machine already carries
  (`shopReleaseIsOrigin`, `pkg/ui/shopscreen.go`, called from `app.go`'s release handling), never
  applied to the mission map's own pack cell or doll slot. Fixed with one mechanism for both cases:
  `command.go`'s release handling computes `originSame` once per release — true when the release
  names the SAME pack cell (`inventoryPackCellAt`) or doll slot (`dollFigureSlotAt`) as the drag
  candidate's own origin (`dragCandIdx`) — and two new leading switch cases resolve a same-origin
  release into the tap outcome (`invEquipRequest` for a pack cell under `invEquipTap`,
  `invDollUnequipRequest` for a doll slot) ahead of the drag-release case, regardless of whether the
  tremor armed the drag. `TestADoubleClicksSecondPressStillEquipsAfterATremorWithinTheCell`
  (`pkg/ui/invequip_test.go`) and `TestDollTapStillUnequipsAfterATremorWithinTheSlot`
  (`pkg/ui/doll_test.go`) each press, move 3-4 pixels within the same cell or slot (confirming
  `dragActive` armed first, so the fix is proven against the harder case), release on the origin,
  and assert the tap outcome — not a drop — comes out. Counterexample F's own doll-slot case is
  pre-existing on `master`, not introduced by this story: `TestDollTapStillUnequipsAfterATremorWithinTheSlot`
  builds a bespoke 20x20 single-slot `SlotMask` because the shared 4x4 `dollMaskFixture` other doll
  tests use is too small to hold a tremor's own few pixels of travel without leaving the slot
  entirely. Both tests are mutation-witnessed by removing the `originSame` computation and its two
  switch cases, which reddens both.

- **The doll's own `originSame` case above did NOT resolve "regardless of whether the tremor armed
  the drag", once a doll drag actually armed** (round-2 adversarial review, tenth pass, 2026-08-17,
  counterexample 1: "the doll tremor guard is inert in the running game"). The paragraph above was
  written against `TestDollTapStillUnequipsAfterATremorWithinTheSlot` as it stood at the ninth pass,
  which never called `SetDollSuppressedFigure` and so never installed the state a live drag installs.
  In production, `refreshDollDrag` (`pkg/game/world.go`) composes and pushes a suppressed mask with
  the origin slot's own code cleared on every frame the drag stays armed, and `dollFigureSlotAt`
  (`pkg/ui/inventory.go`) reads that suppressed mask whenever the suppression is in force for the
  subject — which command.go's own `originSame` doll case did too, so it asked the origin's identity
  against a mask that could never again answer the origin's own slot for as long as the drag stayed
  armed. Measured directly against the running mechanism (mission and shop alike; the shop's own
  restatement is below): a release back on the exact slot the press began on, after the drag had
  crossed `TapSlop`, raised neither `TakeInventoryDollUnequip` nor `TakeInventoryDrop` — the gesture
  vanished. Fixed by asking a DIFFERENT question at the origin-identity compare than `dollFigureSlotAt`
  asks everywhere else: `Viewer.dollFigureSlotAtUnsuppressed(x, y int) (int, bool)`
  (`pkg/ui/inventory.go`) reads `v.invSubject.SlotMask` directly, never the suppressed one, and
  `command.go`'s `originSame` doll case now calls it instead of `dollFigureSlotAt`. The origin's own
  identity does not change because the drawn picture did.
  `TestDollTapStillUnequipsAfterATremorWithinTheSlot` now pushes a real
  `SetDollSuppressedFigure(sub.ID, 1, pic, suppressedMask)` between the arming move and the release
  under test, with a setup assertion that `dollFigureSlotAt` answers no slot at the origin once that
  push lands — confirming the fixture actually installs the state production installs, rather than
  merely recording that a call was made. Mutation-witnessed by reverting the origin-identity call from
  `dollFigureSlotAtUnsuppressed` back to `dollFigureSlotAt`, which reddens the test
  (`TakeInventoryDollUnequip = (0,false)`, want `(0,true)`).

  A dead duplicate of the doll's own plain-tap case, `case v.dragCandKind == dragFromDoll:` in
  `command.go`'s own release switch (below the drag-release case), was removed in the same pass: it
  re-tested the identical condition `originSame` already resolves ahead of it in the same switch, was
  reachable only when `originSame` had already answered false for that exact comparison AND the drag
  had never armed (in which state `dollFigureSlotAt` and `dollFigureSlotAtUnsuppressed` read the
  identical unsuppressed mask, since a drag that never armed never pushes a suppression), and so could
  never itself resolve differently from `originSame`. It carried the comment "THE PLAIN TAP (1005
  spec, 'press to take off')" — this story's primary gesture — on code that never ran; the adversarial
  reviewer confirmed this by replacing its body with `_ = 0` and observing the whole suite stay green.

- **The shop's own `dest == origin` tremor guard (`app.go`, spec's own "The five-place grid" below)
  had the identical defect**, restated for `ShopScreenView.SlotMask`: at release,
  `shopGridControlAt(shopView, p)` reads `shopView.SlotMask`, which `ShopSuppressDoll`'s own per-frame
  push (`refreshShopDrag`, `pkg/game/shopview.go`) has already substituted for the suppressed
  composition by the time a release is processed, `app.go`'s own per-frame arm running before the
  release switch on the very same frame. A release back on the origin doll slot answered `ok=false`
  from `shopGridControlAt` (the origin's own mask pixel cleared), so neither `ShopClick` nor
  `ShopDrag` ever crossed the seam. Fixed the same way: `ShopScreenView` gained a second field,
  `OrdinaryDollMask`, which `ShopScreen()` (`pkg/game/shopview.go`) populates from
  `t.shopFigureMasks[i]` unconditionally, NEVER substituted by the suppression arm that still
  substitutes `SlotMask` alone. A new function, `shopReleaseIsOrigin(v ShopScreenView, origin
  ShopControl, p image.Point) bool` (`pkg/ui/shopscreen.go`), answers the origin-identity question
  against `OrdinaryDollMask` for a doll origin (falling back to the ordinary `shopGridControlAt` read
  for the other three surfaces, which carry no suppression); `app.go`'s release handling asks it
  first, ahead of the `dest, ok := shopGridControlAt(...)` read the cross-cell drag case still uses.
  `TestTheApplicationATremorReturningToTheDollIsATapNotADrag` (`pkg/ui/shopdrag_test.go`) is
  mutation-witnessed by reading `v.SlotMask` instead of `v.OrdinaryDollMask` inside
  `shopReleaseIsOrigin`, which reddens it (`clicked = []`, want `[3]`) — but only once the test's own
  fixture, `fakeShopTown` (`pkg/ui/town_test.go`), actually installs the suppressed state:
  `ShopSuppressDoll` now records `suppressSlot`, and `ShopScreen()` builds a suppressed COPY of the
  ordinary mask (every pixel equal to `suppressSlot` zeroed) whenever it is set, in place of
  `ShopSuppressDoll` merely appending the call to a slice the fixture's own `ShopScreen()` never read
  back. Before that fixture fix, the same mutation would not have reddened this test, because the
  fixture's `SlotMask` never moved regardless of what `ShopSuppressDoll` was called with.

- **The unit test above proved the assertion load-bearing, not that the shop's own integration
  witness reached it** (round-2 adversarial review, eleventh pass, 2026-08-17, counterexample C1):
  `scenarios/1005-doll-and-shop.json`'s steps 20-22 (the shop's own three-step press/move/release
  cited above) carried a `move` offset of (3,3) window pixels. `shopDragMoved` (`pkg/ui/app.go`)
  accumulates in FRAME pixels, computed after `a.place.WindowToFrame`, a 2.5x reduction at this
  window size, so a (3,3) window-pixel move produced a Manhattan distance of at most 2 frame
  pixels, never crossing `TapSlop` (4): the drag never armed, `ShopSuppressDoll` never suppressed
  anything, and the release reached `shopReleaseIsOrigin` in a state indistinguishable from the
  original unfixed code. Reverting `OrdinaryDollMask` to `SlotMask` inside `shopReleaseIsOrigin`
  left the scenario passing unchanged. Widening the offset to (10,10) alone does not repair this:
  `HeadlessShopPoint`'s own "doll" surface resolution (`pkg/ui/headlesspointer.go`) picked its pixel
  by scanning `shopGridControlAt`, which reads the possibly-suppressed `SlotMask`, so once a widened
  move crossed `TapSlop` and suppression engaged, the next step's own attempt to name "doll slot 1"
  (the release, offset-free) found no pixel at all (`the open room answers no pixel for doll 1`),
  since the suppressed mask no longer paints any pixel as that slot's own layer. Fixed with two
  changes: `HeadlessShopPoint`'s doll branch now scans `shopDollSlotAt(view.OrdinaryDollMask, p)`
  directly, the same mask `shopReleaseIsOrigin` itself reads, so the scenario's own point resolution
  answers the slot's true location regardless of any drag in progress; and the move step's offset
  widened to (16,16), which reliably crosses `TapSlop` in frame pixels regardless of sub-pixel
  rounding at the press point. With both changes, reverting `OrdinaryDollMask` to `SlotMask` inside
  `shopReleaseIsOrigin` now reddens the SCENARIO itself: `step 23 (assert_shop): the shop doll slot
  1 = 265, want empty`. The scenario's own step 23 assertion was already correct (`worn:
  [{slot:1, empty:true}]`, `carries: [{code:265, count:1}]`, the item comes off); the prose above
  this file carried, in error, "asserting the item is still worn," corrected where the three-step
  sequence is introduced.

## FR-8 — the ground drop, during a mission

A drag released outside every inventory box during a mission — the doll, the worn box and the pack
bar alike, `inventoryCaptures`' own single surface — drops the carried item to the ground in a sack
at the cell under the cursor (`ITEM-DROP-008`; `DIV-088`). This is the one part of the story that
reaches hashed simulation state: two new `sim.Command` kinds and a `pkg/sim` function that applies
them.

- `sim.KindDropCarried` (13) and `sim.KindDropWorn` (14) are `ITEM-CMD-007`'s destination code 3
  (the ground), source code 2 (an actor's container element, `Spell` the element index) and source
  code 1 (an equipment slot, `Spell` 1..12) respectively. `X`, `Y` carry the release cell in world
  units.
- `World.dropToGround` (`pkg/sim/drop.go`) is `ITEM-DROP-008`'s own geometry, shared by both
  sources: the requested cell is honoured only within a Chebyshev window of 2 of the entity's OWN
  CURRENT CELL, read at application time rather than at release time — two independent
  `abs(...) > 2` comparisons, a 5x5 square and not a radius. A destination this world cannot encode
  (out of bounds, `sackFault`), or one outside the window, falls back to the entity's own cell. **The
  fallback cell is itself re-checked against `sackFault` before a sack is planted there**
  (round-2 adversarial review, 2026-08-16, follow-up 1): the entity's own current cell is bounds-safe
  on every write path this build's normal play reaches — `placeAt`, the one gameplay write of
  position, is gated through `terrainOpen`'s own `cellIndex`, and the OTHER write, the per-tick
  movement step, relies on `searchRoute`'s own bounds safety, asserted by that routine's own contract
  rather than independently proven here. An entity CAN exist out of bounds at construction — nothing
  in `NewWorld`/`NewStockedWorld` refuses it, `pkg/sim/world_test.go`'s own `sample()` places one
  there on purpose — so the fallback is guarded rather than trusted: on the (today unreachable through
  normal play) fault branch, the item is discarded rather than planted off the world's own grid, since
  both callers have already removed it from the entity before `dropToGround` runs and there is no cell
  left to put it back on. `TestDropFromAnEntityOutsideBoundsPlantsNoSack` (`pkg/sim/drop_test.go`)
  builds an entity at X=-1 and asserts the sack count stays zero after the drop command applies,
  mutation-witnessed by removing the guard (`Sacks() = 1, want 0`). The drop is never refused for an
  in-bounds entity. Gold is always zero: `ITEM-DROP-008` puts gold on a different opcode, and no
  screen in this build carries a coin the player can pick up.
- `World.dropFromContainer` removes one unit of the container element at `Spell`, on `KindEquip`'s
  own element-not-unit indexing (0138 FR-9); at a count above 1 the element stays at count minus 1,
  at a count of 1 it is removed entirely. `World.dropFromEquipment` clears equipment slot `Spell`
  the same way `unequip` does. Both then call `dropToGround`. Both refuse silently — the entity's
  container or equipment unchanged — on an out-of-range source; an empty equipment slot is refused
  the same way.
- On the `pkg/ui` side, `Viewer.dropCellAt(x, y int) (int32, int32)` turns a release point into a
  world cell through `groundCellAt`, or into `dropOffMapCell` (`1<<20` on both axes, a sentinel far
  outside any map) when the point falls off the drawn tiles — `ITEM-DROP-008`'s own "never refused"
  carried through a release this package cannot place on the ground at all: `pkg/sim`'s Chebyshev
  window falls back to the dropper's own cell for that input exactly as it does for any other
  out-of-window request.
- `command.go`'s two drag-release arms (`dragFromPack`, `dragFromDoll`) gain a second case beside
  the existing equip/unequip one: `!v.groundSurfaceCaptures(in.CursorX, in.CursorY)` raises
  `v.raiseGroundDrop(worn, idx, x, y)` — a fourth one-shot request (`invDropRequest`/`invDropWorn`/
  `invDropX`/`invDropY`, `viewer.go`), `invEquipRequest`'s own index-plus-one shape, widened with a
  `worn` flag because a third field alone cannot say which container the index names. A release
  that lands back on the doll box's own frame or background, or anywhere else
  `groundSurfaceCaptures` still claims, raises neither request — unchanged from FR-5.

  **The gate was `!v.inventoryCaptures(...)` alone through round 2's ninth pass, which measures only
  the doll box, the worn box and the pack bar** (round-2 adversarial review, tenth pass, 2026-08-17,
  unproven-observation item). `command.go`'s own dispatch order checks the HUD toggle strip
  (`hudToggleCaptures`), the spellbook strip (`spellbookCaptures`), the minimap (`minimapCaptures`)
  and the side panel (`panelCaptures`) only for a PRESS, ahead of the block that arms
  `v.invGrab`/a drag candidate; none of the four is asked again at RELEASE. A release during an
  already-armed drag that lands on any of those four surfaces therefore reached the ground-drop
  arm unchallenged, since `inventoryCaptures` alone answers false there. Measured directly (a
  scratch fixture pressing a pack cell, dragging past `TapSlop`, and releasing over the spellbook
  strip's own geometry) before being fixed: `TakeInventoryDrop()` answered `(worn=false idx=1 x=1
  y=25 ok=true)` — a real ground-drop request, over a screen surface that HUD code owns and draws
  on top of the map. The same fixture over the minimap answered `(worn=false idx=1 x=34 y=5
  ok=true)`. Fixed with one combined gate, `Viewer.groundSurfaceCaptures(x, y int) bool`
  (`pkg/ui/inventory.go`): `inventoryCaptures(x, y) || hudToggleCaptures(x, y) ||
  spellbookCaptures(x, y) || minimapCaptures(x, y) || panelCaptures(x, y)`, used at both drag-release
  arms above and by `HeadlessGroundPoint` (`pkg/ui/headlesspointer.go`, the scenario driver's own
  ground-cell probe), which asked the same narrower three-surface question before this pass.
  `TestGroundDropRefusesTheSpellbookStripDuringAnArmedDrag` and
  `TestGroundDropRefusesTheMinimapDuringAnArmedDrag` (`pkg/ui/doll_test.go`) each press a pack cell,
  drag past `TapSlop` onto the surface's own geometry (`v.spellbookGeometry()` /
  `v.minimapGeometry()`), release, and assert `TakeInventoryDrop()` answers `ok=false`;
  mutation-witnessed by reverting the two release-arm gates to `!v.inventoryCaptures(...)`, which
  reddens both (the spellbook case reproduces the measured `ok=true` above exactly). The panel and
  HUD-toggle surfaces are covered by the same combined gate but carry no dedicated test in this
  story: neither is reachable by a drag release in the shipped campaign missions this build's
  scenarios drive, so their coverage rests on `groundSurfaceCaptures` being one function used
  everywhere rather than four independently-asked ones.

  **No scenario point form carried a pixel offset before this pass**, so no scenario JSON step
  could express a tremor gesture — a move that stays within the SAME logical slot or cell
  (`doll_slot`, `pack_cell`, …) while still crossing `TapSlop`'s 4 Manhattan pixels. `HeadlessPoint`
  (`pkg/game/headlesspointer.go`) gained two fields, `OffsetX`/`OffsetY` (JSON `offset_x`/`offset_y`,
  both `omitempty`), added to whatever pixel `resolveBase` names for the point's own form before the
  point is used; a step with `offset_x: 3, offset_y: 3` on the same `doll_slot`/`pack_code` the
  press named produces a move inside the slot rather than a move onto a different one.
  `scenarios/1005-doll-and-shop.json` (version 4, 87 steps) uses this for three tremor sequences.
  The mission doll's "hero" member (slot 1) and the mission doll's "join:51" member (slot 1) each
  gain a `move` step with `offset_x: 3, offset_y: 3` inserted into the file's own PRE-EXISTING
  tap-unequip press/release pair, so the pair becomes press/move/release; its own following assert
  (slot 1 now empty, the item carried) already checks the tap outcome the tremor guard fixes, so no
  separate assert or re-equip cycle is needed. A first attempt inserted a full separate
  press/move/release/assert/re-equip cycle instead; it roughly doubled the elapsed simulation time
  inside the hero's own section and drifted `join:51` (entity 43, deterministic under `pkg/sim`)
  outside the mission's static camera viewport by a later `select_member 'join:51'` step, so the
  in-place insertion is used instead. The shop doll (index 1) gains its own three-step
  press/move(16,16)/release, asserting the item comes off (worn empty, carried), followed by a
  two-step drag from the pack back onto the doll's own drawn area (`doll_box`, below) restoring
  party state for the scenario's later, pre-existing steps. The offset is (16,16) rather than
  (3,3): `shopDragMoved` (`pkg/ui/app.go`) accumulates in FRAME pixels, after `WindowToFrame`'s
  2.5x reduction at this window size, so a (3,3) window-pixel move never crosses `TapSlop` (4) and
  the drag never arms at all — the round-2 adversarial review's eleventh pass found the fixture
  never exercised the shop guard `shopReleaseIsOrigin` reads (below). (16,16) reliably crosses
  `TapSlop` in frame pixels regardless of sub-pixel rounding at the press point, and
  `HeadlessShopPoint`'s own "doll" surface resolution (`pkg/ui/headlesspointer.go`) now reads
  `OrdinaryDollMask` rather than the possibly-suppressed `SlotMask`, so the release step still
  finds the origin's own pixel once the drag has armed and suppressed it.
- `Viewer.TakeInventoryDrop()` drains it, `TakeInventoryDollUnequip`'s own shape.
  `mapWorld.dropFromInventory()` (`pkg/game/world.go`) calls it every `frameAdvance`, alongside
  `equipFromPack`/`unequipFromWorn`/`unequipFromDoll`, and `mapWorld.enqueueDrop` turns the drained
  values into the matching command kind. Neither function resolves anything against the item
  table or asks a wear rule: a drop is never refused on what the item is, only on where it came
  from, and `pkg/sim`'s own refusals above are the only ones that apply.

## FR-9 — the town shop screen

The same four behaviours FR-1 through FR-6 build for the mission's doll, restated for the shop's
own character region (`SHOP-FIGURE-041`, `SHOP-FIGURE-042`): per-pixel hover naming the worn item,
click-to-unequip, drag-to-equip from the shop's shelf or pack, and drag-off-doll. The shop keeps no
`sim.World`; every write below is a direct mutation of the party record, `shopPackItems`' own
precedent restated for equipment, and persists the same way that record already does.

- `shopDollSlotAt(mask *SlotMask, p image.Point) (int, bool)` (`pkg/ui/shopscreen.go`) is
  `dollFigureSlotAt`'s own lookup restated for `shopFigureRect` (480,240)-(640,480): the shop
  figure is drawn at that rect's own size with NO centring, so the offset math the mission's
  centred figure needs has no counterpart here — the mask is read at the point less the rect's own
  origin. The answer is zero-based, `dollFigureSlotAt`'s own indexing. **It refuses the name-plate
  rect and both picker chevron rects before consulting the mask** (round-2 adversarial review,
  2026-08-16, Counterexample 1): `drawShopCharBlock` paints the name plate and the two chevrons
  opaque OVER the composed figure, so a mask cell under any of the three can be occupied while the
  drawn pixel there is plate or chevron, not doll. `TestShopDollSlotAtRefusesThePlateAndTheChevronsEvenWhereTheMaskIsOccupied`
  (`shopscreen_test.go`) marks all three rects occupied in a synthetic mask and asserts each still
  answers `(0, false)`, restoring `DIV-085`'s own property — "a hit test and a drawn pixel can never
  disagree" — for the shop's own two overlays, not only for the plate that round 1 already covered.
  **It also refuses the character block's own one-pixel border** (round-2 adversarial review, second
  pass, 2026-08-16, minor): `drawShopCharBlock` paints `outline(dst, shopCharRegion, invBorder)`
  AFTER the figure, and `shopCharRegion` (480,238)-(640,480) shares its left edge, right edge and
  bottom edge with `shopFigureRect` (480,240)-(640,480) — only the top differs, by the two-row
  furniture gap. The border overwrites the figure's own left column, right column and bottom row;
  `shopDollSlotAt` now refuses a point on any of the three lines before consulting the mask.
  `TestShopDollSlotAtRefusesTheCharacterBlocksOwnBorderEvenWhereTheMaskIsOccupied`
  (`shopscreen_test.go`) marks all three lines occupied in a synthetic mask and asserts each still
  answers `(0, false)`, mutation-witnessed by removing the refusal.
- `shopDollAreaAt(p image.Point) bool` (round-2 adversarial review, tenth pass, 2026-08-17) is
  `shopDollSlotAt`'s own geometric refusal logic — the name-plate rect, both picker chevron rects,
  the character block's own one-pixel border — extracted into its own function, mask-independent:
  it answers whether `p` lands on the doll's own drawn area at all, never which slot. `shopDollSlotAt`
  calls it before consulting the mask, unchanged from the caller's own point of view. It exists
  because `shopGridControlAt`'s doll case can only resolve a release to a slot the mask marks — a
  WORN item's own pixels — and has no fallback for a release meant for the doll while it wears
  nothing there, unlike the mission side's `dollBox()`, a fixed screen box tested by membership
  alone, independent of what is currently drawn inside it. `app.go`'s release handling (inside
  `case armed && moved >= TapSlop:`) adds a fallback branch after the ordinary
  `dest, ok := shopGridControlAt(shopView, p)` read: when `origin.Kind != ShopControlDoll` and
  `shopDollAreaAt(p)` answers true where `shopGridControlAt` answered `!ok`, the release resolves to
  `ShopControl{Kind: ShopControlDoll}` and `dragShop` runs the same as any other doll destination —
  `ShopDrag`'s own doll-destination cases (`pkg/game/shopview.go`) match on `to.Kind ==
  ShopControlDoll` alone and never consult `to.Index`, so naming no slot is sufficient. Before this
  fix, a shop character fully undressed by drag could not be re-dressed by dragging from the pack
  or shelf onto the doll — only by tapping each item in from the pack one at a time — since every
  drag release over the bare doll area found no marked pixel and fell through to the ordinary miss.
  `TestTheApplicationDragsFromPackToAnUnwornDollArea` (`pkg/ui/shopdrag_test.go`) presses a pack
  cell, drags onto an unmarked point inside `shopFigureRect`, releases, and asserts
  `town.dragged == [[{PackCell,1},{Doll,0}]]`; mutation-witnessed by reverting the `app.go` fallback
  condition to `false && ...`, which reddens it (`dragged = []`, want the one pair above).
- `shopGridControlAt(v ShopScreenView, p image.Point) (ShopControl, bool)` widens `ShopControlAt`
  with the doll, tested first: `shopFigureRect` stands entirely inside the merchant's room picture
  and over no grid cell, so the order carries no ambiguity. It answers a new `ShopControlKind`,
  `ShopControlDoll` (`Index` the zero-based slot), for a doll pixel; `ShopControlShelfCell`,
  `ShopControlPackCell` and `ShopControlTableCell` pass through unchanged from `ShopControlAt` — the
  table joined the recognised set at round 2, see "The five-place grid" below; every other kind — a
  button, an arrow, the picker, the merchant, a room rectangle — is still a miss, since none of them
  is a drag surface this story's gestures use. This is the drag machine's own hit test, for both an
  origin and a destination.
- `ShopHoverLines` tests the doll first, through `shopDollSlotAt`, and answers
  `ShopScreenView.SlotInfo[slot]` — a new field, `[12][]string`, one slot's characteristics apiece,
  populated by `ShopScreen()` from `shopEquippedCode`+`itemInfoLines` for whatever the ordinary
  (never the suppressed) doll currently wears.
- `ShopScreenView.SlotMask` is populated by `ShopScreen()` from `townScreen.shopFigureMasks[i]`,
  the mask `composeShopFaces` built beside the shown member's figure.
- The App-level drag machine (`app.go`'s `stepTown`, `inShop` branch) mirrors `command.go`'s: a
  press over the doll, the shelf grid or the pack strip (`shopGridControlAt`) is held as a
  candidate (`a.shopDragOrigin`/`a.shopDragArmed`/`a.shopDragMoved`/`a.shopPressX`/`a.shopPressY`),
  and its high-water distance is latched every frame it stays armed — including the release frame,
  before the release is dispatched, so the live suppression push below sees the same frame a
  release will resolve. On release:
  - crossed `TapSlop`, landing on a DIFFERENT family than it began on: `flow.dragShop(origin,
    dest)` across the seam. A release naming no recognised surface at all — `DIV-088`'s own "no
    ground on a town screen" — sends nothing; the item stays exactly where the drag began.
  - crossed `TapSlop`, landing on the EXACT SAME CELL it began on: `flow.clickShop(dest)`, Shift
    filled in, rather than `dragShop`. `TapSlop` is 4 Manhattan pixels, high-water, latched every
    frame a press stays armed; a grid cell is 80 pixels square and the doll's own marked pixels sit
    far closer together than that, so an ordinary hand's tremor crosses 4 pixels while the gesture
    never leaves the cell it pressed. `ShopDrag`'s switch has no same-family case — dragging a cell
    onto its own family is not a shop action — so before round 2's second-pass review this reached
    `ShopDrag(origin, dest)`, unrecognised, and the click the player meant was silently dropped: on
    the doll specifically, this pre-empted the ORDINARY tap-to-unequip arm below it in the same
    switch. That review's fix used `dest.Kind == origin.Kind` — matching Kind alone — which resolved
    the tremor case but also matched a GENUINE drag between two DIFFERENT cells of the same family
    (shelf cell 0 to shelf cell 5, or any doll slot to another, since every doll pixel is its own
    cell): the round's own THIRD pass (2026-08-16, counterexample 1) found this and narrowed the
    compare to `dest == origin`, the exact cell a hand's own tremor produces. A drag that crosses to
    a different cell of the same family now falls to `dragShop` below, which `ShopDrag`'s own switch
    still answers with no case at all — a no-op, per the bullet below.

    **This same-cell compare was itself provably wrong for the doll once a drag had armed**
    (round-2 adversarial review, tenth pass, 2026-08-17, counterexample 1). It read `dest, ok :=
    shopGridControlAt(v, p)` unconditionally and compared `dest == origin`, and `shopGridControlAt`
    reads `ShopScreenView.SlotMask` for a doll pixel via `shopDollSlotAt`. By release time on an
    armed doll drag, `SlotMask` is the mask `ShopSuppressDoll`'s per-frame push already substituted,
    with the origin slot's own code cleared — the same substitution `refreshShopDrag`
    (`pkg/game/shopview.go`) performs every frame the drag stays armed, running ahead of the
    release switch on the same frame. `shopGridControlAt` therefore answered `ok=false` at the
    origin slot for as long as the drag stayed armed, so `dest == origin` could never hold and the
    tremor guard above never fired for the doll; `ShopDrag(origin, dest)` (`dest.Kind` the zero
    value) reached `ShopDrag`'s switch and was silently dropped, same as before round 2's
    second-pass fix. Fixed by asking the origin-identity question against a mask that is never
    substituted: `ShopScreenView` carries a second field, `OrdinaryDollMask`, populated by
    `ShopScreen()` from `t.shopFigureMasks[i]` before the later suppression substitution overwrites
    `SlotMask`; a new function `shopReleaseIsOrigin(v ShopScreenView, origin ShopControl, p
    image.Point) bool` (`pkg/ui/shopscreen.go`) reads `OrdinaryDollMask` via `shopDollSlotAt` for a
    doll origin, and falls back to the ordinary `shopGridControlAt(v, p)` compare for the shelf,
    pack and table families, which carry no suppression. `app.go`'s release handling calls
    `shopReleaseIsOrigin` first; only when it answers false does it fall through to
    `shopGridControlAt(shopView, p)` for the cross-family `dragShop` case. `dest` — not `origin` —
    is still used for the tremor click, matching what a literal second click at the release point
    would resolve; for the doll this is now `origin` itself, since `shopReleaseIsOrigin`'s own
    match confirms the release landed on the slot the press began on.
  - never crossed `TapSlop`, origin the doll: `flow.clickShop(origin)` — round 1's own
    tap-to-unequip, sharing `ShopClick`'s one mutation door with a genuine click rather than a
    second one for the gesture.
  - never crossed `TapSlop`, origin the shelf or pack: falls through to the ORIGINAL click
    dispatch, unchanged — a plain click on a grid cell behaves exactly as it did before this story.
  - `TestTheApplicationATremorWithinOneShelfCellIsATapNotADrag` and
    `TestTheApplicationATremorReturningToTheDollIsATapNotADrag` (`pkg/ui/shopdrag_test.go`) drive a
    press, an out-of-cell move past `TapSlop`, and a release back on the SAME cell (shelf) or the
    SAME marked pixel (doll) through `a.step` the way ebiten's own input loop does, and assert a
    click reached the seam and no drag did — both mutation-witnessed by reverting to the
    unconditional `dragShop` call. `TestTheApplicationDragsBetweenTwoShelfCellsOfTheSameFamily`
    (same file, round 2's third pass) drives a press on shelf cell 0, a move past `TapSlop` onto
    shelf cell 5, and a release there, and asserts `ShopDrag(shelf 0, shelf 5)` reached the seam and
    no click did — mutation-witnessed by reverting `dest == origin` to `dest.Kind == origin.Kind`,
    which resolves this same case as a click on cell 5.
- **The live suppression** (`flow.suppressShopDoll`/`ShopSuppressDoll`/`townScreen.refreshShopDrag`)
  is pushed every frame a doll-sourced candidate has crossed `TapSlop`, `mapWorld.refreshDollDrag`'s
  own per-frame push restated for a screen with no world underneath it: it recomposes the shown
  member's figure with the dragged slot's code cleared, through the SAME `composeInventorySubject`
  the mission uses, and caches the result — a call naming the same slot already in force costs one
  compare and no recompose. `refreshShopDrag` applies the same `member.Weapon` slot-1 fallback
  `composeShopFaces` applies when the worn-equipment record leaves slot 1 empty (round-2 adversarial
  review, 2026-08-16, follow-up 4): without it, a member whose weapon reaches the doll only through
  `Weapon` — never through `Worn[1]` — would suppress a bare slot 1 while the ordinary figure still
  drew the weapon there, the picture-vs-mask disagreement FR-2 closed for the mission's own doll,
  reopened here on the shop's second figure path.
  `TestRefreshShopDragAppliesTheSameWeaponFallbackAsComposeShopFaces` (`pkg/game/shopdrag_test.go`)
  builds a member whose weapon reaches the doll only through that field and asserts the suppressed
  figure differs from the ordinary one at slot 1. `ShopScreen()` substitutes the suppressed figure
  and mask for the ordinary ones while a suppression is in force **for the shown member specifically**
  — `t.shopSuppressMember` is compared against the loop index, not merely `t.shopSuppressSlot != 0`,
  so switching the picker mid-drag shows every other member's own ordinary figure, never the
  dragging member's suppressed one under a different face. Today's own input wiring never reaches
  this branch through a real gesture — arming a drag and stepping the picker share one physical
  button, so `app.go` can never interleave them — but the guard is reachable at the model's own API,
  one call at a time, and `TestShopScreenNeverSubstitutesAnotherMembersSuppressedFigure`
  (`pkg/game/shopdrag_test.go`) drives it that way and mutation-witnesses the member comparison
  rather than leaving it as untested defence-in-depth (round-2 adversarial review, 2026-08-16,
  follow-up 3, correcting round 1's own "not reachable" comment, which was too broad: reachable at
  the model API, only unreachable through today's input wiring).
- **The two shop compositors applied the slot-1 fallback; every raw-array reader did not**
  (round-2 adversarial review, second pass, 2026-08-16, counterexample 1). `composeShopFaces` and
  `refreshShopDrag` both draw the shown member's weapon at slot 1 when `Worn[0]` is empty and
  `member.Weapon` is set; `shopEquippedCode`, `shopUnequipDoll`, `shopUnequipToTable` and the `old`
  value `shopWearInto` reads before overwriting slot 1 all read `t.shopWornSlots(i)` directly, with
  no fallback — a shop doll drawn with a fallback weapon at slot 1 answered no hover text for it,
  refused to unequip it (both to the pack and to the table), and, worn into by a fresh item, lost it
  outright: `shopWearInto`'s own `old := worn[slot-1]` read 0 for a slot the figure showed occupied,
  so nothing was staged to the pack for the item the drag replaced. `shopSlot1Code(i int, worn
  *[sim.EquipSlots]uint16) (code uint16, viaFallback bool)`
  (`pkg/game/shoproom.go:287`) is the two compositors' own rule, shared: `worn[0]` if occupied,
  otherwise `member.Weapon.Code` if set, otherwise 0, with `viaFallback` telling a writer which
  field a displaced code came from. `shopEquippedCode` calls it for slot 1; `shopWearInto`,
  `shopUnequipDoll` and `shopUnequipToTable` all read slot 1's current code through it and, when
  `viaFallback` is true, write no array cell — the code came from `member.Weapon`, which stays set,
  and the latch below is what records that it has been taken.
  `TestShopEquippedCodeAppliesTheSameWeaponFallbackAsComposeShopFaces`,
  `TestShopUnequipDollTakesOffAFallbackWeaponAndClearsIt`,
  `TestShopUnequipToTableTakesOffAFallbackWeapon` and
  `TestShopWearIntoDisplacesAFallbackWeaponRatherThanLosingIt` (`pkg/game/shopdrag_test.go`) cover
  the four call sites, each mutation-witnessed by reverting to the direct `worn[0]` read.
- **The fallback above answered `member.Weapon`'s code unconditionally, duplicating an already
  unequipped weapon** (round-2 adversarial review, third pass, 2026-08-16, counterexample 2).
  `PartyMember.Weapon` is copied unconditionally through `CarryParty`/`OwnParty`
  (`pkg/mapload/carry.go`, `party.go`) and never cleared by a real, live `sim.unequip`: unequipping
  the starting weapon mid-mission leaves `Carry.Equipped[0] == 0` AND `Carry.Items` already holding
  the weapon's own code, with `member.Weapon` still set to the same weapon it always was. Every one
  of `shopSlot1Code`'s four callers answered the fallback for exactly this state and minted a
  second copy of an item the player already carries: `shopUnequipDoll` appended the code to a pack
  that already held it, `shopUnequipToTable` staged a second, directly sellable table place for it,
  `shopWearInto` displaced it a second time on top of whatever already sat in the pack, and
  `composeShopFaces`/`refreshShopDrag` — the two compositors `shopSlot1Code` was itself restated from
  (counterexample 1, above) — drew the duplicate weapon on the doll a SECOND time on their own
  unconditional `member.Weapon != nil` test, disagreeing with `shopSlot1Code`'s own readers the same
  way the pre-`shopSlot1Code` state disagreed for the never-worn case.
  Fixed with the persisted per-member latch FR-2 describes, rather than a live pack scan at each
  call: `shopWeaponFallbackCode(i int) (code uint16, ok bool)` (`shoproom.go`) answers
  `member.Weapon.Code` only while `member.WeaponMaterialized` is false, and `shopSlot1Code`,
  `composeShopFaces` and `refreshShopDrag` all call it instead of reading `member.Weapon` directly,
  so the two compositors and every raw-array reader agree again. A live pack scan cannot replace the
  latch: `shopWearInto`, called from `shopEquipFromPack`, computes `old` — slot 1's displaced code —
  AFTER the pack has already lost the item being equipped FROM it; if that item happened to be the
  fallback weapon's own code, a scan of the now-shorter pack would find the code gone and answer the
  fallback available again, exactly at the moment `shopWearInto` is about to place a different code
  into the same slot. The latch is raised, through `materializeStartingWeapon`, inside
  `shopWearInto`, `shopUnequipDoll` and `shopUnequipToTable` themselves, before any pack mutation
  runs, so the state a later read sees is never mid-operation.
  **Every slot-1 take-off raises it, whichever field the code came from** (seventh pass, 2026-08-17,
  R2). The two unequip sites raised it on their `viaFallback` arm alone, so a member who took a REAL
  weapon off left the latch down; `shopWeaponFallbackCode` then offered his starting weapon into the
  slot he had just emptied, and the next take-off appended a SECOND unit of that code to the pack,
  minted from nothing. `shopWearInto` has had the unconditional rule since the fifth pass; this is
  the same rule on the way out. Witnessed by
  `TestTakingARealWeaponOffInTheShopRetiresTheFallback` and
  `TestStagingARealWeaponOnTheTableRetiresTheFallback` (`pkg/game/weaponlatch_test.go`), and in the
  running game by `scenarios/1005-doll-and-shop.json`, whose shop leg takes a shipped weapon off a
  roster companion and asserts the doll's own slot 1 is empty afterwards.
  `TestShopWeaponFallbackCodeRefusesWhenThePackAlreadyHoldsTheCode`,
  `TestShopEquippedCodeDoesNotDuplicateAPostMissionUnequippedWeapon`,
  `TestShopFallbackCannotAppendASecondWeaponToThePack`,
  `TestShopFallbackCannotStageASecondSellableWeapon` and
  `TestShopEquippingThePackedStartingWeaponDoesNotDisplaceADuplicate` (`pkg/game/shopdrag_test.go`)
  cover the read side, the two write sides (`shopUnequipDoll`, `shopUnequipToTable`) and the
  pack-then-wear ordering `shopWearInto`'s own comment names, each mutation-witnessed against the
  latch. The fallback still fires for the state it exists to cover: a companion whose starting
  weapon was never folded into the array at all (`rosterTemplate`, `DIV-070`'s own `NPC_Scrakan`),
  since that member's pack never holds the code to begin with, and `Carry != nil` alone is not a
  valid discriminator between the two states — a companion of this shape can also hold a `Carry`
  after fighting a mission.
- **`ShopDrag(from, to ui.ShopControl) ui.TownAction`** (`townscreen.go`, via `shopview.go`) is the
  seam's own dispatch, **twelve** recognised pairs — every ordered cross-family pair the doll, the
  shelf, the pack and the table admit among them: four pairs (pack→doll, doll→pack, shelf→doll,
  doll→shelf) restated from the shop's own pre-drag click handlers; five more pairs added when
  round 2's own first landing wired the table in (doll→table, pack→table, shelf→table,
  table→pack, table→shelf); three more added at round 2's second pass, below (table→doll and the
  direct pack↔shelf pair):
  - pack cell → doll: `shopEquipFromPack` — wears at no charge, `EquipTarget`'s slot and
    `shopUsable`'s class check both asked before anything is taken from the pack.
  - shelf cell → doll: `shopEquipFromShelf` — buys and wears in one gesture (`DIV-087`: no claim
    states whether the original's shelf-to-equip pair charges the purse; this build treats a
    shelf item released on the doll as a purchase followed by a wear attempt). Every refusal —
    no wearable slot, the wrong class, a purse the price outruns — leaves the shelf and the purse
    untouched: all three are checked before `Shop.RemoveFromShelf` is ever called.
  - doll → pack cell: `shopUnequipDoll` — takes the worn item off into the pack.
  - doll → shelf cell **or doll → table cell** (one case arm, both destinations): `shopUnequipToTable`
    — takes the worn item off and stages it on the table for sale (`DIV-089`: no claim states
    whether the original's own inventory carries a comparable gesture at all; this build reuses
    `shopFromPack`'s own merge-or-open rule, so a second unit of the same code at the same price
    joins the table place already there, and a full table (five places) refuses the move, leaving
    the slot worn).
  - pack cell → table cell **or pack cell → shelf cell** (one case arm, both destinations):
    `shopFromPack(t.packBase+from.Index-1, false)` — the same staging a plain click on a pack cell
    already does, dispatched by a drag instead (round-2 adversarial review, 2026-08-16, the
    five-place-grid determination, below). The shelf destination was added at round 2's second pass
    (counterexample 5): `ITEM-CMD-007`'s own opcode space pairs the container code (2) with a shop
    code (4..8) symmetrically, table and shelf alike, and this build has no shelf-only stock
    mutation to reach beyond staging — a shelf's own contents are server-authored inventory, not a
    place the model writes player goods into — so a shelf drop reaches the SAME staging act a table
    drop already does, not a second one invented for the destination alone.
  - shelf cell → table cell **or shelf cell → pack cell** (one case arm, both destinations):
    `shopClickShelfCell(from.Index)` — the same staging a plain click on a shelf cell already does.
    The pack destination was added at the same second pass, the mirror of the bullet above: an
    unpriced direct pack insert would let a shelf item enter the pack with no purse ever consulted,
    so a pack destination reaches the priced staging act instead.
  - table cell → pack cell or table cell → shelf cell (one case arm, both origins): `shopOffTable(from.Index, false)`
    — takes the item back off the table, to the pack (`"back in your pack"`) if it was the party's
    own, to the shelf (`"back on his shelf"`) if it was staged from the shelf; the destination the
    drag named is not consulted, since a table item has exactly one place it can return to and
    `shopOffTable` already knows which.
  - table cell → doll: `shopEquipFromTable` (round-2 adversarial review, second pass, 2026-08-16,
    counterexample 3) — the table's own take-and-wear gesture, `shopEquipFromPack`'s and
    `shopEquipFromShelf`'s pattern restated over `Shop.TakeOffTable` in place of a pack or shelf
    removal: `shopWear` checked before anything leaves the table, the purse checked for a non-`Mine`
    place, and the price charged only on success. This was one of three pairs the round-2
    first-pass switch left unrecognised (with the direct pack↔shelf pair, counterexample 5, above)
    — a plain click on a table cell always meant "take it off," never "wear it," so it had no
    pre-drag click precedent to restate and needed its own function.
  - every SAME-family pair (doll↔doll, shelf↔shelf, pack↔pack, table↔table) is a no-op at this
    seam: dragging a cell onto its own family names no shop action, so `ShopDrag`'s switch carries
    no case for it and never will. **The App-level drag machine no longer sends one here at all**
    (round-2 adversarial review, second pass, 2026-08-16, counterexample 4, below) — a release naming
    the same family as its origin is judged as a click before `ShopDrag` is ever called.
  - a release naming no surface the shop names at all is still a no-op the way it always was: the
    model is never mid-drag, since nothing is taken from either side until the destination is known
    to accept it.
  - `TestShopDragDispatchesTheFourTablePairs` (`pkg/game/shopdrag_test.go`) drives round 2's first
    four new arms against one shop fixture in sequence — shelf-to-table, table-to-pack, pack-to-table,
    table-to-shelf — and asserts the table's own contents and the returning surface's contents after
    each, mutation-witnessed both together and individually.
    `TestShopEquipFromTableWearsAMinePlaceForFree` and `TestShopEquipFromTableBuysAndWearsANonMinePlace`
    (same file) cover the table→doll pair; `TestShopDragPackToShelfStagesOnTheTable` and
    `TestShopDragShelfToPackStagesOnTheTable` (same file) cover the two second-pass shelf/pack
    pairs — all four mutation-witnessed.
- `Shop.RemoveFromShelf(shelf ShopShelf, i int) (ShopItem, bool)` (`pkg/game/shop.go`) takes one
  element from a shelf and shifts the rest down, the shelf's own removal the purchase path needed
  and did not yet have.
- `ShopClick` gains a `ShopControlDoll` case: `shopUnequipDoll(c.Index + 1)`, the same tap-to-unequip
  act the drag machine's own untapped-TapSlop branch calls, through the one mutation door.

**The five-place grid, and the table's own place in it** (round-2 adversarial review, 2026-08-16,
`ITEM-CMD-007`'s "4..8" determination): `SHOP-TRAY-025` reads the dispatcher `R0061`'s own
range check, `4 <= code <= 8`, and its own arithmetic on it — `code-4 == 0` resolves the **shop
tray**, otherwise `code-5` indexes a **shelf**. `SHOP-SCREEN-031` independently draws the three
grids the shop screen tiles: the **table** at five cells in one row (x 32..432, five 80-pixel
columns), the **backpack** at five cells the same way one row lower, and the **shelf** at six cells
in two columns of three (`cols*rows`, `index = row*cols + col`). Code `4` in `ITEM-CMD-007`'s own
vocabulary is this build's **table** — `SHOP-TRAY-025`'s tray and `SHOP-SCREEN-031`'s five-cell
table grid are the same surface under two names, and this build's own `ShopControlTableCell` is that
surface. Codes `5..8` are this build's **shelf** — but the dispatcher's own range check admits only
four shelf indices (`code-5` for `code` in `5..8`, i.e. indices `0..3`) against `SHOP-SCREEN-031`'s
six drawn cells; whether the original pages the shelf through a scroll offset the same way this
build's own `shelfBase` does, so a sixth or seventh shelf item still reaches a `4..8` code through a
different index base, is not settled by either claim and is not needed to answer the question this
investigation was asked. `spec.md`'s round-1 text and `contract.md`'s claims table both wrote "4..8
= a shop shelf" for the whole range, folding the tray in with the shelf; that is imprecise but not
wrong for round 1's own purpose, since round 1 built no table-drag and the distinction was moot.

**Whether table-drag belongs in this story**: settled here as YES, and built (the three new `ShopDrag`
pairs above). Two independent reasons. First, the owner's own directive
(`contract.md:28-34`) reads «в магазине и инвентаре можно взять любой предмет потянув его кликом
drag» — "any item," not "any item on the shelf or the pack" — and `contract.md:10-12`'s own result
clause names "a shop grid" in the singular indefinite, not "the shelf grid" by name, which reads as
covering whichever grid a shipped screen shows, the table included. Second, `SHOP-TRAY-025` treats
the tray (code 4) and the shelf (codes 5-8) symmetrically in the SAME dispatcher, against the SAME
two-sided range check, through the SAME generic move opcode `0x22` that also carries the equip slot
(1) and the pack (2) — the original engine's own command vocabulary draws no line between "a grid you
may drag onto" and "a grid you may not," and round 1's decision to wire three of the five recognised
sources/destinations (doll, shelf, pack) while leaving the table unwired was this build's own line,
not a researched one. `DIV-090` — round 1's "not built" note, reasoned from "nothing in
`contract.md`'s result list depends on it" — is retired spent-and-closed on this determination:
the omission was in scope from the owner's own words, not a disclosed simplification.
`DIV-091` records the table's addition as a divergence row in its own right, since no claim states
that a PLAYER gesture (as opposed to the dispatcher's own opcode) reaches the tray this way; the
divergence is the same shape `DIV-087`/`DIV-089` already record for the shelf's own two gestures.

**A pre-existing system property, not a defect this story introduces**: `data.ArmorFromCode`
refuses a shield-class code by name, and `shopArmourPool` (`pkg/game/shop.go`) interleaves the
shield pool and the armour pool onto the SAME `ShelfArmour` shelf. `EquipTarget` therefore cannot
resolve a shield code to any slot at all — a shield on that shelf answers `shopWear`'s "he cannot
wear that" refusal exactly as a truly unwearable item would. Shields were out of this story's scope
(`contract.md`) before this was found, and remain so; it is recorded here because it shapes which
shelf index this story's own tests must pick.

**The cursor carries a picture during a shop drag too** (round-2 adversarial review, 2026-08-16,
Counterexample 2; `DIV-090` retired spent-and-closed): `contract.md:10-12`'s own result clause names
the shop grid in the same sentence as the cursor ("An item can be picked up from the pack bar **or a
shop grid**... carried on the cursor"), which the round-2 landing's own reasoning ("nothing in
`contract.md`'s result list depends on it") had missed — this was an in-scope gap, not a disclosed
simplification. `ShopScreenView` gained `SlotIcon [12]*image.RGBA` beside `SlotInfo`, the shown
member's own icon per occupied slot (`ShopScreen()`, from `shopIcon(code)`). `App` gained
`shopDragIcon *image.RGBA`, captured once — the same frame `a.shopDragMoved` first reaches
`TapSlop`, `command.go`'s own `dragIcon` capture restated — by `shopDragOriginIcon(v, origin)`
(`pkg/ui/app.go`), which reads the origin's own picture: `SlotIcon` for the doll, `Shelf[i].Icon`,
`Pack[i].Icon` or `Table[i].Icon` for the three grids. `shopDragItemPresent()` answers it only past
`TapSlop`; `ComposeShopScreen` (`pkg/ui/shopscreen.go`) takes two new parameters, `dragIcon
*image.RGBA, hasDrag bool`, and blits the icon centred on the hover point, drawn LAST — over the
hover box, `Viewer.Draw`'s own "carried item last of all" restated for the shop's one-shot software
compositor. `a.shopDragIcon` is reset on press, on release and by `clearShopDrag` (below); the reset
on press and the reset on release are redundant with each other given this build's own input
sequencing (a press always follows a release, never another press) — both are kept as defence in
depth, on `dollSuppressOwner`'s own precedent, and neither is independently mutation-witnessed
against the other's presence.

**Escape drops an armed shop drag before the room unwinds** (round-2 adversarial review, 2026-08-16,
follow-up 2): `App.step`'s Escape branch calls the new `clearShopDrag()` — zeroing
`shopDragArmed`/`shopDragOrigin`/`shopDragMoved`/`shopDragIcon` — before `flow.escape()` unwinds the
room. Without it, leaving the shop room by Escape with the button still physically held left
`a.shopDragArmed` true; the eventual release lands outside `stepTown`'s own `inShop` gate once the
screen has changed, so nothing would ever have cleared it, and the next shop visit would arm a stale
origin the instant the button first crossed `TapSlop` for any reason.

## FR-10 — the pointer step, and the scenario that drives this story in a shipped mission

**A story about pressing, moving and releasing over a picture had no way to be driven**, so its whole
evidence was unit tests plus a human's own hands. Scenario vocabulary before this story reached a
control BY NAME — a row, a button, a member id — and an inventory has none: its surfaces are pixels
of a composed figure. `pkg/game/headless.go`'s version rises to **4** with three commands and one
top-level field, all on the front-end stage:

- `pointer` — `action` (`press`, `move`, `release`) and `at`, one named surface: `doll_slot` 1..12,
  `doll_box`, `pack_cell` (a container element index), `pack_code`, `ground`, `shop`
  (`doll`/`doll_box`/`shelf`/`pack`/`table`/`picker_prev`/`picker_next`, by index or by `code`),
  `shop_idle`. The shop's own `doll_box` surface (round-2 adversarial review, tenth pass,
  2026-08-17) resolves `shopFigureRect`'s own center directly, `HeadlessDollBoxPoint`'s pattern
  restated for the shop: the shop's per-pixel `doll` surface can only name a WORN slot's own mask
  pixel, and has no answer for a release meant to land on an unworn doll — the gap
  `shopDollAreaAt` (below) fixes in production, and `doll_box` is the scenario's own way to drive
  it, `doll_slot`/`doll_box`'s own mission-side split restated for the shop.
  Every point is resolved by asking the PRODUCTION hit test for a pixel this frame
  (`pkg/ui/headlesspointer.go`: `HeadlessDollSlotPoint`, `HeadlessPackCellPoint`,
  `HeadlessDollBoxPoint`, `HeadlessGroundPoint`, `HeadlessShopPoint`, `HeadlessShopIdlePoint`), and
  the edge is dispatched through `ui.App`'s own input path, so a scenario presses what a player
  presses and no geometry is restated in the runner.
- `assert_inventory` — `subject`, `weapon_fallback`, `figure`, `equipment`, `carries`, `sacks`.
  `figure` is what the doll DRAWS and `equipment` is what the array HOLDS; the two differ exactly
  where a starting weapon is drawn without being equipped, which is what `weapon_fallback` names.
- `assert_shop` — `member`, `gold`, `doll`, `worn`, `carries`, `table`.
- `window` — the window size in pixels, reaching `ui.App.Layout`, the same door a real window
  creation or resize uses. A screen with no room at that size refuses to open, so an inventory
  scenario states a size large enough for one.

A version-3 file naming any of them is refused, exactly as a version-2 file naming a version-3
command already was (`0155` FR-7's own rule).

`HeadlessSelectEntity` retries its pixel search up to eight times. A press and a release are two
frames and every frame advances the world, so a walking unit leaves the pixel the search chose and a
single attempt selected whatever had walked into it.

`scenarios/1005-doll-and-shop.json` is this story's own integration witness and is described in
`closure.md`. It runs from a lawful install against an original save, so it is not a `go test` case
(golden rule 2); `pipeline/check-scenarios.sh` is what executes it.
