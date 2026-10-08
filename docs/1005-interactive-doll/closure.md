# 1005 — the interactive doll — closure

## Twelve-aspect matrix

Both rounds: round 1 (the mission map screen — hover, tap-to-unequip, drag between the doll and the
pack) and round 2 (the ground drop, the town shop screen). Each row's note names which round
supplies its evidence where the two differ.

| Aspect | Status | Note |
|---|---|---|
| Data | N/A | No table is added or read differently, either round. `FigureDrawOrder`, `ItemFigureLayerPath` and the rest of `pkg/data`'s figure vocabulary are read exactly as before. |
| Runtime state | PASS | Round 1: the carried item and its origin (`dragCandKind`, `dragCandIdx`, `dragActive`, `dragIcon`) live on `Viewer`. Round 2: the ground drop's own one-shot request (`invDropRequest`/`invDropWorn`/`invDropX`/`invDropY`, `pkg/ui/viewer.go`) follows the same shape; the shop's own drag-candidate state (`shopDragArmed`/`shopDragOrigin`/`shopDragMoved`/`shopPressX`/`shopPressY`) lives on `App`, since the shop screen has no `Viewer`-owned selection of its own to hang it from. Neither round issues a command, nor mutates the party record, until a release resolves. The round's own fifth pass (below, counterexamples A and B) found the starting-weapon fallback's own materialization had been tracked as a value RE-DERIVED from the party's present pack/array contents at every read, rather than as the history it actually is; `PartyMember.WeaponMaterialized` (persisted, see Persistence/save-load below) carries it. **That fix was incomplete and the row said otherwise until the seventh pass** (2026-08-17): the latch's own SEED still scanned the container for the starting weapon's code, so a second unit of that code — a shipped duplicate, loot, or a purchase — read as "already materialized" and retired a fallback that had never fired. The scan is removed; `weaponFallbackSpent` now has one term, whether slot 1 is occupied. The latch has exactly one writer, `materializeStartingWeapon` (`pkg/game/weaponlatch.go:47`), every write and read site is enumerated in the seventh-pass section below with its `file:line`, and `TestWeaponMaterializedHasOneWriter` fails when a site appears that the enumeration does not name. **The eighth pass (2026-08-17, C3) adds the enumeration's first reader outside `pkg/game`** — `mapload.PartyLoadout` (`pkg/mapload/loadout.go:97`) — and widens the scan test to parse `pkg/mapload` as well, so the same mechanism now covers both packages the latch reaches; see the eighth-pass section below. |
| Simulation | PASS (round 2) | Two new `sim.Command` kinds, `KindDropCarried` (13) and `KindDropWorn` (14), and one new `pkg/sim` file, `drop.go`: `dropToGround` (the Chebyshev-window geometry, `ITEM-DROP-008`), `dropFromContainer`, `dropFromEquipment`. This is the one part of the whole story that reaches hashed simulation state (`contract.md`'s AMBER threshold High). Round 1 added no command kind: its equip and unequip both route through the pre-existing `sim.KindEquip`/`sim.KindUnequip`. The shop's own equip/unequip/drag moves reach NO `sim.World` at all — the shop keeps none — and mutate the party record directly, `shopPackItems`' own precedent. The seventh pass adds no command kind and no `pkg/sim` change: `materializeFallbackWeapon` writes the container through the pre-existing exported out-of-`Step` mutator `ReplaceStock`, the same door `MoveCarried` and `TakeSack` already use, and a ground drop still goes through `KindDropCarried`. **The eighth pass (2026-08-17, C3) fixes a defect in this same hashed reach, at the pre-existing mission-open mint rather than at a new command kind**: `mapload.PartyLoadout` folds directly into the entity's `CombatBlock` at every mission's own party mint (`start.go`, the loop building `sim.Entity`'s `DamageBase`/`DamageSpread`/`ToHit`/`Defence`/`Absorption`), and it passed `everEquipped` as an unconditional `false` — a member whose `WeaponMaterialized` latch was already raised (his starting weapon taken off and sold in an earlier mission) was minted with the starting weapon's damage folded into his hashed `CombatBlock` at the next mission's own mint. `PartyLoadout` now passes `p.WeaponMaterialized`, matching `Rearm`'s own in-mission `everEquipped` (`pkg/game/rearm.go:266`), which already read it correctly. No command kind and no `pkg/sim` file changes; the fix is entirely inside `pkg/mapload`'s own loader logic, one tier above `pkg/sim` in the DAG, and touches no `sim.World` method. **The ninth pass (2026-08-17) fixes a third defect in the same hashed reach, at a third call site of `mapload.ResolveEquipmentLoadout`**: `recomputeRaisedSkills` (`pkg/game/rearm.go`), the live per-tick pass that folds a skill raise into the entity's own hashed `CombatBlock` through `SetDerived`, read `everEquipped` as SUBJECT-SCOPED — true only for the current inventory subject — rather than from `PartyMember.WeaponMaterialized`. A non-subject member whose latch had been raised, by either `mw.rearm`'s own tap-off or the shop's own picker on a member other than the one shown, still read `everEquipped` false and was re-armed with the sold starting weapon's damage into `CombatBlock` the next time any of his skill levels rose. Fixed by reading `mw.resolveWeaponMaterialized(mw.missionPartyMember(c.id), eq)` — the LIVE element inside `mw.mission.party`, not `characterDerive.member`, which is a value copy frozen at mission open — matching `Rearm` and `PartyLoadout`'s own rule. See the ninth-pass section below. |
| Player input | PASS | Round 1: FR-2 through FR-5, the hit test, the tap-to-unequip gesture and the drag machine. Round 2: FR-8 (the ground-drop release arm, `command.go`) and FR-9 (the shop's own App-level drag machine, `app.go`'s `stepTown`). Round 2's second-pass review (counterexample 4, below) fixed a tremor past `TapSlop` that never leaves its own origin surface swallowing a click instead of resolving one. The round's own third-pass review narrowed that fix further (counterexample 1, below): `dest.Kind == origin.Kind` also caught a genuine cross-cell drag within one family, which now falls through to `dragShop`'s own no-op instead; the same pass also refused to arm a doll drag on a fallback-only slot 1 (counterexample 4) — **reversed at the seventh pass on the owner's directive**, so that slot is now armed like any other and the mission side materializes a real item and cancels an armed drag released over the HUD toggle bar instead of leaving it live for the next unrelated release (counterexample 5). The round's own fifth pass (below) found the mission map's own gesture surface had four further gaps, all now fixed: a plain move order under a subject-led multi-selection was swallowed with no drag in flight (counterexample C), a doll-origin drag's release checked the wrong, narrow rectangle and resolved to nothing (counterexample D), and the SAME tremor-past-`TapSlop`-within-one-cell hole the second pass fixed for the shop was open on the mission map for both a double-click's second press (counterexample E) and a doll tap-to-unequip (counterexample F). **The tenth pass (2026-08-17) found the second/third/fifth-pass tremor guards above were themselves inert in production, on both the mission doll and the shop doll**: `originSame`/`dest == origin` read the SUPPRESSED mask at release time, which a live armed drag has already cleared at the origin slot, so a release back on the origin could never again read as the origin and the guard never fired outside a test fixture that never pushed the same suppression. Fixed by asking the origin-identity question against an unsuppressed mask (`dollFigureSlotAtUnsuppressed`, `pkg/ui/inventory.go`; `ShopScreenView.OrdinaryDollMask` and `shopReleaseIsOrigin`, `pkg/ui/shopscreen.go`), and by fixing both fixtures (`TestDollTapStillUnequipsAfterATremorWithinTheSlot`, `TestTheApplicationATremorReturningToTheDollIsATapNotADrag`) to push the same suppressed state production installs; see the tenth-pass section below. The same pass found and fixed a release over the HUD toggle strip, the spellbook strip, the minimap or the side panel during an armed drag reaching the ground-drop arm unchallenged (`Viewer.groundSurfaceCaptures`, `pkg/ui/inventory.go`), and removed one dead duplicate case in `command.go`'s release switch. Building the tenth pass's own required tremor scenario surfaced one further gap: the shop's cross-family drag destination had no answer for a release on an unworn doll area (the mask names only a worn slot's own pixels, and the mission side's occupancy-independent `dollBox()` has no shop counterpart), so a bare shop doll could not be re-dressed by drag. Fixed with `shopDollAreaAt` (`pkg/ui/shopscreen.go`) and an `app.go` release fallback; see the tenth-pass section below. |
| AI | N/A | No unit decision changes, either round. |
| UI/HUD | PASS | Round 1: the mask (FR-1), the hover popup (FR-3), the cursor-carried icon (FR-5), the suppressed doll picture (FR-6), and the round-1 review fix below. Round 2: `shopDollSlotAt`/`shopGridControlAt`/`ShopHoverLines`' new doll arm (the shop's own hit test and hover), the shop's own suppressed-figure substitution (`ShopSuppressDoll`/`refreshShopDrag`, `ShopScreen()`'s substitution block) — `dollSubject`'s own pattern restated for a screen with no `Viewer`-owned doll — and, at the round's own adversarial review, the cursor-carried icon for a shop drag (`ShopScreenView.SlotIcon`, `shopDragOriginIcon`, `App.shopDragIcon`, `ComposeShopScreen`'s two new parameters), the doll's name-plate and chevron refusal in `shopDollSlotAt`, and the mission doll's own hit-test gate corrected to match what it draws under a multi-unit selection (`dollFigureSlotAt`, see the round-2 review section below). No disclosed gap remains: `DIV-090` (the shop cursor icon) is retired spent-and-closed. The review's own second pass (2026-08-16) added `shopDollSlotAt`'s refusal of the character block's own one-pixel border (minor, below). The review's own third pass (2026-08-16) closed a gap between what the doll draws under a multi-unit selection and what the reserved worn-box and pack-bar ground still captures for input once `inventoryEligible` stops drawing them (`wornBoxArea`/`packBarArea`, counterexample 6, below). The round's own fifth pass (below) found the mission map's own doll sprite disagreed with the inventory doll over a fallback-only weapon (`refreshAppearance` composing from the raw, un-widened equipment while the inventory doll already read the widened source, counterexample I) and fixed it. |
| Triggers/scripts | N/A | No script opcode touched, either round; the milestone census is unchanged (below). |
| Inventory/equipment | PASS | Round 1: the subject's `SlotMask`, `unequipFromDoll`, the drag-to-equip and drag-to-unequip paths, routed through the existing `mapWorld.enqueueEquip`/`enqueueUnequip`. Round 2: `shopWearInto`/`shopWear`/`shopEquipFromPack`/`shopEquipFromShelf`/`shopUnequipDoll`/`shopUnequipToTable` (`pkg/game/shoproom.go`) are the same rule (`EquipTarget`'s slot, the wear/class check, the swap that returns a displaced item rather than discarding it) restated over the party record directly. The review's own second pass (2026-08-16) closed two disagreements between a compositor's own slot-1 fallback and a reader consulting the raw array beside it — the shop's own readers (`shopSlot1Code`, counterexample 1) and the mission doll's own trackers (`currentFigureEquipment`, counterexample 2) — and wired `shopEquipFromTable` (counterexample 3) and the direct shelf↔pack pair (counterexample 5), all below. The review's own third pass (2026-08-16) closed a duplication the second pass's own fallback rule left open — `shopWeaponFallbackCode` now refuses the fallback once the code is already sitting in the member's own pack (counterexample 2, below) — and widened the mission doll's own drag-source read to the same fallback-aware equipment `SlotInfo` already used (`currentFigureEquipment`, counterexample 3), gating a fallback-only slot 1 out of the drag machine on the consumption side (`WeaponFallback`, counterexample 4). The round's own fifth pass (below) found the fallback's own "already spent" tracking, both in the shop and in `buildInventorySubject`, was itself re-derived from present pack contents rather than carried as history, reopening the same duplication the third pass had closed for one call path only (counterexamples A and B); fixed with the persisted `PartyMember.WeaponMaterialized` latch, read directly by every consumption site rather than re-derived by any of them. The round's own seventh pass (below) found that fix incomplete in three ways and closed all three: the latch's own SEED was still a container scan (R3), the shop raised it on one arm of a take-off only (R2), and a mid-mission save wrote a clone of the party that never carried it (R1). It also brought the mission side to the shop's rule for a fallback-only slot 1, on the owner's directive: pressing it takes the item off, and `materializeFallbackWeapon` puts a real one in the container. **This row was PASS while the mission doll's general equipment slots (every slot but the slot-1 fallback, already covered above) read a stale array** (round-2 adversarial review, twelfth pass, 2026-08-17, C1): `buildInventorySubject` composed the figure and `SlotMask` from `member.Worn` directly — the array as read at the party's last assembly, never rewritten afterward — instead of preferring `member.Carry.Equipped`, which every other equipment reader (the shop, the town screen, `mapload.EquipmentFromParty`'s own callers) already did. A member who finished a prior mission wearing something different from what he was last assembled with opened the next mission's doll showing the STALE loadout. A first correction extracted `missionDollEquipment` calling `mapload.EquipmentFromParty(member)`; a scenario-level mutation-kill of that first correction (below) found it left a narrower, same-mission case open — `member.Carry` is nil for the whole of the mission that assembled a member, so a mid-mission equipment change made through the doll and then saved and reloaded still read the stale `member.Worn`, because the resumed world's own live state was never consulted. Corrected a second time, within the same pass: `missionDollEquipment(w *sim.World, id sim.EntityID, member mapload.PartyMember)` (`pkg/game/inventory.go:193`) now reads `w.Equipped(id)` first, falling back to `mapload.EquipmentFromParty(member)` only when the world holds no entity for `id` yet, and both `buildInventorySubject` and `world.go`'s own `mw.invComposedEquipment` seed call it with the mission's world, the subject's entity id and the party record. See the twelfth-pass section below for both corrections, their scenario-level mutation proofs, and the reverse-direction (`CarryRoster` mid-mission joiner) finding. |
| Persistence/save-load | PASS | Round 1: `mapWorld` gained `figureMasks`/`invDollSuppressSlot`, both ruled `derivable` (`docs/0143-save-and-load/spec.md` FR-5, `TestEveryMapWorldFieldIsRuled`). Round 2: a player-placed sack round-trips through the WORLD's PRE-EXISTING sack encoding — no new persisted field. `TestAWorldHoldingAPlayerPlacedSackRoundTripsByteIdentically` (`pkg/sim/drop_test.go`) drops one item, marshals, unmarshals, re-marshals, and checks the two byte forms are identical and the two hashes agree. `pkg/sim/binary.go`'s `formatVersion` (53) is untouched by this story — confirmed by `git diff master -- pkg/sim/binary.go` showing no change — because `KindDropCarried`/`KindDropWorn` are transient command kinds, not persisted state, and the sack they produce is encoded exactly as a script-placed or a death-dropped one already is. The shop's own new `townScreen` fields (`shopFigureMasks`, `shopSuppressSlot`, `shopSuppressFigure`, `shopSuppressMask`) are presentation state on a struct no save format carries at all — the shop is re-entered fresh from the party record on every visit — so `TestEveryMapWorldFieldIsRuled`'s own ruling does not apply to them and none was needed. The round's own fifth pass adds `PartyMember.WeaponMaterialized bool` (`pkg/mapload/start.go`) — a plain value field on the party record itself, not on `mapWorld`, so `TestEveryMapWorldFieldIsRuled`'s own field-by-field ruling does not enumerate it; it round-trips through `Snapshot.Party`'s existing gob encoding with no format change, `clonePartyMember`'s existing plain value copy, and `OwnParty`'s existing per-member copy loop with no added statement (`TestOwnPartyCarriesWeaponMaterializedAsAPlainValue`, `pkg/mapload/party_test.go`). `docs/0143-save-and-load/spec.md`'s FR-5 table is unchanged by this addition: FR-5 rules `mapWorld`'s own fields, and this field lives on `mapload.PartyMember`, already inside the carried party the save format persists in full. **This row was PASS while a mid-mission save dropped the field, and that is the seventh pass's R1** (2026-08-17): `FrontEnd.liveDriver` (`pkg/game/resume.go:291`) stored a CLONE of the party beside the live mission, so `Snapshot` wrote the clone and every mid-mission write to a `*PartyMember` was absent from a save taken during the mission — a mission-boundary save carried the latch, a mid-mission save did not. `liveDriver` now aliases the caller's slice and names it in place through `mapload.NameParty`. Witnessed twice: `TestAMidMissionLatchReachesTheSave` (`pkg/game/weaponlatch_test.go`), and step 87 of `scenarios/1005-doll-and-shop.json` (step 77 before the eleventh pass's own in-place insertions), which saves and reloads inside a shipped mission and fails verbatim when the clone is restored (below). **Adding `WeaponMaterialized` also changed the gob type descriptor `Snapshot.Party` carries, which broke a byte-exact fixture unrelated to the field's own persistence** (eleventh pass, master-merge task 0, 2026-08-17): `TestReleasedEnvelopeOneSimulationFormFiftyThreeFixture`'s committed `releasedSaveFixtureBase64` was encoded before the field existed, so the merge's own encoder produced different bytes for the same `Snapshot` value and the test failed. The fixture was regenerated from the current encoder; `pkg/sim`'s own hash, form, tick and purse assertions in the same test were unchanged, confirming an envelope-level byte change with no `pkg/sim` binary-form version bump. That regeneration is correct for what the test claims today — this build reproduces its own current encoding byte for byte — but it narrowed what the fixture had witnessed before: whether a save an EARLIER build actually wrote still decodes under this one. `preWeaponMaterializedSaveFixtureBase64` (twelfth pass, 2026-08-17, C3, `DIV-095`) restores `master`'s own pre-regeneration bytes as a second, separately named constant and checks it by DECODE alone (`TestPreWeaponMaterializedSaveFixtureStillDecodes`, `pkg/game/save_test.go`), which is the fixture that now stands for the older, backward-compatibility claim; see `DIV-095`. |
| Campaign/session | PASS (round 2) | The shop's own purse (`Town.gold`) and shelf (`Shop.shelves`) are read and written by `shopEquipFromShelf`/`Shop.RemoveFromShelf`, `DIV-087`'s own purchase-then-wear rule. `shopUnequipToTable` stages a doll-origin item on the table for later sale, `DIV-089`. Both persist the same way `shopFromPack`'s pre-existing table and shelf moves already do — no new persistence mechanism. **This row was PASS while the mission-boundary consequence of `WeaponMaterialized` was untested** (2026-08-17, C3): `mapload.CarryParty` carries the latch across a mission boundary correctly (Persistence/save-load row, above), but `PartyLoadout` — read at the NEXT mission's own construction — ignored it, so a member who sold his starting weapon in one mission arrived at the next mission re-armed with it. The campaign consequence was a stat that should not have survived a session boundary surviving every one of them. Fixed in `pkg/mapload/loadout.go:97`; witnessed by `TestPartyLoadoutRespectsTheMaterializedLatch` (`pkg/mapload/loadout_test.go`), which derives two party members through `PartySpawnWithTable` — one never latched, one latched with an empty slot 1 — and asserts they mint different `Combat.DamageBase`/`DamageSpread` pairs. **This row was PASS while the mission doll itself crossed a session boundary showing the wrong figure** (round-2 adversarial review, twelfth pass, 2026-08-17, C1): the equipment a member carried out of one mission (`Carry.Equipped`, set by `CarryParty`/`CarryRoster` at that mission's finish) is exactly the campaign-session fact the NEXT mission's doll has to open showing, and `buildInventorySubject`'s stale-array bug (Inventory/equipment row, above) meant it did not. `scenarios/1005-doll-carry-over-worn.json` witnesses the session-boundary case directly: it unequips armour in mission 20, confirms the pack over the mission-end notices and town screen, then re-enters a different mission (30) and asserts the doll opens with the same item still off the figure and still in the pack — the campaign's own carry-over, not a same-mission re-render. |
| Shipped content | PASS | Neither the mask (round 1) nor the drop/shop code (round 2) reads a shipped item's own fields to decide whether it applies: `dropToGround`/`dropFromContainer`/`dropFromEquipment` take a bare `uint16` code and never resolve it against `mapload.Table` at all (`ITEM-CMD-007`'s own vocabulary is code-agnostic), and `shopWear`/`EquipTarget` are the SAME table lookup every pre-existing equip path already uses. No enumeration over shipped item rows is owed: the mechanism does not discriminate by item identity, so there is no per-row case a sweep could miss (contrast `1004`'s carrier-vs-spell miss, `implementation/SDD/WORKFLOW.md`'s own cited case, where the mechanism DID discriminate by the field a sweep did not vary). The seventh pass adds a shipped-content witness in place of an argument: `scenarios/1005-doll-and-shop.json` equips, unequips, drops, sells and stages REAL shipped item codes (265, 33076) belonging to a real save's own party, through production loading. |
| Interactions with existing mechanics | PASS | Round 1: FR-7, worn-box double-click, pack-cell double-click, pack-bar scroll, box-select-not-hijacked. Round 2: `shopWearableShelfIndex` (test helper) documents a PRE-EXISTING system property this story's own tests had to work around rather than one it introduced — `data.ArmorFromCode` refuses a shield-class code by name, and `shopArmourPool` interleaves the shield pool and the armour pool onto one shelf, so `EquipTarget` cannot resolve a shield to any slot at all. A shield dragged onto the doll answers `shopWear`'s ordinary "he cannot wear that" refusal, the ONE path every unwearable item already takes; nothing in this story's own code treats a shield specially. Not a new divergence: shields were out of `contract.md`'s scope before this was found and remain so. **The eighth pass (2026-08-17, C3) closes a disagreement between this story's own `WeaponMaterialized` latch and the pre-existing mission-open mint (`mapload.PartySpawnWithTable`/`PartyLoadout`, `pkg/mapload/start.go`), a mechanism this story did not introduce and had not touched before this pass**: the mint always resolved a member's opening `Loadout` with `everEquipped` hardcoded `false`, on the ground (`PartyLoadout`'s own prior doc comment) that mission construction is always this member's FIRST resolution — true before this story, false once the latch can now ride in from an earlier mission. The interaction was untested because the mint predates the latch and the latch predates a test that drove them together. Fixed by reading `p.WeaponMaterialized` at the one call site; `TestWeaponMaterializedHasOneWriter` (`pkg/game/weaponlatch_scan_test.go`) now parses `pkg/mapload` as well as `pkg/game`, so a future site that touches the latch without updating this table fails the same way a `pkg/game` site already did. **The ninth pass (2026-08-17) closes the same interaction at a third call site, `recomputeRaisedSkills`, and against a mechanism THIS ROUND introduced**: the shop's own picker (`shopUnequipDoll`, `shopUnequipToTable`, `shopWearInto`) already raised `PartyMember.WeaponMaterialized` for whichever member is shown, not only the mission's own subject, and a live skill raise reaching a non-subject member never read that write. Fixed at `pkg/game/rearm.go`; see the ninth-pass section below. `TestWeaponMaterializedHasOneWriter` is itself widened at this pass: it previously matched only `*ast.SelectorExpr`, so a composite-literal key (`mapload.PartyMember{WeaponMaterialized: true}`) reached and set the field invisibly, and it scanned `pkg/game` and `pkg/mapload` alone, so a reader in any of the eleven `cmd/*` directories that import `pkg/mapload` was also invisible. Both gaps are closed; see the ninth-pass section below. **The twelfth pass (2026-08-17, C1) closes a disagreement between this story's own mission-doll compositor and `pkg/mapload`'s pre-existing Carry-over-Worn rule, a mechanism this story did not introduce**: `mapload.EquipmentFromParty` (`pkg/mapload/loadout.go`), the reader every party-equipment consumer outside this story's own doll already called, prefers `member.Carry.Equipped` over `member.Worn` whenever a Carry exists; `buildInventorySubject` read `member.Worn` directly instead, disagreeing with the shop and the town screen for any member who had already crossed one mission boundary. A first correction routed the doll's own composition through that reader; a scenario-level mutation-kill of that first correction found it left a same-mission resume case open, because `member.Carry` stays nil for the whole of the mission that is still assembling it, so `EquipmentFromParty` itself falls back to the stale `member.Worn` for any equipment change made and then saved mid-mission. Corrected a second time: `missionDollEquipment(w *sim.World, id sim.EntityID, member mapload.PartyMember)` (`pkg/game/inventory.go:193`) now prefers a live read off the world, `w.Equipped(id)`, ahead of `EquipmentFromParty`, agreeing with `mapload/start.go`'s own mission-open mint (which resolves the identical Carry-then-Worn preference into the world's OWN entity at fresh open) while additionally staying correct across a resume, where `resumeWorld` (`pkg/game/resume.go`) has already replaced the world with the save's own live snapshot before `openMission` runs. `CarryRoster`'s own mid-mission-joiner branch (a unit added to the party by a script, folded in from `w.BoundarySurvivors`) exercises the identical divergence in the opposite direction — the joiner's `Worn` comes from its roster template while its `Carry.Equipped` is read from the live world at the same finish call — and is closed by the same live-world preference, now the PRIMARY mechanism rather than `EquipmentFromParty`'s fallback; see the twelfth-pass section below for both corrections, their scenario-level mutation proofs, and the reachability finding. |

A known in-scope GAP would fail this landing; there is none above.

## Round-1 adversarial review: the suppressed mask fix

The adversarial review found one counterexample against the UI/HUD row above. `refreshDollDrag`
(`pkg/game/world.go`) composed the suppressed figure through `composeInventorySubject` and pushed
only `composed.Figure` through `SetDollSuppressedFigure`, discarding `composed.SlotMask`.
`dollFigureSlotAt` (`pkg/ui/inventory.go`) always read `v.invSubject.SlotMask` — the ordinary,
unsuppressed mask — with no reference to the suppression state `dollSubject` already checks three
fields for. `hoveredItemInfoAt` (`pkg/ui/itempopup.go`) calls `dollFigureSlotAt` with no
drag-awareness of its own.

The effect: while a drag holds one of the doll's own slots, FR-6 correctly draws the doll box with
that slot's layer cleared, but a cursor over the pixels that layer used to own still named the
lifted slot, and the hover popup still showed that slot's item — a hit test disagreeing with the
picture actually on screen, contrary to `DIV-085`'s own stated property ("a hit test and a drawn
pixel can never disagree").

**Fix**: `Viewer.SetDollSuppressedFigure` gained a fourth parameter, `mask *SlotMask`, and a new
field `dollSuppressMask` carries it. `refreshDollDrag` now passes `composed.SlotMask` — the mask
`composeInventorySubject` built beside the same picture — through the same call as
`composed.Figure`. `dollFigureSlotAt` makes the SAME three-field suppression check `dollSubject`
already makes (`dollSuppressOwner == v.invSubject.ID && dollSuppressSlot != 0 && dollSuppressPic !=
nil`) and reads `v.dollSuppressMask` in that branch instead of `v.invSubject.SlotMask`. No pixel
draws differently; only which mask a hit test consults changed. `spec.md` FR-2 and FR-6 are updated
to describe this.

No new `DIV` id: this is a defect against `DIV-085`'s own stated property, not a new divergence.

## Integration witness

**`scenarios/1005-doll-and-shop.json` is this story's integration witness** (seventh pass,
2026-08-17). It runs the production build with no window, from the `en` lawful install, against the
owner's own original save `666 - mission 20 [original]`, and drives the feature through the same
`ui.App` input path a mouse drives. 87 steps: the town's shop room, then a walk into mission 30 and
the map screen's own doll. What it demonstrates, in order:

1. **The composed `InventorySubject.SlotMask` answers the correct slot at a pixel of a known layer.**
   Every `doll_slot` point is resolved by asking `HeadlessDollSlotPoint` for a pixel the running
   figure's own mask assigns to that slot; a press there raises that slot's own request.
2. **A press-move-release through `Viewer.command` removes a real shipped item from the real
   equipment array into the real container.** Steps 54-56 tap doll slot 1 (press, move, release) and
   step 58 asserts equipment slot 1 empty with code 33076 in the container.
3. **A pack→doll drag equips a real shipped item through `enqueueEquip`, wear rule included.** Steps
   59-61 drag `pack_code 33076` onto the doll box and step 63 asserts slot 1 holds it and the pack
   does not.
4. **A release outside every inventory box plants a sack.** Steps 64-66 drag doll slot 1 to a map
   pixel no inventory box claims and step 68 asserts the code is in the world's sacks and in no
   container.
5. **The shop half, against a real table and a real purse.** Steps 20-22 take a shipped weapon off a
   roster companion's shop doll into his pack, by a press-move-release drag whose move crosses
   `TapSlop` in frame-pixel space (the counterexample C1 fix, below); step 23 asserts it off. Steps
   24-26 put it back through the doll box (assert at 27); steps 28-30 take it off again through an
   ordinary pack drag (assert at 31); steps 32-34 stage it on the table (assert at 35); steps 36-38
   take a shelf item into the pack, unasserted before the walk to mission 30 that follows.

**Two of the defects this pass fixed make the scenario fail, and both failures were reproduced by
reverting the exact production line a reader would change.** Verbatim, from this worktree against
the `en` root:

```
=== R2 reverted: the shop raises the latch on the fallback arm only: exit 1 ===
againrom: headless: step 23 (assert_shop): the shop doll slot 1 = 265, want empty

=== R1 reverted: liveDriver stores a clone of the party: exit 1 ===
againrom: headless: step 87 (assert_inventory): inventory weapon_fallback = true, want false (figure [265 45571 0 0 0 46631 46895 6162 6420 43544 0 44060], equipment [0 45571 0 0 0 46631 46895 6162 6420 43544 0 44060])

=== fixed tree: exit 0 ===
```

Re-run at the round's eleventh pass, against the current scenario (87 steps, after the shop tremor
fix below moved the file's own step count from 77): both mutations reproduce at the same messages,
step 23 unchanged and step 77 renumbered to step 87 by the insertions between them, none of which
lie inside either leg the mutations exercise.

The scenario was written before any production change and failed on its first run for exactly these
two reasons, in this order: step 23 first, and step 87 (step 77 before this pass's insertions) once
step 23's clause was relaxed far enough to reach it.

**Both populations are driven in the same run.** The hero leg (steps 51-68) is an ordinary member
whose slot 1 holds a real item. The companion leg (steps 69-87) is `join:51`, a `DIV-070`-shaped
roster companion — `rosterTemplate` sets `PartyMember.Weapon` for him — whose weapon is taken off,
dropped on the ground, saved and reloaded; after the reload the doll must NOT draw the starting
weapon again, which is what step 87 asserts.

**What the scenario does not witness, stated rather than implied.** The mission-side materialization
of a DRAWN-but-unequipped starting weapon (item C / F4 below) needs a member whose slot 1 is empty
while `Weapon` is set and the latch is down. No member in the 666 save is in that state: the hero's
latch is seeded at `openMission` and both companions carry a real weapon in the array. The shipped
instance of that population is `NPC_Scrakan`, mission 151 (`DIV-070`), which this save cannot reach.
That behaviour is therefore witnessed by `pkg/game/weaponlatch_test.go`'s two cases and by the shop
half, which applies the same rule, and not by a shipped mission. R3 (the removed pack scan) is
likewise not observable in this scenario: after the ground drop the container no longer holds the
code, so the scan and the latch agree there. It is witnessed by
`TestASecondUnitOfTheStartingWeaponsCodeDoesNotRetireTheFallback`.

**The seat gate runs it.** `pipeline/check-scenarios.sh` takes `AGAINROM_IMPL` as of the master merge
below, so the eleventh pass ran it directly against this worktree, no junction needed. The script's
own bytes, its selection and its per-scenario command are the seat's, on both lawful roots:

```
check-scenarios: checkout <seat>/wt-1005b @ 4e0cc33
check-scenarios: selected 10 scenario(s) under <seat>/wt-1005b/scenarios
check-scenarios: asset root <seat>/gameversions/en
check-scenarios: ok   scenarios/0152-save666.json
check-scenarios: ok   scenarios/0154-synthetic-spells.json
check-scenarios: ok   scenarios/0155-mission10-escort.json
check-scenarios: ok   scenarios/0155-mission20-sweep.json
check-scenarios: ok   scenarios/0155-synthetic-melee.json
check-scenarios: ok   scenarios/0156-mission30-cure.json
check-scenarios: ok   scenarios/0159-mission40-join.json
check-scenarios: ok   scenarios/0163-chargen-mission10.json
check-scenarios: ok   scenarios/0163-mission-to-town.json
check-scenarios: ok   scenarios/1005-doll-and-shop.json
check-scenarios: ok (10 of 10)
```

```
check-scenarios: checkout <seat>/wt-1005b @ 4e0cc33
check-scenarios: selected 10 scenario(s) under <seat>/wt-1005b/scenarios
check-scenarios: asset root <seat>/gameversions/ru
check-scenarios: ok   scenarios/0152-save666.json
check-scenarios: ok   scenarios/0154-synthetic-spells.json
check-scenarios: ok   scenarios/0155-mission10-escort.json
check-scenarios: ok   scenarios/0155-mission20-sweep.json
check-scenarios: ok   scenarios/0155-synthetic-melee.json
check-scenarios: ok   scenarios/0156-mission30-cure.json
check-scenarios: ok   scenarios/0159-mission40-join.json
check-scenarios: ok   scenarios/0163-chargen-mission10.json
check-scenarios: ok   scenarios/0163-mission-to-town.json
check-scenarios: ok   scenarios/1005-doll-and-shop.json
check-scenarios: ok (10 of 10)
```

The `ru` root is new at the eleventh pass, and closes counterexample C3 without a direct fix: the
merge below picks up `App.HeadlessGameMenuAction` from story `1008` (master commit `4d44dd2`), which
fixed the CP866 SAVE/LOAD row selection two of these scenarios need on that root. The nine
pre-existing scenarios pass unchanged on this branch, which is the regression half of the same
instrument.

### The script-gap census

`go build -o /tmp/mr ./cmd/missionrun` and the milestone census, run from this worktree against the
`en` root, both before and after every change this story makes (the story makes none to `pkg/sim` or
to any script path, so before and after are the same build):

```
AGAINROM_ASSETS=<againrom>/gameversions/en /tmp/mr -mission 10 -trace -ticks 1 | grep -c UNSUPPORTED
AGAINROM_ASSETS=<againrom>/gameversions/en /tmp/mr -mission 20 -trace -ticks 1 | grep -c UNSUPPORTED
```

Both report **0**, for mission 10 and mission 20 alike. `pipeline/milestone-baseline.txt` carries no
`cannot run` row for `m10` or `m20` on either root, which is the same census reading zero
unsupported instants at the time the baseline was recorded. **Unchanged** at round 1 — expected,
since round 1 touches no `pkg/sim` code and no script opcode.

Re-run at round 2's own landing, same build, same command, same root:

```
go build -o /tmp/mr ./cmd/missionrun
AGAINROM_ASSETS=<againrom>/gameversions/en /tmp/mr -mission 10 -trace -ticks 1 | grep -c UNSUPPORTED   # 0
AGAINROM_ASSETS=<againrom>/gameversions/en /tmp/mr -mission 20 -trace -ticks 1 | grep -c UNSUPPORTED   # 0
```

Both still report **0**. **Unchanged** — round 2's own `sim.Command` additions (`KindDropCarried`,
`KindDropWorn`) are new command KINDS, not script opcodes; the census counts script nodes this build
cannot run, and this story adds no script opcode either round. `missionrun`'s own AI driver never
issues a drop command (it is a player-only gesture, both this story's own claim and `contract.md`'s
own scope), so the census exercises the SAME `stepWorld` dispatch switch the two new `case` arms
were added to, without exercising those two arms specifically — the census is a script-coverage
instrument, not evidence for this story's own feature, and is not claimed as such.

This story's own result is not a script-gap number: `contract.md` states it directly — "The equipment
doll answers which item a pixel belongs to, on both screens that draw it" — and round 1's half of
that is witnessed by the full `pkg/ui` and `pkg/game` test suites (below), run against the SAME
production code paths a player's mouse drives: `dollFigureSlotAt`, `hoveredItemInfoAt`,
`TakeInventoryDollUnequip`, the drag machine in `command.go`, and `refreshDollDrag`/`unequipFromDoll`
in `pkg/game/world.go`. No test in this story builds a stand-in for any of these; every assertion
runs the function a player's own gesture would run, through `Viewer.command`/`Viewer.step` exactly as
`pkg/ui`'s own pre-existing gesture tests do (`command_test.go`, `marquee_test.go`).

A driven, on-screen witness of round 1 — hovering, pressing and dragging the doll in a running
mission — was not performed as part of this story: this lane has no window to observe, and the
owner's own machine was not driven to avoid sending synthetic input into a window not confirmed
foreground. `builds/current/` is rebuilt at landing; the owner's own play is the confirming witness
this closure defers to.

**Round 2's own witness follows the same shape.** Every new test calls the actual production
function through its actual call chain, not a stand-in: `pkg/sim/drop_test.go` calls `sim.Step`
with `KindDropCarried`/`KindDropWorn` commands, the SAME entry point a real mission's own tick uses;
`pkg/ui/doll_test.go`'s two new ground-drop cases (`TestDragFromPackReleasedOutsideEveryBoxDropsToTheGround`,
`TestDragFromDollReleasedOutsideEveryBoxDropsToTheGround`) drive `Viewer.command` through a full
press-move-release cycle, exactly as a mouse would; `pkg/ui/shopdrag_test.go`'s three App-level tests
drive `App.step` the same way, through `app.go`'s own `stepTown`.

**One crossing is deliberately untested at the `pkg/game` level, by an established convention this
story did not introduce.** `equipFromPack`'s own test file states the rule directly (`equip_test.go`,
its own header comment): the one added statement crossing from a `pkg/game` drain function into
`pkg/ui` (`mw.view.TakeInventoryEquip()`) is `pkg/ui`'s own surface to witness, and reaching it from
`pkg/game` "would mean simulating a press through a machine this task does not own." `enqueueEquip`
— not `equipFromPack` — is `pkg/game`'s own tested entry point. `unequipFromWorn` and
`unequipFromDoll` follow the identical pattern (neither is called from any `pkg/game` test either).
`dropFromInventory` (`mw.view.TakeInventoryDrop()` then `mw.enqueueDrop(...)`) is this story's own
instance of the SAME crossing, and is left at the same boundary for the same reason: `enqueueDrop` is
tested directly (`pkg/game/drop_test.go`), and the one-line forward it receives from
`TakeInventoryDrop` carries no branch of its own — nothing a separate `pkg/game`-level test could
catch that the `pkg/ui`-side drag tests above and the `pkg/game`-side command tests below do not
already catch on either side of the same line.

Re-run at the seventh pass, same commands, same root, on the merged tree:

```
AGAINROM_ASSETS=<seat>/gameversions/en mr -mission 10 -trace -ticks 1 | grep -c UNSUPPORTED   # 0
AGAINROM_ASSETS=<seat>/gameversions/en mr -mission 20 -trace -ticks 1 | grep -c UNSUPPORTED   # 0
```

Both still **0**, and `pipeline/milestone-baseline.txt` still carries no `cannot run` row for `m10`
or `m20` on either root. **Unchanged**, which is this story's honest claim about that census: the
story's own result is a scenario that now runs, not a script-gap number that falls.

A driven, on-screen witness — dragging an item off the map with a mouse, in a window — was not
performed: this lane has no window to observe, and the owner's own machine was not driven, since no
window could be confirmed foreground at the instant of a synthetic event. The headless scenario above
drives the same production dispatch a mouse drives and is not offered as a substitute for having seen
it. `builds/current/` is rebuilt at landing and the owner's own play is the confirming witness this
closure defers to.

## Test suite

`go build ./... && go vet ./... && gofmt -l $(git ls-files --cached --others --exclude-standard '*.go') && go test -trimpath -count=1 ./...` is clean: build, vet and gofmt report nothing, every package's tests pass, including every pre-existing `pkg/ui` and `pkg/game` test unmodified by this story. `bash scripts/check-no-game-assets.sh` reports clean.

New and extended tests, each verified by reverting the specific production line it claims to witness
and observing the named failure, then restoring the line (working tree returned to a clean `git
status` after every revert):

| Test | File | Reverted line | Failure produced |
|---|---|---|---|
| `TestDollFigureSlotAtReadsTheMask` | `pkg/ui/doll_test.go` | `dollFigureSlotAt`'s `return n - 1, true` → `return n, true` | Both occupied-slot assertions fail: `dollFigureSlotAt over slot 1's own pixels = (1,true), want (0,true)` and the slot-2 case similarly off by one |
| `TestHoveredItemInfoAtAsksTheDollBox` | `pkg/ui/doll_test.go` | `hoveredItemInfoAt`'s doll-check block (`pkg/ui/itempopup.go`) removed | `hoveredItemInfoAt over slot 1 = ([],false), want ([slot one],true)` |
| `TestDragPackCellToDollEquips` | `pkg/ui/doll_test.go` | `command.go`'s `dragFromPack` release case: `v.invEquipRequest = v.dragCandIdx + 1` removed | `TakeInventoryEquip = (0,false), want (1,true)` |
| `TestDragDollSlotToPackUnequips` | `pkg/ui/doll_test.go` | `command.go`'s `dragFromDoll` release case: `v.invDollUnequipRequest = v.dragCandIdx + 1` removed | `TakeInventoryDollUnequip = (0,false), want (0,true)` |
| `TestPressOnADollSlotWithoutSlopUnequips` | `pkg/ui/doll_test.go` | `command.go`'s tap case: `v.invDollUnequipRequest = slot + 1` removed | `TakeInventoryDollUnequip = (0,false), want (0,true)` |
| `TestDollSubjectSubstitutesTheSuppressedFigure` | `pkg/ui/doll_test.go` | `dollSubject`'s substitution guard: `if v.dollSuppressOwner == e.ID && ...` prefixed with `false &&` | `dollSubject while slot 3 is suppressed` returns the ordinary figure and `suppressSlot:0` instead of the suppressed one |
| `TestComposeUnitFigurePaintsInFigureDrawOrder` (mask addition) | `pkg/game/figures_test.go` | `composeUnitFigure`'s main-loop `paintFigureLayerMasked(base, layer, mask, n)` → `paintFigureLayer(base, layer)` | `mask.At(0,0) = (0,false), want (1,true)` — the picture's own colour assertion, unchanged in the same test, stays green, proving the mask assertion is an independent witness and not derived from the colour check |
| `TestComposeInventorySubjectPaintsEveryOccupiedSlotInFigureDrawOrder` (mask addition) | `pkg/game/inventory_test.go` | `composeInventorySubject`'s main-loop `paintFigureLayerMasked(subject.Figure, layer, subject.SlotMask, n)` → `paintFigureLayer(subject.Figure, layer)` | `SlotMask.At(0,0) = (0,false), want (1,true) — the weapon slot, painted last` — again with the picture's own colour assertion unaffected |
| `TestHoverDuringADollDragReadsTheSuppressedMask` (new, round-1 review fix) | `pkg/ui/doll_test.go` | `dollFigureSlotAt`'s suppression check: `if v.dollSuppressOwner == v.invSubject.ID && v.dollSuppressSlot != 0 && v.dollSuppressPic != nil { mask = v.dollSuppressMask }` removed | `dollFigureSlotAt over the lifted slot's own pixels while suppressed = (0,true), want no slot` and `hoveredItemInfoAt over the lifted slot while suppressed = ([slot one],true), want nothing` |
| `TestDragReleasedOnTheWornBoxStillReturnsToOrigin` (renamed at round 2's own landing; superseded `TestDragReleasedElsewhereReturnsToOrigin`, whose own premise — a release outside every box does nothing — round 2's ground drop made false. This row's own citation was stale until the round-2 review, 2026-08-16, item 5: it still named the superseded test and a line `command.go` no longer carries in that shape) | `pkg/ui/inventory.go` | `inventoryCaptures`'s worn-box check: `if box, ok := v.wornBox(); ok && p.In(box)` → `if box, ok := v.wornBox(); false && ok && p.In(box)` | `TakeInventoryDrop = (worn=false,idx=0,true), want nothing — the worn box is an inventory box` |
| `TestHoverDuringADollDragNamesTheExposedSlot` (new, at the landing) | `pkg/ui/doll_test.go` | `dollFigureSlotAt`'s `mask = v.dollSuppressMask` → `mask = v.invSubject.SlotMask` | `dollFigureSlotAt over the exposed layer while slot 1 is suppressed = (0,true), want (6,true)`. Under the second mutation, the suppressed branch returning `0, false`: `= (0,false), want (6,true)` |
| `TestClearingTheTopSlotExposesTheLowerSlotInBothPictureAndMask` (new, at the landing) | `pkg/game/inventory_test.go` | `paintFigureLayerMasked`'s mask write gated on the pixel being unset, so the mask keeps the FIRST layer to paint it | `SlotMask.At(0,0) = (12,true), want (7,true)`. That mutation also reddens three pre-existing tests, so this test's lever on the compositor is not a new one; what it adds is the lifted composition, which nothing else builds |

The last two rows were added at the landing, after the adversarial review's closing observation that
`TestHoverDuringADollDragReadsTheSuppressedMask` would not, on its own, discriminate a correct fix
from an implementation answering nothing whenever a suppression is active. Both mutations were run
before anything was written: the shipped test reddens on each, the second on its third assertion,
which queries a slot the suppression does not touch. The gap the mutations did expose is a different
one. Both of that test's fixtures store the same value at every pixel where either is non-zero, so
no assertion over them can fail on a suppressed mask built without the layers UNDER the lifted one.
That is the case `DIV-085`'s stated property turns on once a doll carries stacked layers. The two
tests above cover it, one per side of the seam.

`TestMageCloakPrimaryIsBehindTheBodyAndSecondaryIsInFront` (`pkg/game/inventory_test.go`) was
extended with the same class of mask assertion at the same three pixels its pre-existing colour
assertions already check (the cloak's primary, its secondary, and the body covering both); it passes
and was not separately mutation-witnessed, since it exercises the mage-cloak canvas-rebuild branch
of the same `paintFigureLayerMasked` helper the two reverts above already exercise in the ordinary
branch.

`TestDragFromInventoryOverTheMapDoesNotStartBoxSelect` was not mutation-witnessed against a specific
reverted line: it proves a PRE-EXISTING mechanism (`v.invGrab`, story 0140) continues to hold under
this story's new drag code, and there is no line this story added whose reversion would plausibly
break it. Checked directly: the `v.invGrab` lines carry no diff in this story.

## Round 2 test suite

`go build ./... && go vet ./... && gofmt -l $(git ls-files --cached --others --exclude-standard
'*.go') && go test -trimpath -count=1 ./...` is clean on the whole tree, both rounds together: build,
vet and gofmt report nothing, every package's tests pass. `bash scripts/check-no-game-assets.sh`
reports clean.

Test file totals, this story's own tests against master's pre-1005 count (0 for a new file): `pkg/sim/drop_test.go` 12 (new file), `pkg/game/drop_test.go` 5 (new file), `pkg/game/shop_test.go` 12
(2 added, `RemoveFromShelf`), `pkg/game/shopdrag_test.go` 23 (new file; 15 at round 2's first pass,
8 more at the second: `shopSlot1Code`'s four call sites — counterexample 1 — the table→doll pair —
counterexample 3 — and the two direct shelf/pack pairs — counterexample 5), `pkg/ui/doll_test.go` 13
(3 added: the two ground-drop release arms and the multi-unit-selection hit-test fix below),
`pkg/ui/shopscreen_test.go` 17 (6 added; 5 at round 2's first pass, 1 more at the second: the
character block's own border refusal, minor), `pkg/ui/shopdrag_test.go` 8 (new file; 6 at round 2's
first pass — the App-level drag machine, its origin-icon lookup, its icon lifecycle and its Escape
case — 2 more at the second, counterexample 4's two tremor cases), `pkg/game/world_test.go` 1 (new
to this story at the second pass: counterexample 2's fallback-consistency test, four independently
mutation-witnessed points in one function).
Each row below is a production line reverted, the test run, the exact failure recorded, and the line
restored — working tree returned to a clean `git status` after every revert:

| Test | File | Reverted line | Failure produced |
|---|---|---|---|
| `TestDropFromContainerOutsideTheWindowLandsAtTheDroppersOwnCell` | `pkg/sim/drop.go` | `dropToGround`'s window test: `if dx > 2 \|\| dy > 2` → `if dx > 3 \|\| dy > 3` | `sack at (8,5), want (5,5) — the dropper's own cell, 3 east is outside the window` |
| `TestDropFromContainerDecrementsAStackAboveOne`, `TestDropFromEquipmentClearsTheSlotAndPlantsTheCode`, `TestDropTouchesNoOtherSlotAndNoOtherEntity` | `pkg/sim/drop.go` | `dropFromContainer`'s `Count--` → `Count -= 2`, and `dropFromEquipment`'s `w.equipment[i][slot-1] = 0` removed (two mutations, one run) | All three tests fail: the stack count is wrong, the slot is not cleared, and the touched-nothing-else check reads a slot the mutation left standing |
| `TestEnqueueDropWornAppendsACommandForTheNamedSlot` | `pkg/game/world.go` | `enqueueDrop`'s worn branch: `Spell: uint16(idx + 1)` → `Spell: uint16(idx)` | `pending[0] = {KindDropWorn 7 9 9 0}, want {KindDropWorn 7 9 9 1}` |
| `TestRemoveFromShelfTakesOneElementAndShiftsTheRest` | `pkg/game/shop.go` | `RemoveFromShelf`'s `append(items[:i:i], items[i+1:]...)` → `append(items[:i:i], items[i:]...)` | the removed index's own item reappears in the shifted result |
| `TestShopEquipDisplacesTheOldWornItemBackToThePack` | `pkg/game/shoproom.go` | `shopWearInto`'s displacement block (`if old != 0 { t.setShopPackItems(...) }`) removed | the displaced item is absent from the pack after the swap |
| `TestShopEquipFromShelfRefusesWhenThePurseCannotPay` | `pkg/game/shoproom.go` | `shopEquipFromShelf`'s purse check prefixed with `false &&` | the shelf item is bought and worn despite an insufficient purse |
| `TestShopUnequipToTableRefusesWhenTheTableIsFull` | `pkg/game/shoproom.go` | `shopUnequipToTable`'s two full-table refusals ignored (`GrowTablePlace`/`PutOnTable` results discarded) | the doll slot is cleared and the table silently exceeds `ShopTablePlaces` |
| `TestShopEquipFromPackRefusesTheWrongClass` | `pkg/game/shoproom.go` | `shopWear`'s class check prefixed with `false &&` | the wrong-class item is worn instead of refused |
| `TestShopDragDispatchesTheFourRecognisedPairs` | `pkg/game/shopview.go` | `ShopDrag`'s doll-to-shelf case routed to `shopUnequipDoll` instead of `shopUnequipToTable` | the item lands in the pack instead of on the table — caught only after the test was strengthened, below |
| `TestShopSuppressDollCachesAndRecomposesOnChange` | `pkg/game/shopview.go` | `refreshShopDrag`'s cache guard (`if want == t.shopSuppressSlot { return }`) removed | the cached-figure-pointer assertion fails: a second call for the same slot recomposes instead of reusing the cache |
| `TestShopScreenShowsTheSuppressedFigureOverTheOrdinaryOne` | `pkg/game/shopview.go` | `ShopScreen`'s substitution block prefixed with `false &&` | `ShopScreen did not substitute the suppressed figure` |
| `TestShopDollSlotAtReadsTheMask`, `TestShopGridControlAtRecognisesTheDollAndTheThreeGrids` | `pkg/ui/shopscreen.go` | `shopDollSlotAt`'s `return n - 1, true` → `return n, true` | Both tests fail: the marked-pixel slot answers one high, and the grid test's doll case answers the wrong index |
| `TestShopHoverLinesAnswersTheDollSlotFirst` | `pkg/ui/shopscreen.go` | `ShopHoverLines`' doll-first branch condition suffixed with `&& false` | the doll's own `SlotInfo` line is not returned; the function falls through to the (empty) grid test |
| `TestTheApplicationDragsFromTheDollToAShelfCell` | `pkg/ui/app.go` | `stepTown`'s release case: `moved >= TapSlop` → `moved >= TapSlop+1000` | no `ShopDrag` call is recorded; the gesture never reaches the crossed-TapSlop branch |
| `TestTheApplicationDragReleasedOutsideEverySurfaceIsANoOp` | `pkg/ui/app.go` | `stepTown`'s destination check: `if dest, ok := shopGridControlAt(...); ok` → always taking the branch | a `ShopDrag` call is sent for a release that named no recognised surface |
| `TestDragFromPackReleasedOutsideEveryBoxDropsToTheGround`, `TestDragFromDollReleasedOutsideEveryBoxDropsToTheGround` | `pkg/ui/command.go` | Both ground-drop trigger conditions, `!v.inventoryCaptures(...)`, prefixed with `false &&` | Both tests fail: no ground-drop request is raised for a release outside every inventory box |

Rows below are the adversarial review's own must-fix items, follow-ups and one gap found while
landing the response, each mutation-witnessed the same way — a production line reverted, the test
run, the failure recorded verbatim, the line restored:

| Test | File | Reverted line | Failure produced |
|---|---|---|---|
| `TestShopDollSlotAtRefusesThePlateAndTheChevronsEvenWhereTheMaskIsOccupied` (Counterexample 1) | `pkg/ui/shopscreen.go` | `shopDollSlotAt`'s plate/chevron refusal, `if p.In(shopNamePlateRect) \|\| p.In(shopPickerPrevRect) \|\| p.In(shopPickerNextRect)`, prefixed with `false &&` | All three excluded points answer a slot instead of none: `shopDollSlotAt at (515,445) (inside (513,443)-(599,475)) = (4,true), want none` and the same for the two chevron rects |
| `TestComposeShopScreenDrawsTheDragIconUnderTheCursor` (Counterexample 2) | `pkg/ui/shopscreen.go` | `ComposeShopScreen`'s drag-icon blit, `if hasDrag && dragIcon != nil && hasHover`, prefixed with `false &&` | `(300,100) = {R:8 G:9 B:12 A:255}, want the icon's own opaque red — centred on hover` |
| `TestShopDragOriginIconReadsEachSurfacesOwnPicture` (Counterexample 2) | `pkg/ui/app.go` | `shopDragOriginIcon`'s table case, `return v.Table[origin.Index].Icon` → `return v.Pack[origin.Index].Icon` | `table: shopDragOriginIcon({Kind:5 Index:3}) = <nil>, want` the table icon's own pointer — the case read the pack's own slice instead |
| `TestTheApplicationCapturesTheDragIconAtTapSlopAndClearsItOnRelease` (Counterexample 2) | `pkg/ui/app.go` | `stepTown`'s capture line, `if a.shopDragIcon == nil && a.shopDragMoved >= TapSlop { a.shopDragIcon = shopDragOriginIcon(...) }`, prefixed with `false &&` | `shopDragItemPresent = (<nil>,false) past TapSlop, want the doll's own SlotIcon[3]` |
| `TestEscapeDropsAnArmedShopDragBeforeUnwindingTheRoom` (follow-up 2) | `pkg/ui/app.go` | `App.step`'s Escape branch: `a.clearShopDrag()` call removed | Three assertions fail: `shopDragArmed is still true after Escape`, `shopDragIcon is still set after Escape`, `shopDragOrigin = {...} after Escape, want the zero value` |
| `TestRefreshShopDragAppliesTheSameWeaponFallbackAsComposeShopFaces` (follow-up 4) | `pkg/game/shopview.go` | `refreshShopDrag`'s `member.Weapon` slot-1 fallback block disabled (`if false { ... }`) | `the suppressed figure carries no slot-1 pixel; refreshShopDrag dropped the Weapon fallback composeShopFaces applies` |
| `TestShopScreenNeverSubstitutesAnotherMembersSuppressedFigure` (follow-up 3) | `pkg/game/shopview.go` | `ShopScreen`'s substitution guard: `i == t.shopSuppressMember &&` removed from the condition | `ShopScreen for member 1 = 0x...c0, want member 1's own ordinary figure 0x...40` and `ShopScreen substituted member 0's suppressed figure for member 1` |
| `TestShopDragDispatchesTheFourTablePairs` (the five-place-grid determination) | `pkg/game/shopview.go` | `ShopDrag`'s three new table-cell case arms removed together, then each removed alone | Together: `shelf-to-table drag left table = [], want the shelf's own item staged, not Mine`. Alone: the pack-to-table arm's own removal reddens the pack-to-table assertion, the shelf-to-table arm's own removal reddens the shelf-to-table assertion, and the table-off arm's own removal reddens the table-to-pack assertion — each arm has exactly one test path that only it satisfies |
| `TestDropFromAnEntityOutsideBoundsPlantsNoSack` (follow-up 1) | `pkg/sim/drop.go` | `dropToGround`'s fallback-cell re-check, both `sackFault` calls after the window test, removed | `Sacks() = 1, want 0 — an out-of-bounds entity's own cell must not become a sack` |
| `TestDollFigureSlotAtAnswersUnderAMultiUnitSelectionLedByTheSubject` (found while landing, see the section below) | `pkg/ui/inventory.go` | `dollFigureSlotAt`'s gate reverted from `v.dollSubject()`'s own figure test back to `v.inventoryEligible()` | `dollFigureSlotAt over slot 1 under a two-unit selection led by the subject = (0,false), want (0,true)` |

Rows below are the round's own second-pass adversarial review (2026-08-16): five must-fix
counterexamples and two minors, each mutation-witnessed the same way:

| Test | File | Reverted line | Failure produced |
|---|---|---|---|
| `TestShopEquippedCodeAppliesTheSameWeaponFallbackAsComposeShopFaces` (counterexample 1) | `pkg/game/shopview.go` | `shopEquippedCode`'s slot-1 arm, `code, _ := shopSlot1Code(worn, t.shopPartyMember(i))` → `code := data.ItemCode(worn[0])` | the hover text for a fallback-worn slot 1 answers empty instead of the fallback weapon's own lines |
| `TestShopUnequipDollTakesOffAFallbackWeaponAndClearsIt` (counterexample 1) | `pkg/game/shoproom.go` | `shopUnequipDoll`'s slot-1 read routed through `shopSlot1Code` reverted to a direct `worn[0]` read | the doll refuses to unequip a fallback-worn slot 1 (`shopUnequipDoll(1) = false, want true`) and `member.Weapon` stays set |
| `TestShopUnequipToTableTakesOffAFallbackWeapon` (counterexample 1) | `pkg/game/shoproom.go` | `shopUnequipToTable`'s slot-1 read routed through `shopSlot1Code` reverted to a direct `worn[0]` read | the same refusal, staged to the table instead of the pack |
| `TestShopWearIntoDisplacesAFallbackWeaponRatherThanLosingIt` (counterexample 1) | `pkg/game/shoproom.go` | `shopWearInto`'s slot-1 `old` value routed through `shopSlot1Code` reverted to a direct `worn[slot-1]` read | the fallback weapon is gone after the swap instead of appearing in the pack: `shopPackItems() = [], want [the fallback weapon's own code]` |
| `TestSwitchInventorySubjectAppliesTheSameWeaponFallbackAsTheFigure` (counterexample 2, four independently reverted points in one test) | `pkg/game/world.go` | (1) `openMission`'s `SlotInfo` built from `mw.currentFigureEquipment()` reverted to `mw.currentEquipment()`; (2) `switchInventorySubject`'s same substitution reverted; (3) `mw.invFigureEquipment` seeded from `eq` instead of `figureEq`; (4) `refreshEquipment`'s guard built from `mw.currentEquipment()` instead of `mw.currentFigureEquipment()` | Each reversion reddens its own assertion: `openMission`'s own `SlotInfo[0]` empty over a fallback-worn slot 1; `switchInventorySubject`'s `SlotInfo[0]` empty; `invFigureEquipment.Occupied(1) = false, want true`; `refreshEquipment`'s guard erases `SlotInfo[0]` on a no-op refresh |
| `TestShopEquipFromTableWearsAMinePlaceForFree`, `TestShopEquipFromTableBuysAndWearsANonMinePlace` (counterexample 3) | `pkg/game/shopview.go` | `ShopDrag`'s new `TableCell → Doll` case, `return t.shopEquipFromTable(from.Index)`, removed | `ShopDrag({TableCell 0},{Doll 0}) = {}, want {Msg:"worn"}` (Mine) and the purchase message (non-Mine); the table place is left untouched in both |
| `TestTheApplicationATremorWithinOneShelfCellIsATapNotADrag`, `TestTheApplicationATremorReturningToTheDollIsATapNotADrag` (counterexample 4) | `pkg/ui/app.go` | `stepTown`'s release arm: the `dest.Kind == origin.Kind` branch removed, `dragShop(origin, dest)` called unconditionally | Both tests fail: `clicked = [], want [0]` (shelf) and `clicked = [], want [3]` (doll) — the release reaches `ShopDrag` with `from.Kind == to.Kind`, unrecognised, and the click is silently dropped |
| `TestShopDragPackToShelfStagesOnTheTable`, `TestShopDragShelfToPackStagesOnTheTable` (counterexample 5) | `pkg/game/shopview.go` | `ShopDrag`'s `PackCell → ShelfCell` and `ShelfCell → PackCell` arms narrowed back to `TableCell`-only destinations | Both releases answer `ui.TownAction{}`, no-op: neither pair reaches `shopFromPack`/`shopClickShelfCell` |
| `TestShopDollSlotAtRefusesTheCharacterBlocksOwnBorderEvenWhereTheMaskIsOccupied` (minor) | `pkg/ui/shopscreen.go` | `shopDollSlotAt`'s border refusal, `if p.X == shopFigureRect.Min.X \|\| p.X == shopFigureRect.Max.X-1 \|\| p.Y == shopFigureRect.Max.Y-1`, removed | All three border lines answer a slot instead of none: `shopDollSlotAt at (480,250) (the character block's own border) = (4,true), want none`, and the same for the right column and the bottom row |

**Two lines in `pkg/ui/app.go` were mutation-tested and found NOT independently witnessed by any
test, honestly recorded rather than left as an unstated claim of coverage.** `a.shopDragIcon = nil`
appears once in the `PrimaryPressed` arm and once in the `PrimaryReleased` arm (`stepTown`). Removing
either ALONE — the other line left in place — leaves the full `pkg/ui` suite green: the press-arm
reset guards against a `PrimaryPressed` frame following another `PrimaryPressed` frame with no
release between them, a sequence this build's own input driver cannot produce, since a release always
clears `shopDragArmed` first; the release-arm reset writes a value `shopDragItemPresent` already
reads as absent the instant `shopDragArmed` goes false on the line above it, so nothing downstream can
observe the difference. Both lines are kept as defence in depth, `dollSuppressOwner`'s own precedent
one file over, and neither is claimed as mutation-witnessed. This is stated here rather than silently
building a witness for it, on the standing rule that less rigor means claiming less, never claiming
more than what was verified.

**`TestShopDragDispatchesTheFourRecognisedPairs` did not discriminate on its first run.** Its
original form asserted only the shelf-to-doll and doll-to-pack pairs, plus the two no-op cases; the
doll-to-shelf pair (`shopUnequipToTable`, `DIV-089`) was named in the test's own doc comment
("the two doll-origin pairs") but never exercised. The mutation above — routing that pair to
`shopUnequipDoll` instead — passed clean on the unstrengthened test. The test was extended with a
third case (re-arm the doll directly, drag it to a shelf cell, assert the item lands on the table
and not the pack); the same mutation, re-applied, now reddens it. This is recorded here rather than
silently fixed, on the standing rule that a returned claim about test coverage is checked by running
the mutation, not by reading the assertion.

`TestShopUnequipDollTakesItOffIntoThePack` (direct call) and `TestShopUnequipDollRefusesAnEmptySlot`,
`TestShopUnequipToTableStagesTheItemForSale` (direct call) were not separately mutation-witnessed:
each exercises the same `shopWornSlots`/`shopUnequipDoll`/`shopUnequipToTable` bodies the rows above
already revert lines in, through a different caller (a direct method call rather than `ShopDrag` or
`ShopClick`), and the guards they check (an empty slot, the price staged correctly) are read, not
computed, by those same bodies.

## Round-2 adversarial review

The review of round 2's pushed sha (`789c8aa`) found two must-fix counterexamples, one
determination to make, five follow-up questions, and — while landing the response, diagnosing a
separate owner report — one further gap in round 1's own code that shares a file with this round's
own changes. Each item's own test row is in the table above; this section states what was found and
what changed, once per item, in `PROSE.md`'s "state facts directly" register.

- **Counterexample 1 — the name plate and the chevrons.** `shopDollSlotAt` already refused all
  three rects at the pushed sha; `TestShopDollSlotAtRefusesThePlateAndTheChevronsEvenWhereTheMaskIsOccupied`
  was added and mutation-witnessed to prove it, restoring `DIV-085`'s stated property for the shop's
  two overlays as well as its name plate.
- **Counterexample 2 — the cursor icon.** Also already built at the pushed sha, across four layers
  (`ShopScreenView.SlotIcon`, `shopDragOriginIcon`, `App.shopDragIcon`, `ComposeShopScreen`'s two new
  parameters); four tests were added and each layer mutation-witnessed separately. `DIV-090` is
  retired spent-and-closed; `spec.md`'s FR-9 "Not built" paragraph, which had reasoned the omission
  was out of scope, is rewritten to describe the built mechanism.
- **The five-place-grid determination.** `SHOP-TRAY-025` and `SHOP-SCREEN-031` read together: code 4
  of `ITEM-CMD-007`'s vocabulary is this build's table (a five-cell grid on both sides), codes 5-8
  are the shelf. Table-drag was judged in scope — the owner's own directive says "any item," and the
  original's own dispatcher treats the tray and the shelf symmetrically in one opcode — and built: the
  three new `ShopDrag` case arms reuse `shopFromPack`/`shopClickShelfCell`/`shopOffTable`, the SAME
  functions a plain click on those cells already calls. `DIV-091` records the addition.
  `spec.md`'s new "The five-place grid" section carries the full reasoning.
- **Follow-up 1 — `dropToGround`'s bounds fallback.** Was unguarded at the pushed sha. Fixed: the
  fallback cell is now re-checked against `sackFault` before a sack is planted there, and the item is
  discarded rather than planted off the grid on the (currently unreachable) fault branch.
  `TestDropFromAnEntityOutsideBoundsPlantsNoSack` is new and mutation-witnessed.
- **Follow-up 2 — Escape and `a.shopDragArmed`.** Already cleared at the pushed sha
  (`clearShopDrag`, called from `App.step`'s Escape branch).
  `TestEscapeDropsAnArmedShopDragBeforeUnwindingTheRoom` is new and mutation-witnessed.
- **Follow-up 3 — the suppression substitution's member check.** Already present at the pushed sha
  (`i == t.shopSuppressMember`), but the code's own comment claimed the guard was simply unreachable,
  which the review flagged as unverified. Re-examined: the guard IS reachable at the model's own API
  (`ShopSuppressDoll` then `shopStepMember`, no `composeShopFaces` between), though not through
  today's input wiring, since a picker step and a held drag share one physical button. The comment is
  corrected to the more precise finding, and
  `TestShopScreenNeverSubstitutesAnotherMembersSuppressedFigure` drives the guard through that API
  path and is mutation-witnessed.
- **Follow-up 4 — `refreshShopDrag`'s weapon fallback.** Already present at the pushed sha, matching
  `composeShopFaces`'s own `member.Weapon` slot-1 fallback.
  `TestRefreshShopDragAppliesTheSameWeaponFallbackAsComposeShopFaces` is new and mutation-witnessed.
- **Follow-up 5 — two stale doc rows.** `closure.md`'s own citation of the superseded
  `TestDragReleasedElsewhereReturnsToOrigin` and `pkg/sim/sack.go`'s `pourSack` doc comment (naming
  one caller where there are three) were both already corrected at the pushed sha.

**A further gap, found while diagnosing an owner report and landed on this same branch because it
touches a file this round already holds** (`pkg/ui/inventory.go`): `dollFigureSlotAt` was gated on
`inventoryEligible()` — the worn box's and the pack bar's own test, `len(present) == 1 &&
present[0].ID == v.invSubject.ID` — while `dollBox` (and so `inventoryCaptures`, which decides
whether a press belongs to the inventory window at all) followed `dollSubject`'s own wider rule: any
selection whose first present member is the subject. An ordinary box-select catching the subject
first, alongside any other unit, left the doll drawing the subject's own composed figure while the
hit test answered nothing for every point on it — a press swallowed into the window's own handling
and then refused there, `DIV-085`'s stated property broken again, with the picture on screen correct
and only the gate wrong. Checked against the reading offered: confirmed by direct reading of
`dollSubject` (`pkg/ui/inventory.go:683-701`), `inventoryEligible` (`:606-612`) and `dollFigureSlotAt`
(`:1141`), and by mutation — reverting the fix to the old gate reddens the new test. Fixed:
`dollFigureSlotAt` now calls `v.dollSubject()` directly and refuses unless it answers a non-nil
`figure` — exactly the branch that draws the composed picture, ordinary or suppressed, and no other
branch. The worn box and the pack bar keep `inventoryEligible`'s narrower gate unchanged, since both
compose per-slot icons this build only ever builds for the party's own subject alone, which a
multi-unit selection leaves nothing correct to draw for either.
`TestDollFigureSlotAtAnswersUnderAMultiUnitSelectionLedByTheSubject` is new, drives the actual
production entry points (`dollFigureSlotAt`, `hoveredItemInfoAt`, a full press-release gesture through
`dollPress`/`tapAt`) under a two-unit selection set the same way the file's own existing
"selection moved off the subject" case already sets `v.sel`, and is mutation-witnessed against the
gate itself. No shipped test asserted the old gate's behaviour under a multi-unit selection
specifically: the full `pkg/ui` suite passed unchanged both before and after the fix, since every
pre-existing doll test used either a single-unit selection or a selection moved to one different unit
entirely. No new `DIV` id: this is a defect against `DIV-085`'s own stated property, round-1's own
convention for the suppressed-mask fix restated.

## Round-2 adversarial review, second pass

The review of the branch's second pushed sha found five must-fix counterexamples and two minors.
Each item's own test row is in the table above; this section states what was found and what
changed, once per item.

- **Counterexample 1 — the shop doll's slot 1 drawn but dead.** `composeShopFaces` and
  `refreshShopDrag` both apply a `member.Weapon` slot-1 fallback before composing the figure;
  `shopWornSlots` and every reader that consulted it directly — `shopEquippedCode`,
  `shopUnequipDoll`, `shopUnequipToTable`, and the `old` value `shopWearInto` read before
  overwriting slot 1 — did not. The chain was checked end to end rather than fixed on sight: this
  is a shipped companion state (`rosterTemplate` sets `PartyMember.Weapon` unconditionally; a worn
  array that never folds a starting weapon into slot 1 is not a constructed fixture, `DIV-070`'s own
  `NPC_Scrakan` finding under counterexample 2 confirms the same shape reaches shipped content), not
  only a fixture built to exercise it. `shopSlot1Code` (`pkg/game/shoproom.go`) is the two
  compositors' own rule, shared, returning whether the answer came from the fallback so a writer
  knows which field a displaced code came from. Fixed, four call sites, four tests.
  A fifth defect was found while fixing this one and is not separately counted: `shopWearInto`'s own
  `old := worn[slot-1]` read 0 for a fallback-worn slot 1, so wearing a new item into slot 1 lost the
  old weapon outright rather than displacing it to the pack. `shopWearInto` now reads `old` through
  `shopSlot1Code` too and clears `member.Weapon` when the displaced code came from it.
- **Counterexample 2 — the same disagreement on the mission doll.** `buildInventorySubject`
  applies the fallback to the figure; `SlotInfo` (built at `openMission` and at
  `switchInventorySubject`) and `refreshEquipment`'s change-guard read `mw.currentEquipment()`
  directly, no fallback. Settled by reading, not assuming: `rosterTemplate`
  (`pkg/mapload/spawn.go:1063`) sets `Weapon: w` unconditionally for a joining companion;
  `startingLoadout` (same file, line 573-576) separately clears `worn[cellWeapon]` for any row whose
  name contains `NPC`. `DIV-070` names mission 151's own companion row, `NPC_Scrakan`, a persistent,
  playable ally — his `Worn[0]` is cleared by that rule while his `Weapon` field is set by
  `rosterTemplate`, exactly the disagreement's own precondition. `switchInventorySubject` — the
  unit-selection-driven doll-subject switch, the mission's own counterpart to the shop's
  `shopMemberIndex`/`shopStepMember` — reaches any present party member, so the state is reachable
  through ordinary play once Scrakan has joined, not only through a constructed fixture. Fixed:
  `mw.currentFigureEquipment()` widens `currentEquipment()` by the same fallback, and `openMission`,
  `switchInventorySubject` and `refreshEquipment`'s guard now build `SlotInfo` through it.
  `mw.invEquipment` (`rearm`'s own tracker) and `mw.bodyEquipment` (`refreshAppearance`'s own
  tracker) deliberately keep reading the raw, un-widened `currentEquipment()` — both apply their own,
  narrower, correctly-scoped fallback rules and must not widen to this one (`spec.md` FR-2 states
  why for each). The false comment at the old `openMission` construction site ("composed from this
  exact live equipment above") is removed; the code it described was not always true.
- **Counterexample 3 — `TableCell → Doll` unrecognised.** `ShopDrag(table, doll)` answered
  `ui.TownAction{}`; `contract.md` names "a shop grid" without excluding the table, and `DIV-091`
  already treats the table as a full drag surface. Fixed: `shopEquipFromTable`
  (`pkg/game/shoproom.go`), the table's own take-and-wear gesture over `Shop.TakeOffTable`,
  `shopEquipFromPack`'s and `shopEquipFromShelf`'s own pattern restated for a fifth surface.
  `DIV-091` is extended, not superseded, to record the fifth pair.
- **Counterexample 5 — `ShelfCell ↔ PackCell` unrecognised.** `ITEM-CMD-007` was read directly
  (`cd research && go run ./tools/claim ITEM-CMD-007`) rather than assumed: the opcode space is a
  single move command whose source and destination codes are symmetric — 1 (equip), 2
  (container/pack), 3 (ground), 4 (shop tray/table), 4..8 (shelf) — on both sides of the same
  dispatcher. Every unwired pair among the five-place grid's own four surfaces was therefore this
  build's own asymmetry (round 1 wired three of the four sources/destinations, round 2's first pass
  added the table on both sides, but never wired the shelf and the pack to each other) and not a
  scope boundary the claim itself draws — wiring both directions is the smaller change to justify.
  Fixed: both directions route through the existing, already-tested staging functions
  (`shopFromPack`, `shopClickShelfCell`), the same unification precedent `DIV-089` already
  established for doll→shelf joining doll→table — no new economic logic (an unpriced pack insert,
  or a shelf-stock mutation this build has nowhere else) was invented for either direction.
  `DIV-092` records the addition.
- **Counterexample 4 — the 4px tremor swallowing a shop click.** `TapSlop` is 4 Manhattan pixels,
  high-water, latched every frame a press stays armed (`command.go:16`); a shop grid cell is 80
  pixels square, and the doll's own marked pixels sit far closer together, so an ordinary hand's
  tremor crosses 4 pixels while a gesture never leaves the surface it pressed. `stepTown`'s release
  arm judged any gesture that crossed `TapSlop` by where it lands, called `ShopDrag`, and `ShopDrag`
  has no case for a same-family pair (dragging a cell onto its own family names no shop action) —
  so the release was silently dropped, pre-empting the doll's own tap-to-unequip arm one case down
  in the same switch for a doll press that trembled back onto the doll. The seam offered a possible
  fix shape (an `origin == dest` fallback) but stated it was a guess, not a measurement, and asked
  where the fallback belonged and whether it interacted with the live suppression push
  (`a.flow.suppressShopDoll`) that runs before the release arm each frame. Read directly: the
  suppression push is gated on `a.shopDragOrigin.Kind == ShopControlDoll && a.shopDragMoved >=
  TapSlop`, computed BEFORE the release switch runs (`a.shopDragArmed` is still true at that point),
  so it fires once per frame regardless of how the release is later judged — it does not interact
  with which branch the release takes, only with whether the origin is the doll. `ShopDrag`'s own
  switch was re-read in full: after counterexamples 3 and 5, every one of the twelve possible
  ordered cross-family pairs among the doll, the shelf, the pack and the table is now wired, and
  same-family (not same-origin-point) is exactly what remains unwired — a narrower and more general
  test than "origin == dest," since it also covers a genuine attempted drag between two cells of one
  family (e.g. pack cell 0 to pack cell 3), which is equally not a shop action. Fixed: `dest.Kind ==
  origin.Kind` routes the release to `flow.clickShop(dest)` instead of `dragShop`, using the release
  point (not the press origin) — the same information a literal second click at that pixel would
  resolve, including for the doll, whose own hit test (`shopDollSlotAt`) already names the exact
  slot released on.
- **Minor — the character block's border painted over the doll.** `drawShopCharBlock` paints
  `outline(dst, shopCharRegion, invBorder)` after the figure; `shopCharRegion` shares its left edge,
  right edge and bottom edge with `shopFigureRect`, so the border overwrote those three lines of the
  figure's own pixels. Fixed: `shopDollSlotAt` refuses a point on any of the three lines before
  consulting the mask, the plate-and-chevron refusal (counterexample 1 of the review's first pass)
  restated for a third overlay.
- **Minor — a stale test-name citation.** `closure.md`'s own test-suite table cited
  `TestShopGridControlAtRecognisesTheDollAndTheTwoGrids`; the shipped test is named
  `TestShopGridControlAtRecognisesTheDollAndTheThreeGrids` (`shopscreen_test.go:492`, renamed when
  the table joined the shelf and the pack as a recognised grid at round 2's first pass). Corrected.

## Round-2 adversarial review, third pass

The review of the branch's third pushed sha found six must-fix production counterexamples. The
returned worktree also carried two documentation-accuracy corrections. Each item's evidence row is
in the table below; this section states what was found and what changed, once per item.

| Test | File | Reverted line | Failure produced |
|---|---|---|---|
| `TestTheApplicationDoesNotClickADifferentDollSlotAfterADollDrag` (counterexample 1) | `pkg/ui/app.go` | `stepTown`'s release arm: `dest == origin` → `dest.Kind == origin.Kind` | `dragged = [], want [{{DollSlot 3} {DollSlot 11}}]` — a genuine cross-slot same-family drag resolves through the destination's click path instead of the drag seam |
| `TestShopFallbackCannotAppendASecondWeaponToThePack`, `TestShopFallbackCannotStageASecondSellableWeapon`, `TestShopEquippingThePackedStartingWeaponDoesNotDisplaceADuplicate` (counterexample 2) | `pkg/game/shoproom.go` | `shopWeaponFallbackCode`'s `!t.shopWeaponFallback[i]` refusal removed, restoring an unconditional `member.Weapon` fallback | `shopUnequipDoll(1)` appends a second copy; `shopUnequipToTable(1)` stages a second sellable copy; `shopEquipFromPack(1)` leaves one displaced duplicate in the pack |
| `TestRefreshDollDragEquipmentSourceAgreesWithTheOrdinaryFigure` (counterexample 3) | `pkg/game/world.go` | `suppressedDollSubject`'s source changed from `currentFigureEquipment()` to `currentEquipment()` | Mask `(0,0) = (0,false), want (1,true)` and figure pixel `(0,0)` is base red rather than the fallback weapon's independent blue pixel |
| `TestAFallbackOnlySlotOneIsNotArmedAsADragSource` (counterexample 4, consumption side) **— REVERSED at the seventh pass on the owner's directive; the test is now `TestAFallbackOnlySlotOneIsTakenOffLikeAnyOther` and the guard is gone** | `pkg/ui/command.go` | The press-arm guard, `if !(slot == 0 && v.invSubject.WeaponFallback) { ... }`, unwrapped to an unconditional arm | `dragActive = true, dragCandKind = dragFromDoll, want false, dragNone — a fallback-only slot 1 armed a drag` |
| `TestSwitchInventorySubjectAppliesTheSameWeaponFallbackAsTheFigure`'s live equip/unequip extension (counterexample 4, production side) | `pkg/game/world.go` | `currentWeaponFallbackActive`'s `&& !mw.invWeaponEverEquipped` term removed | `WeaponFallback stayed true after slot 1 became real with the same code, want false` — the sticky tracker's own contribution is isolated from `currentFigureEquipment`'s bare `!occupied` test, which the earlier four assertions could not discriminate |
| `TestReleasingAnArmedDollDragOverTheHudToggleBarCancelsIt` (counterexample 5) | `pkg/ui/command.go` | The four reset lines in the `hudToggleCaptures`/`PrimaryReleased` branch (`v.dragActive = false`, `v.dragCandKind = dragNone`, `v.dragIcon = nil`, `v.invGrab = false`) removed | `dragActive` and `invGrab` remain true and `dragCandKind` remains `dragFromDoll` after release, instead of returning to the idle state |
| `TestInventoryCapturesTheReservedGroundOnlyWhileADollDragIsInFlight` (counterexample 6; renamed at the fifth pass, when counterexample C narrowed its contract) | `pkg/ui/inventory.go` | `inventoryCaptures`'s area fallback (the `wornBoxArea()`/`packBarArea()` branch behind `dollInventoryActive()`) removed | `inventoryCaptures over the pack bar's own reserved ground under a two-unit selection = false, want true`; a release there raises a stray ground drop, `worn=true idx=0 x=15 y=28` |
| `TestInventoryDoesNotCaptureInvisibleAreasWithoutASelection` (counterexample 6, negative control) | `pkg/ui/inventory.go` | `dollInventoryActive`'s own body replaced with `return true` unconditionally | `inventoryCaptures = true with no selection, want false — an invisible pack bar must stay map ground outside the subject-led multi-selection case` |
| (`docs/DIVERGENCES.md`, counterexample 7) | `docs/DIVERGENCES.md` | n/a — a documentation self-contradiction, not a mutable production line | The paragraph above `DIV-084` said `DIV-092` was "returned unused" while the row below it carries a live `OPEN` state; corrected, see below |
| (comment accuracy, counterexample 8) | `pkg/game/world.go` | n/a — a documentation comment, not a mutable production line | `currentFigureEquipment`'s doc comment named two narrower callers where the function has four, and left `refreshEquipment`'s and `refreshDollDrag`'s own widening unstated; corrected, see below |

- **Counterexample 1 — a genuine same-family drag resolved as a click.** The second pass's own
  `dest.Kind == origin.Kind` test (closing that pass's counterexample 4) matches both a tremor that
  never leaves its origin cell AND a real drag between two different cells of one family — pack cell 0
  to pack cell 3, shelf cell 0 to shelf cell 5, doll slot 3 to slot 8. Both resolved through
  `flow.clickShop(dest)`, so a genuine drag bought or unequipped whatever sat at the RELEASE cell
  rather than doing nothing, the exact defect `ShopDrag`'s own missing same-family case is supposed to
  leave inert. Fixed: `dest == origin` (`ShopControl` is comparable — `Kind`, `Index`, and `Shift`,
  which is false on both sides at this point in the frame) narrows the tap-not-drag test to the exact
  cell a tremor cannot leave. A drag that crosses to a different cell of the same family now falls to
  `dragShop` below, which `ShopDrag`'s own switch still answers with no case at all — a no-op, per the
  second pass's own reasoning for that case.
- **Counterexample 2 — the fallback duplicating an already-unequipped weapon.** `shopSlot1Code`'s
  fallback (second pass, counterexample 1) answers `member.Weapon`'s code whenever the worn array's
  own slot 1 reads empty, with no check of the member's pack. `PartyMember.Weapon` is copied
  unconditionally through `CarryParty`/`OwnParty` and never cleared by a real mid-mission
  `sim.unequip`: unequipping the starting weapon during a mission leaves `Carry.Equipped[0] == 0` AND
  `Carry.Items` already holding the weapon's own code, with `member.Weapon` still set. Every one of
  `shopSlot1Code`'s four callers answered the fallback for this state and minted a second copy of an
  item the player already carries. Fixed: `shopWeaponFallbackCode` (`pkg/game/shoproom.go`) answers
  the fallback only when the code is not already present in `shopMemberPack(member)`, and
  `shopSlot1Code` now calls it instead of reading `member.Weapon` directly. The fallback still fires
  for the state it exists to cover — a companion whose starting weapon was never folded into the
  array at all (`rosterTemplate`, `DIV-070`'s own `NPC_Scrakan`) — since that member's pack never
  holds the code to begin with.
- **Counterexample 3 — the mission doll's suppressed figure disagreeing with the ordinary one at
  slot 1.** `refreshDollDrag` composed the suppressed doll picture from `mw.currentEquipment()`,
  the raw array; `refreshEquipment` (called every tick a drag is not held) composes the ordinary
  figure from `mw.currentFigureEquipment()`, the same array widened by the slot-1 startWeapon
  fallback the second pass's own counterexample 2 wired everywhere else. A drag held on any slot
  OTHER than 1, for a member whose slot 1 is fallback-only, suppressed a figure missing the weapon
  layer the ordinary figure still drew the tick before — `DIV-085`'s stated property (the hit test
  and the picture agree) broken again, restated for the drag-held figure rather than the mask.
  Fixed: `refreshDollDrag` delegates to `suppressedDollSubject`, whose production source is
  `mw.currentFigureEquipment()`, the same source `refreshEquipment` already uses. `ui.Viewer`'s drag
  state remains private to package `ui`, so the game test does not invent a setter for it.
  `TestRefreshDollDragEquipmentSourceAgreesWithTheOrdinaryFigure` instead calls the production
  composition seam over a live `mapWorld`; replacing that seam's source with
  `currentEquipment()` reddens both an independent mask pixel and an independent figure pixel.
- **Counterexample 4 — a fallback-only slot 1 armed as a drag source.** Once counterexample 3 widens
  `refreshDollDrag`'s own source, a fallback-only slot 1 draws a weapon layer indistinguishable on
  screen from a real one — but nothing in the live entity's equipment array or any container backs
  it, so `sim.unequip` and `dropFromEquipment` both refuse it outright. A press-drag or a
  tap-to-unequip on that slot armed exactly as if the weapon were real, then resolved into a command
  the simulation silently ignores, leaving the picture unmoved with no explanation offered to the
  player. Fixed on both sides. Production: `ui.InventorySubject` gained `WeaponFallback bool`
  (`pkg/ui/inventory.go`), set from `mw.currentWeaponFallbackActive()` (`pkg/game/world.go`) at
  `openMission`, `switchInventorySubject`, and `refreshEquipment`'s own guarded recompute — the same
  three sites that already build `SlotInfo` from `currentFigureEquipment`. Consumption:
  `command.go`'s press-arm block refuses `dragCandKind, dragCandIdx = dragFromDoll, slot` when
  `slot == 0 && v.invSubject.WeaponFallback`; the slot stays hover-only, FR-3's own independent
  promise. The consumption side is mutation-witnessed by
  `TestAFallbackOnlySlotOneIsNotArmedAsADragSource`. **This refusal was reversed at the seventh pass**
  (owner directive, 2026-08-17: pressing an item on the doll takes it off): the slot is now armed like
  any other and the mission side materializes a real item, `spec.md` FR-2. The production rule is mutation-witnessed by
  `TestSwitchInventorySubjectAppliesTheSameWeaponFallbackAsTheFigure`: removing
  `currentWeaponFallbackActive`'s `!mw.invWeaponEverEquipped` gate makes the fallback return after a
  live equip followed by a live unequip, leaving `WeaponFallback`, `invFigureEquipment`, and
  `SlotInfo[0]` wrong together.
- **Counterexample 5 — an armed drag surviving a release over the HUD toggle bar.** `hudToggleCaptures`'
  own `PrimaryReleased` branch cleared `v.held` and returned, without touching `dragActive`,
  `dragCandKind`, `dragIcon` or `invGrab` — the one early-return path in `command()` that bypasses the
  inventory's own release handling below, where every other surface resets those four fields. An
  armed doll or pack drag released over the toggle bar stayed armed on every later frame: `invGrab ||
  ...` kept reading true regardless of where the cursor went next, so the NEXT primary release
  anywhere off the inventory reached the drag-release switch with a stale origin and resolved as a
  ground drop at that unrelated release's own cell. Fixed: the same four fields are reset,
  unconditionally, in this branch too. `TestReleasingAnArmedDollDragOverTheHudToggleBarCancelsIt`
  arms a drag off the bar, releases on the bar, and asserts a later, unrelated release raises no
  stray drop.
- **Counterexample 6 — the reserved inventory ground losing its own capture under a multi-unit
  selection.** `dollBox` follows `dollSubject`'s wide rule (any selection whose first present member
  is the subject draws the doll — the round-2 review's own multi-selection fix, above); `wornBox` and
  `packBar` follow `inventoryEligible`'s narrow rule (exactly one selected entity) and draw nothing
  under a multi-unit selection. `inventoryCaptures` asked `wornBox`/`packBar` directly, so the SCREEN
  GROUND those two boxes occupy on every other selection stopped capturing input the moment a
  multi-unit selection left them undrawn — a release there read as "outside every inventory box" and
  raised a stray ground drop at a cell the player was never looking at, while a doll-origin drag could
  still be armed in the same selection state (`dollFigureSlotAt` shares `dollSubject`'s wide gate).
  Fixed: `wornBoxArea`/`packBarArea` (`pkg/ui/inventory.go`) are `wornBox`/`packBar`'s own rectangles
  with the `inventoryEligible` half of the gate dropped; `inventoryCaptures` now asks these instead,
  capturing the reserved ground even where nothing is drawn on it, while `wornBox`/`packBar`
  themselves — and every caller that composes their CONTENT — keep the narrow gate unchanged.
- **Counterexample 7 — `DIV-092`'s own paragraph contradicting its row.** The paragraph naming which
  ids `DIV-087` through `DIV-092` cover said `DIV-092` was "returned unused," while the row itself,
  three lines below in the same file, carries a live `OPEN` state (town shop / a direct shelf-pack
  drag). `DIVERGENCES.md`'s own allocation rule — "the next free id is the one after the highest seen
  in this file or in this paragraph" — reads the paragraph, not the row, so the self-contradiction
  could have carried forward into reissuing a live id; a returned id is never reissued in this
  project. Fixed: the paragraph now names `DIV-092` as spent at the second-pass review, matching the
  row, and states the correction separately with its own date.
- **Counterexample 8 — `currentFigureEquipment`'s own doc comment undercounting its narrow
  callers.** The comment named two callers that must NOT widen to this function's fallback
  (`rearm`'s stat recompute and `invWeaponEverEquipped`'s opening seed); the function actually has
  four such callers, `invFigureEquipment`'s own seed (openMission/switchInventorySubject, the same
  raw-array reason as the seed the comment already named) being the third, and `refreshAppearance`
  — the world-sprite compositor a mission runs continuously — being the fourth, never previously
  justified in writing at all. Widening `refreshAppearance` to this fallback would show a hero who
  genuinely unequipped a real weapon in slot 1 holding his STARTING weapon again on the map itself,
  since `refreshAppearance`'s own read carries no `everEquipped` gate the way `rearm`'s two readers
  do. Fixed: the comment enumerates all four narrow callers, states why each must not widen, and
  names the two callers that DO widen (`refreshEquipment`, `refreshDollDrag`) and why. No production
  behaviour changed; this is a documentation-accuracy fix — a comment that undercounts its own
  callers is the kind of premise a later change reads and trusts.

**A naming collision, found and corrected before push.** This pass's own fixes were first labelled
"round-2 adversarial review, second pass" in code comments and this file, the same label the EARLIER,
separately-landed wave above (counterexamples 1-5, the section preceding this one) already owns for a
different set of counterexamples. Found by reading the existing `spec.md` and this file before
writing new prose into them, not by a tool. Corrected by relabelling every comment and heading this
pass added — and only the lines this pass added, checked against `git diff` on each file before and
after — to "third pass." Two further, smaller defects surfaced while relabelling and are recorded
here rather than silently fixed: a line-wrapped `second\n// pass` split across two comment lines in
three places (`pkg/game/world.go:1775`, `:2165`, `pkg/game/world_test.go:3420`) was missed by the
first relabelling pass, which matched the phrase on one line, and corrected in a second pass over the
same diff; and the relabelling script's own line-ending behaviour on this platform converted whole
files from LF to CRLF where it touched even one line, which `gofmt -l` then flagged for eleven files
until every affected file was rewritten back to LF.

## Round-2 adversarial review, fifth pass

The review at the branch's fifth pushed sha (`124efe4`) found nine counterexamples, lettered A
through I, assuming the feature incomplete. This is the fourth section in this file to record a
round-2 adversarial review, following the unlabeled first pass, the second pass and the third pass
above; the review count runs to five because `124efe4` itself landed a distinct fix (a double-click's
second press dragging onto a different pack cell after the pack reindexed under it, a pack-reindex
race unrelated to any of A through I) between the third pass and this one, without its own numbered
review section. Full technical detail for each item is in `spec.md`, at the FR its own fix belongs to
(FR-2 for A, B, I; FR-7 for C, D, E, F); this section states what was found, what changed, and the
mutation each witness catches, once per item.

| Test | File | Reverted line | Failure produced |
|---|---|---|---|
| `TestBuildInventorySubjectWeaponFallbackStaysOffOnceMaterialized` (counterexamples A, B) | `pkg/game/inventory.go` | `buildInventorySubject`'s `!member.WeaponMaterialized` check reverted to `member.Weapon != nil` alone | `unread` names the weapon's own layer and icon addresses — the fallback re-fires for a member whose materialization already latched |
| `TestSwitchInventorySubjectAppliesTheSameWeaponFallbackAsTheFigure`'s `WeaponMaterialized` extension (counterexamples A, B) | `pkg/game/world.go` | **This row was false and is corrected at the seventh pass.** It claimed the write-back was checked by removing it. Re-measured 2026-08-17: removing `resolveWeaponMaterialized`'s write-back leaves the ENTIRE suite green, because `rearm` raises the same bit a frame later. The line was unwitnessed | (nothing failed) — a witness was written for it at the seventh pass: `TestASeenRealSlotOneKeepsTheFallbackRetiredAfterAnUnequip`, below |
| `TestOwnPartyCarriesWeaponMaterializedAsAPlainValue` (counterexamples A, B, structural) | `pkg/mapload/party_test.go` | n/a — Go value-copy semantics give this guarantee structurally; no mutation run |
| `TestInventoryCapturesTheReservedGroundOnlyWhileADollDragIsInFlight` (counterexample C) | `pkg/ui/inventory.go` | `inventoryCaptures`'s added `v.dragCandKind != dragFromDoll \|\|` clause removed | A plain press-and-release with no drag armed is captured by `packBarArea`, want it to reach the map's own order handling |
| `TestInventoryCapturesTheReservedGroundOnlyWhileADollDragIsInFlight` (counterexample D) | `pkg/ui/command.go` | `dragFromDoll`'s release arm reverted from `v.packBarArea()` to `v.packBar()` | `TakeInventoryDollUnequip()` answers `(0,false)` after a release inside the wide, narrow-empty area, want `(idx,true)` |
| `TestADoubleClicksSecondPressStillEquipsAfterATremorWithinTheCell` (counterexample E) | `pkg/ui/command.go` | `originSame` computation and its two leading switch cases removed | `TakeInventoryEquip()` answers `(0,false)` after a tremor-armed release back on the double-click's own cell, want `(2,true)` |
| `TestDollTapStillUnequipsAfterATremorWithinTheSlot` (counterexample F) | `pkg/ui/command.go` | Same removal as counterexample E — one mechanism, both witnesses | `TakeInventoryDollUnequip()` answers `(0,false)` after a tremor-armed release back on the doll's own slot, want `(0,true)` |
| `TestRefreshAppearanceAgreesWithTheDollOverAFallbackWeapon` (counterexample I) | `pkg/game/world.go` | `refreshAppearance`'s `eq := mw.currentFigureEquipment()` reverted to `mw.currentEquipment()` | `mw.bodyEquipment.Occupied(1)` answers `(false, true)`, want `(true, true)` — the map sprite drops the fallback weapon the inventory doll still shows |

- **Counterexamples A and B — the fallback's own materialization tracked as re-derivable state.**
  Diagnosed by reading `weaponFallbackSpent`, `shopMemberPack`'s pack scan, and
  `buildInventorySubject`'s own inline pack scan side by side: all three ask whether the starting
  weapon's code is PRESENTLY in the pack or array as a stand-in for whether the fallback has EVER
  materialized, which disagree once a materialized weapon later leaves the party's own pack or array
  entirely (sold, dropped, or carried across a mission boundary). Fixed with the persisted
  `PartyMember.WeaponMaterialized` latch (`spec.md` FR-2, above, has the full call-site list).
  `buildInventorySubject`'s own inline pack scan — found in this pass, not flagged by the review
  itself, which named only the shop and world.go call sites — was the third site carrying the same
  defect; the review's own diagnosis generalized to it once the pattern was known.
- **Counterexample C — the multi-selection capture gate too wide.** The third pass's own
  `dollInventoryActive()` gate (its counterexample 6) reserves ground correctly but arms on
  SELECTION STATE alone, not on a drag actually starting there, so an ordinary move order under that
  selection was swallowed. Fixed by adding `v.dragCandKind == dragFromDoll` to the gate.
- **Counterexample D — the release arm asking the wrong rectangle.** `dragFromDoll`'s own release
  check used `packBar()`, the narrow box `inventoryEligible` empties under the same multi-selection
  `packBarArea` exists to cover; a release counterexample C's own fix now correctly lets through
  still resolved to nothing. Fixed by asking `packBarArea()` instead — the same rectangle
  `inventoryCaptures` gates on.
- **Counterexamples E and F — a tremor within one cell losing the gesture.** `TapSlop` (4 px
  Manhattan) is small against a 48-pixel pack cell or a comparably large doll slot, so an ordinary
  hand tremor arms a drag without leaving the cell or slot the gesture began on; the release switch
  tried `v.dragActive` before checking whether the release named the drag's own origin, and neither
  the drag-release case nor the tap case recognises "released on my own origin while armed" as a
  case, so the gesture resolved to nothing. Fixed with `originSame`, one mechanism for both the pack
  cell (E) and the doll slot (F), modeled on the shop's own `dest == origin` guard
  (`app.go:1677`, third pass counterexample 1). Counterexample F is pre-existing on `master`, not
  introduced by this story — `command.go`'s own doll press-arm code predates it — and was previously
  undisclosed asymmetrically against the shop's own documented tremor case (second pass
  counterexample 4).
- **Counterexample I — the map sprite and the inventory doll disagreeing over a fallback weapon.**
  `currentFigureEquipment`'s own doc comment (corrected this pass) undercounted its narrow callers at
  two where four remain; `refreshAppearance` was a genuine fifth reader still on the raw,
  un-widened `currentEquipment()`, one FR-2's own third-pass fix (counterexample 3, the suppressed
  drag figure) had not reached. Fixed by widening `refreshAppearance` to `currentFigureEquipment()`
  and its own `bodyEquipment` tracker's two seed sites to match, found and fixed beyond the review's
  own stated finding for internal consistency between the tracker's write and its read.

None of the nine counterexamples contradicts a claim cited in `contract.md`'s Claims table or an
existing `DIV` row's own stated property: all nine are defects in this build's own already-authored
mechanism against its own stated rule, the same class the third pass's own six counterexamples were
(`docs/DIVERGENCES.md`'s Reconciliation note on the third pass, above). No new `docs/DIVERGENCES.md`
row is added.

**A divergence-id allocation check was run and found no collision to record.** This story's worktree
was briefed that `DIV-093`/`DIV-094` are allocated to a concurrent, unmerged `hotfix-spell-effects`
branch and that `DIV-095` is the next free id from that vantage point. This worktree's own copy of
`docs/DIVERGENCES.md` carries no row past `DIV-092` — the highest allocated by this branch's own
history — so applying the file's own stated allocation rule in isolation would name `DIV-093` next,
which disagrees with the `DIV-095` briefing; expected, since the hotfix branch's allocation has not
merged into this worktree and so cannot appear in this copy of the file. No row was written by this
pass, so the disagreement had no consequence here; it is recorded for whichever pass next needs a
divergence id from this branch, so that pass reads the file's tail again rather than reusing either
number from this paragraph.

## Reconciliation

`DIV-084`, `DIV-085`, `DIV-086` are recorded in `docs/DIVERGENCES.md`, each as `UNKNOWN`: research
carries no claim on ROM1's own inventory drag gesture, doll hit-testing mechanism, or doll hover
popup. Each row states the implemented (authored) mechanism and a revisit condition — a claim naming
the corresponding ROM1 mechanism, should one surface.

No claim cited in `contract.md`'s Claims table is contradicted by round 1: `DLG-FIGURE-021` (no layer
selector) is honoured by FR-6's equipment-clearing approach rather than any compositor change;
`ITEM-EQUIP-006`/`ITEM-ARMSLOT-031`/`ITEM-WEAR-05x` are unmodified, since every equip and unequip in
round 1 routes through the existing `mapWorld.enqueueEquip`/`enqueueUnequip`.

Round 2 adds five rows, `DIV-087` through `DIV-091`, all in `docs/DIVERGENCES.md`:

- `DIV-087` (`UNKNOWN`) — the shop screen carries no ROM1 claim on what a release of a
  purse-on-shelf item onto the doll does; FR-9 wears it under the same rule as a pack item.
- `DIV-088` (`UNKNOWN`) — no ROM1 claim on the release-outside-the-window ground-drop gesture
  during a mission; FR-8 implements `ITEM-DROP-008`'s Chebyshev window with the dropper's own cell
  as fallback, per the owner's own directive in `contract.md`.
- `DIV-089` (`UNKNOWN`) — no ROM1 claim on a doll-to-shelf drag in the shop; FR-9 stages the item on
  the table (`shopUnequipToTable`) rather than the pack, matching the shop's existing sell path.
  `TestShopDragDispatchesTheFourRecognisedPairs`'s third sub-case (added this round, see Test suite)
  is this row's own witness.
- `DIV-090` (`FIDELITY-DEBT`, `CLOSED` at the round's own adversarial review) — recorded a shop drag
  carrying no cursor-follow icon. Retired spent-and-closed: `contract.md:10-12`'s own result clause
  names "a shop grid" in the same sentence as the cursor, which the round's original reasoning had
  missed. The cursor-icon mechanism (`ShopScreenView.SlotIcon`, `shopDragOriginIcon`,
  `App.shopDragIcon`, `ComposeShopScreen`'s two new parameters) is built and mutation-witnessed, see
  the Round-2 adversarial review section above.
- `DIV-091` (`UNKNOWN`, extended at the review's second pass) — no ROM1 claim states that a
  PLAYER gesture, as opposed to the dispatcher's own opcode, reaches the shop tray (`ITEM-CMD-007`
  code 4) the way this build's table-drag now does; the shape of the row is `DIV-087`'s and
  `DIV-089`'s own, restated for the table. `TestShopDragDispatchesTheFourTablePairs` is the row's
  own witness for the four pairs the first pass built; `TestShopEquipFromTableWearsAMinePlaceForFree`
  and `TestShopEquipFromTableBuysAndWearsANonMinePlace` witness the fifth pair (table→doll,
  counterexample 3) the second pass added to the same row.
- `DIV-092` (`UNKNOWN`, new at the review's second pass) — no ROM1 claim states that a PLAYER
  gesture reaches a direct shelf-to-pack or pack-to-shelf move; `ITEM-CMD-007`'s own opcode space
  pairs the container code symmetrically with every shop code, so this build wires both directions
  through the SAME staging functions the table's own pairs already use (counterexample 5).
  `TestShopDragPackToShelfStagesOnTheTable` and `TestShopDragShelfToPackStagesOnTheTable` are this
  row's own witness.

Both rows were allocated to this story's round 2 at its first landing; `DIV-092` was reserved and
unused at that point and is spent here.

The third pass adds no new `docs/DIVERGENCES.md` row. The six production counterexamples and two
documentation corrections are defects against an already-authored mechanism or its own record,
not a new gap against ROM1 research. Counterexamples 1 through 6 correct code that disagreed with
its own stated rule (`DIV-085`'s
hit-test-agrees-with-picture property, restated three times over — the shop drag machine, the
fallback-vs-pack duplication, the mission doll's suppressed figure, the drag-arm gate, the HUD
toggle bar, the multi-selection ground); items 7 and 8 correct this project's own
documents (`docs/DIVERGENCES.md`'s paragraph, `currentFigureEquipment`'s doc comment) rather than
code. No claim cited in `contract.md`'s Claims table, and no existing `DIV` row's own stated
property, is contradicted by any of the eight.

### Reconciliation, seventh pass

**`DIV-111` is this pass's one new row** (`docs/DIVERGENCES.md`, `DEVIATION`, `OPEN`): the doll draws
a starting weapon that the equipment array does not hold, and taking it off materializes a real item
into the container without ever filling slot 1. The state does not exist in ROM1 —
`ITEM-DEATH-012` makes the `NPC` substring test a death-time corpse rule — and it exists here only
because `DIV-070` applies that test at spawn. The row names `DIV-070`'s own fix as its revisit
condition: repairing the debt makes the fallback, the latch and the materialization all deletable.
This mismatch was created at the round's third pass and carried by no ledger row until now, which
is the defect `docs/DIVERGENCES.md`'s own opening rule names.

**The row's own "Implemented behaviour" cell was wrong at this pass and is corrected at the eighth**
(2026-08-17, adversarial review counterexample C3, below): it stated the drawn weapon "contributes
nothing to `rearm`'s stat computation while it is drawn," which is the opposite of the mechanism —
the fallback DOES arm the member's stats while drawn, on both `Rearm`'s and `PartyLoadout`'s own
`ResolveEquipmentLoadout` call, exactly as a living NPC armed from his template would be. The actual
gap this pass's own production code left open was on the OTHER side of a take-off: `PartyLoadout`,
the mission-construction path, ignored the latch and kept resolving the fallback weapon after it had
been taken off, so a member who sold his starting weapon arrived at his next mission re-armed with
it. See the eighth-pass section below for the fix and its witness.

The id is `DIV-111` and not `DIV-093`: `DIV-093`/`DIV-094` belong to the spell-effects hotfix
branch, `DIV-095`..`DIV-102` are reserved to story `1008`, `DIV-103` to the teleport-explored
hotfix, and `DIV-104`..`DIV-107` were spent by story `1007` (merged into master ahead of this
pass, along with `DIV-108`..`DIV-110`, returned unused). `pipeline/next-div-id.sh` reported
`DIV-111` as the next free id, highest mention `DIV-110`, at the point this row was taken; the
same check was re-run immediately before push.

No other claim in `contract.md`'s Claims table is contradicted by this pass. It adds no `pkg/sim`
change, no command kind and no format version.

### Reconciliation, eighth pass

The eighth pass adds no new `docs/DIVERGENCES.md` row. Its two production defects (C1, the pin;
C3, the mission-open re-arm) and its one bookkeeping defect (C2, the duplicate id) are all corrected
against this build's own already-authored mechanism, not against a new gap versus ROM1 research.
`DIV-111`'s own "Implemented behaviour" cell is corrected in place (`docs/DIVERGENCES.md`), on the
same rule the third-pass and fifth-pass sections above already followed for a row whose text, not
its type or status, was wrong: the row stays `DEVIATION`/`OPEN`, its owner directive and ROM1 gap
are unchanged, and only the sentence describing what the code does is rewritten.

C1 (the research pin) and C2 (the duplicate divergence id) carry no divergence of their own: a pin
merge and an id renumbering are bookkeeping, not a behavioural gap against ROM1. C2's own
resolution is recorded in the previous section, where `DIV-104` becomes `DIV-111` throughout this
document and `docs/DIVERGENCES.md`; the merge that supplied C1's forward pin (`4aae01f`) also
supplied `headlessActivateWorldMap` (`pkg/ui/headless.go`), which the
`"walk out to mission 30"` pointer step at `scenarios/1005-doll-and-shop.json:315` depends on and
which was absent on this branch before the merge.

## Round-2 adversarial review, seventh pass

The pass was given five items: build the integration witness FIRST, end the starting-weapon latch
defect that had returned four times, bring the mission side to the shop's rule for a fallback-only
slot 1 on the owner's directive, and repair four documentation defects. The witness was built before
any production line was changed, and it failed twice on the branch as it then stood — the two
failures are quoted verbatim in the Integration witness section above.

### Every write and read of the starting-weapon latch

`PartyMember.WeaponMaterialized` (`pkg/mapload/start.go:300`, `:289` at an earlier pass) records
that a member's starting weapon has been turned into a real item. This is the whole set of sites in
the shipped tree.

| Site | `file:line` | Kind | What it does |
|---|---|---|---|
| `materializeStartingWeapon` | `pkg/game/weaponlatch.go:47` | **write** | The only assignment in package `game`. Nil-tolerant. |
| `rearm` | `pkg/game/world.go:2646` | write, through the writer | Slot 1 occupied at a stat recompute means the weapon is real; raises the latch for the subject's own member. |
| `resolveWeaponMaterialized` | `pkg/game/world.go:2289` (`2259` at the ninth pass; shifted by intervening edits) | write, through the writer | The seed's write-back: the first time slot 1 is seen occupied for real, the latch is raised so a later unequip cannot bring the drawn weapon back. |
| `materializeFallbackWeapon` | `pkg/game/world.go:2114` (`2102` at an earlier pass) | write, through the writer | The mission-side take-off: appends the code to the container and raises the latch. |
| `shopWearInto` | `pkg/game/shoproom.go:325` (`323` at an earlier pass) | write, through the writer | Slot 1 written by the shop; whatever it displaced is real from that write on. |
| `shopUnequipDoll` | `pkg/game/shoproom.go:498` (`509` at an earlier pass) | write, through the writer | Any slot-1 take-off to the pack, real or fallback (R2). |
| `shopUnequipToTable` | `pkg/game/shoproom.go:544` (`558` at an earlier pass) | write, through the writer | Any slot-1 take-off to the table, real or fallback (R2). |
| `resolveWeaponMaterialized` | `pkg/game/world.go:2291` (`2261` at the ninth pass) | **read** | Returns the latch. `openMission` (`world.go:1072`, `1055` at the ninth pass) and `switchInventorySubject` (`world.go:1829`, `1805` at the ninth pass) reach it, seeding `mw.invWeaponEverEquipped`. |
| `canonicalizePartyAppearance` | `pkg/game/world.go:1339` | read | An unraised latch means the member is DRAWN holding his starting weapon: the function widens slot 1 with `member.Weapon.Code` before deriving the body, which is `currentFigureEquipment`'s own rule restated for an arbitrary party member rather than for the inventory subject. Added by the map-sprite appearance hotfix, 2026-08-22 (owner reports: a mage with no staff equipped drawn holding one; a fighter dressed in the shop walking onto the map bare). Reading the raw array here would draw DIV-070's population bare-handed on the map while their own doll shows them armed. |
| `shopWeaponFallbackCode` | `pkg/game/shoproom.go:278` (`268` at an earlier pass) | read | A raised latch means the shop offers no fallback code. |
| `spellClientClass` | `pkg/game/spell.go:184` | read | Story1119 resolves Cast capability from the live equipment/body-name class, including an unmaterialized starting weapon. It does not require loaded art or mutate the latch. |
| `missionDollEquipment` | `pkg/game/inventory.go:193` | read | A raised latch means the opening doll composes slot 1 from the array alone. Split out of `buildInventorySubject` (round-2 twelfth pass, 2026-08-17, C1b) so `world.go`'s own `invComposedEquipment` seed calls the same composition `buildInventorySubject` uses, instead of restating it. Takes the mission's world and the subject's entity id (round-2 twelfth pass, C1, second correction) so it can prefer a live read (`equippedIfAny`, `pkg/game/inventory.go:213`) over `mapload.EquipmentFromParty(member)`, which it now calls only as the no-live-entity fallback. |
| `PartyMember.WeaponMaterialized` | `pkg/mapload/start.go:300` (`289` at an earlier pass) | **persist** | A plain value field; carried by `clonePartyMember`'s value copy, `CloneParty`, `OwnParty` and `Snapshot.Party`'s gob encoding with no added plumbing. |
| `nativeCityHiredEquipmentMismatch` | `pkg/game/nativecity.go:786` | read | Story 1127 correction (R-2, `pipeline/reviews/story1127-pass1.md`): compares a live hired mercenary's own latch against the fresh tavern template `restoreHiredMercenaries` would rebuild him from; a mismatch refuses the native SAV rather than silently reverting the member's own materialization on the next load. Does not raise or lower the latch. |
| `capturePartyPolicy` | `pkg/game/currentpartypolicy.go:38` | read | Captures the current latch in SAV policy; the private policy field is `StartingWeaponSpent`, encoded under its existing wire name. |
| `currentPartyPolicy.member` | `pkg/game/currentpartypolicy.go:55` | write, through the writer | Restores a raised latch through `materializeStartingWeapon`; the zero constructor already represents an unspent fallback. |
| `capturePartyWeapon` | `pkg/game/currentpartyweapon.go:15` | read | Retains a starting fallback only for an empty slot whose latch is unraised. |
| `currentPartyMember.restoreFromState` | `pkg/game/currentpartyread.go:263` | read | Derives the current weapon view from ordinary equipment and restored materialization history. |
| `MemberWeapon` | `pkg/mapload/currentweapon.go:13` | read | The common current-weapon reader suppresses a previously spent starting fallback. |
| `projectCurrentPartyActorFields` | `pkg/game/savnativeattributes.go:36` | read | Reads the captured latch when deriving native Body from current worn items and permanent gains. |

**The enumeration is kept true by a test, not by a request to the next reader.**
`TestWeaponMaterializedHasOneWriter` (`pkg/game/weaponlatch_scan_test.go`) parses package `game`'s
own non-test source and fails on three things: an assignment to the field outside
`materializeStartingWeapon`, a mention of the field inside a function its table does not name, and a
table row naming a function that no longer touches the field. The table is an exact set and cannot
drift into a superset. A site added without a row fails the suite, and the failure message names this
document and asks for the `file:line` before the row is added. Four adversarial passes were spent on
sites that each set this bit their own way; a comment asking the next reader to remember would have
failed nothing.

### What changed

- **R3 — the seed was a container scan.** `weaponFallbackSpent` answered "the fallback is spent"
  when slot 1 was occupied OR the starting weapon's code was anywhere in the entity's stock. A
  container holds codes, not identities, so a second unit of the same code retired a fallback that
  had never fired, permanently. Fixed at the root: the scan is deleted and the function has one
  term, `eq.Occupied(1)`. It no longer takes the item stacks at all, and neither does
  `resolveWeaponMaterialized`, so no caller can reintroduce the scan by passing one.
- **R2 — the shop minted a second starting weapon.** `shopUnequipDoll` and `shopUnequipToTable`
  raised the latch on their `viaFallback` arm only. A member who took a REAL weapon off slot 1 left
  the latch down, `shopWeaponFallbackCode` then offered his starting weapon into the slot he had just
  emptied, and the next take-off appended a second unit of that code to the pack. Both sites now
  raise the latch for any slot-1 take-off, which is the rule `shopWearInto` has had since the fifth
  pass.
- **R1 — a mid-mission save dropped the latch.** `liveDriver` stored `mapload.OwnParty(party)`, a
  clone, so `Snapshot` wrote the clone while the mission wrote `ms.Party`. Fixed where the truth
  lives: `liveDriver` aliases the caller's own slice and names it in place through the new
  `mapload.NameParty`, and `OwnParty` is now `CloneParty` plus `NameParty`, so the two share one
  identity rule. **Nothing else on `PartyMember` was losing mid-mission writes**: the only writes to
  a `*PartyMember` during a mission are the three latch writes enumerated above, all reached through
  `missionPartyMember`, which resolves into `mw.mission.party`, the same slice. The alias covers any
  field a later story adds, because it removes the copy rather than adding a field to a copier.
- **Item C — a fallback-only slot 1 is taken off like any other.** On the owner's directive
  («При наведении на предмет на кукле мы можем на нее нажать — это снимает предмет»),
  `pkg/ui/command.go`'s refusal to arm a drag or a tap on such a slot is removed, and the mission
  side answers with `materializeFallbackWeapon`: the starting weapon's code is appended to the
  entity's CONTAINER through `ReplaceStock`, the latch is raised, and the unequip needs no simulation
  command because the item is already where an unequip would have put it. A ground drop issues
  `KindDropCarried` for the element just appended. The container and not the equipment array,
  deliberately: filling slot 1 would re-arm the member's stats and would restore the corpse drop that
  `startingLoadout`'s NPC rule exists to suppress. This is the shop's own rule, brought to the
  mission side.

### Mutation witnesses

Each mutation reverts the exact production line a reader would change, runs the suite, and is then
restored from a backup outside the repository. No `git stash` was used at any point: the stash stack
is one ref shared across every worktree of this repository and two other branches are live.

| # | Line reverted | Result |
|---|---|---|
| M1 | `liveDriver` stores `mapload.OwnParty(party)` again (`resume.go:291`) | RED `TestAMidMissionLatchReachesTheSave` |
| M2 | `shopUnequipDoll` raises the latch on the `viaFallback` arm only (`shoproom.go:509`) | RED `TestTakingARealWeaponOffInTheShopRetiresTheFallback` |
| M2b | `shopUnequipToTable` likewise (`shoproom.go:558`) | RED `TestStagingARealWeaponOnTheTableRetiresTheFallback` |
| M3 | `weaponFallbackSpent`'s container scan restored (`world.go`) | RED `TestASecondUnitOfTheStartingWeaponsCodeDoesNotRetireTheFallback` |
| M4 | `enqueueUnequip`'s `materializeFallbackWeapon` call removed (`world.go`) | RED `TestTakingTheDrawnStartingWeaponOffPutsARealOneInThePack` |
| M4b | `enqueueDrop`'s worn-arm call removed (`world.go`) | RED `TestDroppingTheDrawnStartingWeaponReachesTheGround` |
| M5 | `command.go`'s press-arm refusal restored | RED `TestAFallbackOnlySlotOneIsTakenOffLikeAnyOther` |
| M6 | The latch assigned outside `materializeStartingWeapon` | RED `TestWeaponMaterializedHasOneWriter` |
| M7 | `resolveWeaponMaterialized`'s write-back removed, leaving `return member.WeaponMaterialized || mw.weaponFallbackSpent(eq)` (`world.go:2259`) | **GREEN on first measurement: no test failed.** See below |
| S1 | M2, measured against the scenario instead of the suite | Scenario exit 1 at step 23 |
| S2 | M1, measured against the scenario instead of the suite | Scenario exit 1 at step 87 (step 77 before the eleventh pass's own in-place insertions) |

**M7 is reported as green rather than replaced with a mutation that reddens.** The fifth pass's own
table claimed that line was witnessed by an assertion inside
`TestSwitchInventorySubjectAppliesTheSameWeaponFallbackAsTheFigure`. Measured, it was not: `rearm`
raises the same bit a frame later, so removing the seed's write-back changes nothing any existing
test observes. The difference is observable only where the seed runs and `rearm` does not.
`TestASeenRealSlotOneKeepsTheFallbackRetiredAfterAnUnequip` (`pkg/game/weaponlatch_test.go`) drives
exactly that: select a member whose slot 1 is genuinely equipped, take the item off through `sim`
alone, and select him again. Without the write-back the second seed reads an empty slot 1, finds no
history, and draws the retired starting weapon back onto the doll. With that test present, M7 reddens
it and nothing else.

### Counterexample bookkeeping

The fifth-pass section states that the review found nine counterexamples lettered A through I and
then documents seven: A, B, C, D, E, F and I. **G and H are not recorded in this repository**, and
this pass could not recover them: the review's own report was returned to the seat and is not stored
in the implementation repo, in `pipeline/LOG.md` or in `PIPELINE-STATUS.md`. The count claim is
corrected here rather than in the fifth-pass section, which is a record of what was written at the
time: seven of the nine are documented and two are unaccounted for.

### Documentation corrections

- `spec.md` is canonicalized to as-built. It named `shopWeaponFallback []bool`,
  `resetShopWeaponFallbacks` and `ensureShopWeaponFallbacks`, none of which exists in any `.go` file;
  it named `weaponFallbackSpent(eq, stacks)` and `resolveWeaponMaterialized(member, eq, stacks)`,
  both of which lost their stacks parameter; it described `command.go`'s refusal, which the owner
  reversed; and it cited `TestInventoryCapturesTheReservedGroundUnderAMultiUnitSelection`, renamed at
  the fifth pass. Every identifier `spec.md` names in backticks is now present in the Go tree,
  checked mechanically over all 246 of them. FR-10 was added for the scenario vocabulary this pass
  built.
- The Runtime-state and Persistence/save-load rows of the matrix above are corrected. Runtime state
  claimed the latch was "read once and never re-derived" while its own seed scanned the container
  (R3). Persistence was PASS while a mid-mission save dropped the field (R1).
- `scenarios/README.md` documents version 4: `pointer`, `assert_inventory`, `assert_shop`, `window`,
  the eight point forms, the popup and one-tick-lag rules a scenario must obey, and the worked
  example.

## Round-2 adversarial review, eighth pass

The pushed seventh-pass sha (`e3a5ed7`) was reviewed and returned FAIL with three counterexamples:
a duplicate divergence id (C2), a research pin behind master (C1), and a mission-open re-arm defect
(C3). None contradicts a claim cited in `contract.md`'s Claims table or an existing `DIV` row's own
stated property; C3 is a defect in this round's own already-authored mechanism against its own
stated rule, the same class every prior pass's counterexamples were.

- **C2 — `DIV-104` collided with story `1007`'s own row of the same id**, landed to master seven
  minutes after this branch wrote its own. `DIV-104` becomes `DIV-111` throughout this document and
  `docs/DIVERGENCES.md`, taken from `pipeline/next-div-id.sh` at write time and re-confirmed
  immediately before push. See "Reconciliation, eighth pass" above.
- **C1 — the research pin (`d91a940`) was behind master's (`4aae01f`)**, thirteen research commits
  and one hotfix merge behind. Fixed by merging `origin/master` into this branch
  (`b6c477a`), which brought the pin forward and supplied `headlessActivateWorldMap`
  (`pkg/ui/headless.go`), a dependency of the integration witness (below). `git submodule status`
  reads no leading character after the merge — `4aae01f87685dc840988ca5b8bebde3ad591149e research`
  — and `bash pipeline/check-pin-forward.sh` reads `ok` for this branch.
- **C3 — mission construction re-armed a member with a weapon he had sold.** `mapload.PartyLoadout`
  (`pkg/mapload/loadout.go:97`) passed `everEquipped` to `ResolveEquipmentLoadout` as an
  unconditional `false`, on the premise — true before `WeaponMaterialized` existed — that mission
  construction is always a member's first resolution. The latch survives a mission boundary
  (`CarryParty`) and a save; the premise did not survive it. A member whose starting weapon was
  taken off and sold arrived at his next mission's own mint (`PartySpawnWithTable`, feeding the
  entity's `CombatBlock`) armed with its damage again, permanently, at every mission after. Fixed by
  passing `p.WeaponMaterialized` — the same bit `Rearm`'s own in-mission `everEquipped` parameter
  (`pkg/game/rearm.go:266`) already read correctly. See `spec.md` FR-2's own closing paragraph and
  the matrix corrections above (Simulation, Campaign/session, Interactions with existing
  mechanics).

### The write/read enumeration gains its first reader outside package `game`

`TestWeaponMaterializedHasOneWriter` (`pkg/game/weaponlatch_scan_test.go`) previously parsed
package `game`'s own non-test source only, so a site in `pkg/mapload` — where C3's own defect
lived — was invisible to it. It now parses `pkg/mapload` as well, keyed `"pkg.Func"` so the two
packages' function names cannot collide, and the sites table gains one row:

| Site | `file:line` | Kind | What it does |
|---|---|---|---|
| `mapload.PartyLoadout` | `pkg/mapload/loadout.go:97` | read | Mission construction's own resolution: a raised latch means an empty slot 1 is "taken off," not "not yet materialized," matching `Rearm`'s in-mission rule. |

Demonstrated to catch an uncatalogued site before the row above was added: with the row removed,
`TestWeaponMaterializedHasOneWriter` fails —

```
weaponlatch_scan_test.go:83: mapload.PartyLoadout mentions PartyMember.WeaponMaterialized at
..\mapload\loadout.go:97:74 and is not in weaponLatchSites.
```

— and passes once the row is restored.

### Mutation witness

| # | Line reverted | Result |
|---|---|---|
| M8 | `PartyLoadout`'s `everEquipped` argument reverted from `p.WeaponMaterialized` to the literal `false` (`pkg/mapload/loadout.go:97`) | RED `TestPartyLoadoutRespectsTheMaterializedLatch` — a latched member with an empty slot 1 resolves the fallback weapon instead of `nil`, and the bare/latched pair mint identical `Combat.DamageBase`/`DamageSpread` |

Both directions were run directly (`go test ./pkg/mapload/... -run TestPartyLoadoutRespectsTheMaterializedLatch -v`): red on the reverted line, green on the restored one.

### Matrix rows corrected

Simulation, Campaign/session and Interactions with existing mechanics are corrected above (Twelve-
aspect matrix) rather than left to read PASS on evidence that predates C3's own fix; each now names
the eighth pass and what changed. No row moves off PASS/N-A: the fix closes a defect in an
already-`PASS` aspect's own supporting evidence, not a new gap.

## Round-2 adversarial review, ninth pass

The pushed eighth-pass sha (`756fe66`) was reviewed and returned FAIL with one counterexample: the
same shape as C3, at a third call site of `mapload.ResolveEquipmentLoadout`.

- **The counterexample — `recomputeRaisedSkills` (`pkg/game/rearm.go`) read `everEquipped` as
  SUBJECT-SCOPED.** The live per-tick pass that folds a raised skill into an entity's hashed
  `CombatBlock` computed `everEquipped := mw.invSubjectSet && sim.EntityID(mw.invSubject.ID) == c.id
  && mw.invWeaponEverEquipped` — true only for the current inventory subject. A member who is not
  the subject always read `false`, so a member whose `PartyMember.WeaponMaterialized` latch was
  raised and whose slot 1 is empty was re-armed with the sold starting weapon's damage the first
  time any of his skill levels rose. The site's own comment stated the premise this was built on:
  "the only producer of `sim.KindUnequip` is the inventory screen's own door, which acts on
  `invSubject` alone." This story falsifies that premise twice over — a persisted latch that rides
  in from an earlier mission (`mapload.CarryParty`), and the shop's own picker
  (`shopUnequipDoll`/`shopUnequipToTable`/`shopWearInto`), which takes a weapon off or puts one on
  ANY party member shown, not only the mission's own subject.
  Fixed by reading `mw.resolveWeaponMaterialized(mw.missionPartyMember(c.id), eq)` in place of the
  subject-scoped compare. `mw.missionPartyMember(c.id)` is the LIVE element inside `mw.mission.party`
  — the same slice `mw.rearm` and the shop screen both write through — not
  `characterDerive.member`, a `mapload.PartyMember` VALUE copied once at mission open
  (`installCharacterDerivations`) and never updated for the life of the mission; reading
  `c.member.WeaponMaterialized` directly would still have returned the stale, mission-open answer.
  `mw.resolveWeaponMaterialized` also raises the latch the first time it observes slot 1 occupied
  for real, on `mw.rearm`'s own precedent, so this pass is one more place that observation can be
  made and must not go unrecorded. A roster entity outside `ms.Start.IDs` — `missionNotices.party`
  covers only the player's own party — makes `missionPartyMember` return nil, which
  `resolveWeaponMaterialized` degrades to `weaponFallbackSpent(eq)`'s present-state reading, the same
  answer this pass already gave every non-subject entity before the fix (`everEquipped` only matters
  when slot 1 reads empty).

### The population of `ResolveEquipmentLoadout`/`Rearm` callers, examined

The review named every resolution of a member's loadout in the tree and asked that each be verified
rather than trusted. All five are accounted for:

| Site | `everEquipped` | Finding |
|---|---|---|
| `mapload.PartyLoadout` (`loadout.go:97`) | `p.WeaponMaterialized` | Fixed at the eighth pass; unchanged here. |
| `recomputeRaisedSkills` (`rearm.go`) | live latch via `resolveWeaponMaterialized` | **The counterexample; fixed this pass.** |
| `Rearm`'s caller in `mw.rearm` (`world.go:2698`, `:2648` at an earlier pass) | `mw.invWeaponEverEquipped` | Confirmed correct: this call operates on `mw.invSubject.ID` alone, so subject-scoped IS the subject, not a narrowing of it. |
| `Rearm`'s two callers in `restoreOriginalActorStock` (`originalsave.go:296`, `:302`) | `armed := worn[0] != 0` | Examined against this pass's own failure class and found NOT the same one: `armed` is read once, synchronously, from the SAME `worn` array `ReplaceStock` wrote two lines above, and consumed in the same call — no tick, no player gesture and no other caller runs between the write and the read. `everEquipped`'s only branch in `ResolveEquipmentLoadout` fires when `code == 0`, and `worn[0] == 0` is exactly `armed`'s own false case, so a nonzero restored code always takes the code-driven branch regardless of `armed` — the value only matters, and is only read, on the branch its own construction keeps consistent. Recorded in place at `originalsave.go`. |
| `Rearm`'s caller in `cmd/missionrun/main.go:819` | `false` (constant) | Rechecked and confirmed: both of this tool's party sources — `game.MissionPartyAs`'s fresh mint and `game.ResumeOriginalSave`'s `restoredMember` (`pkg/game/originalparty.go`) — leave `WeaponMaterialized` at its zero value on every entry, and this file issues no `sim.KindUnequip` anywhere, so slot 1 can only move from empty to occupied within one run, never back. The constant matches the code, not only the comment. |
| `mapload.blockFor`'s person arm (`fromalm.go:843`) | `false` (constant) | Confirmed correct, unchanged: a map placement is not a `mapload.PartyMember` and carries no latch; "a map placement has never had anything taken off" is still true. |

### Documentation corrections, ninth pass

- `spec.md:694-695` named `shopSlot1Code(worn *[sim.EquipSlots]uint16, member *mapload.PartyMember)`.
  The real signature is `shopSlot1Code(i int, worn *[sim.EquipSlots]uint16)` (`shoproom.go:287`).
  Corrected; the identifier check this document's seventh-pass section ran compares names, not
  signatures, which is why it passed a stale one.
- `currentEquipment`'s own doc (`world.go`) claimed exactly FOUR narrow readers and named three
  explicitly (`openMission`/`switchInventorySubject`'s seeds, `rearm`, `currentWeaponFallbackActive`).
  This story added a fifth, `headlessInventory`'s `Equipment` field (`headlesspointer.go:333`,
  `:274` at an earlier pass),
  reporting the raw array deliberately beside the same state's `Figure` field, which reads
  `currentFigureEquipment`. The read is correct and intended; the count was not. Corrected to FIVE
  with the fifth reader named.
- `DIV-111` (`docs/DIVERGENCES.md`) said both `Rearm` and `PartyLoadout` resolve a bare loadout for a
  latched member "in the mission he took it off in and in every later one," which was not true of
  `recomputeRaisedSkills` before this pass's fix. Corrected to name all three production readers and
  the ninth-pass defect.

### `TestWeaponMaterializedHasOneWriter` widened, two blind spots closed

The review demonstrated, rather than argued, that the scan test was narrower than this document
claimed. Adding

```go
func probeEvadeScan() mapload.PartyMember { return mapload.PartyMember{WeaponMaterialized: true} }
```

to `pkg/game/weaponlatch.go` left `TestWeaponMaterializedHasOneWriter` GREEN. Measured directly
(`go test ./pkg/game/... -run TestWeaponMaterializedHasOneWriter -v`), confirmed, then reverted. Two
causes, both closed in `weaponlatch_scan_test.go`:

1. The walk matched only `*ast.SelectorExpr`. A composite-literal key —
   `mapload.PartyMember{WeaponMaterialized: true}` — reads and writes the field through an
   `*ast.KeyValueExpr` the walk never visited. A new `case *ast.CompositeLit` inspects composite-
   literal keys and enforces the same one-writer rule an `*ast.AssignStmt` already got: with the probe
   restored, the test now reports `game.probeEvadeScan SETS PartyMember.WeaponMaterialized in a
   composite literal at weaponlatch.go:54:72` and fails.
2. `weaponLatchDirs` scanned `pkg/game` and `pkg/mapload` alone, so a reader in any of the eleven
   `cmd/*` directories that import `pkg/mapload` in non-test code —
   `grep -rl 'againrom/pkg/mapload"' --include=*.go . | grep -v _test.go | xargs -n1 dirname | sort -u`
   — was invisible to it. All eleven are now scanned, keyed by their own directory base name rather
   than the shared package name `main` (every one of the eleven is `package main`, and `main.main`
   across eleven directories would collide under the prior `"pkg.Func"` naming).

Neither blind spot found a live site: no production `cmd/*` directory currently references the
field, and the only composite-literal writers in the tree were already in `_test.go` files this
test excludes (`inventory_test.go:770`, `loadout_test.go:37`, `party_test.go:65`, all fixtures
constructing a latched member, not production code).

### Mutation witness, ninth pass

| # | Line reverted | Result |
|---|---|---|
| M9 | `recomputeRaisedSkills`'s `everEquipped` line reverted from `mw.resolveWeaponMaterialized(mw.missionPartyMember(c.id), eq)` to the subject-scoped compare (`rearm.go`) | RED `TestRecomputeRaisedSkillsReadsTheLiveLatchNotTheMissionOpenSnapshot` — `DamageBase = 10, want 0`, reproducing the counterexample's own control values exactly (companion `DamageBase` 10 sworded, 0 bare) |

Both directions run directly
(`go test ./pkg/game/... -run TestRecomputeRaisedSkillsReadsTheLiveLatchNotTheMissionOpenSnapshot -v`):
red on the reverted line, green on the restored one.

`TestRecomputeRaisedSkillsReadsTheLiveLatchNotTheMissionOpenSnapshot`
(`pkg/game/weaponlatch_test.go`) drives every step through production entry points: `openMission`
with a two-member party, `switchInventorySubject` to make the ally the subject, `enqueueUnequip` to
tap the drawn fallback off (`materializeFallbackWeapon`, not a direct field write),
`switchInventorySubject` back to the hero, and one real `sim.KindAttack` tick that crosses the
ally's own Blade skill from level 0 to 1 (FR-3's first-award rise) while he is not the subject.
`recomputeRaisedSkills` is never called directly.

### Reconciliation, ninth pass

No new `docs/DIVERGENCES.md` row. `DIV-111`'s "Implemented behaviour" cell is corrected in place —
the same rule the seventh- and eighth-pass sections above followed — because the row's type
(`DEVIATION`) and status (`OPEN`) are unchanged; only the sentence naming which readers honour the
latch was wrong and is rewritten to name all three, including the one this pass fixed.

### Rule added: `implementation/AGENTS.md`

The owner's brief for this pass asked for one new rule, beside the existing one that a story
changing how something is DRAWN owes an enumeration of every path that PRODUCES it: a story that
introduces a persisted record of history owes an enumeration of every site that previously
re-derived the same fact from present state. Added to `implementation/AGENTS.md`'s golden-rules
register, carrying this story's own evidence — three separate adversarial passes (fifth, eighth,
ninth) each found one more site holding the pre-latch premise in its own comment, true when written
and falsified by this story: "nothing has looked at this member's equipment before"
(`buildInventorySubject`, fifth pass), "mission construction is always this member's first
resolution" (`PartyLoadout`, eighth pass), and "an entity that is not the subject cannot have
unequipped" (`recomputeRaisedSkills`, ninth pass, this section).

## Round-2 adversarial review, tenth pass

The pushed ninth-pass sha (`a4e70d7`) was reviewed and returned FAIL with five counterexamples and
one unproven observation, all aimed at the drag machinery rather than at the weapon latch this
document's fifth through ninth passes covered.

- **Counterexample 1 — the doll tremor guard is inert in production, on both the mission doll and
  the shop doll.** `command.go`'s own `originSame` doll case (Player input row, second and third
  passes above) and the shop's own `dest == origin` compare (Player input row, fifth line) both read
  the SUPPRESSED mask at release time — `dollFigureSlotAt` for the mission doll,
  `shopGridControlAt`/`shopDollSlotAt` reading `ShopScreenView.SlotMask` for the shop doll. Once a
  doll drag arms, `refreshDollDrag`/`refreshShopDrag` (`pkg/game/world.go`,
  `pkg/game/shopview.go`) push a suppressed mask with the origin slot's own code cleared, on every
  frame the drag stays armed including the release frame. The origin-identity compare therefore
  asked the origin's own mask pixel a question the suppression had already made unanswerable for as
  long as the drag stayed armed: a release back on the exact slot a press began on could never again
  read as that slot, and the tremor fell through to a genuine drag/drop instead of the tap outcome
  FR-4/FR-9 promise. Two prior passes (round 2's second and third, above) built and mutation-tested
  the tremor guard itself; neither pass's own fixture pushed the suppression a live drag installs, so
  neither test could have caught this. Fixed by asking the origin-identity question against a mask
  the suppression never substitutes: `Viewer.dollFigureSlotAtUnsuppressed`
  (`pkg/ui/inventory.go:1313`) reads `v.invSubject.SlotMask` directly, and `command.go`'s `originSame`
  doll case calls it in place of `dollFigureSlotAt` (`command.go:862`). `ShopScreenView` gained a
  second field, `OrdinaryDollMask` (`pkg/ui/shopscreen.go:307`), populated by `ShopScreen()`
  (`pkg/game/shopview.go:139`) from the same source `SlotMask` reads before the suppression
  substitution overwrites it; a new function, `shopReleaseIsOrigin`
  (`pkg/ui/shopscreen.go:540`), answers the origin-identity question against `OrdinaryDollMask` for
  a doll origin and falls back to the ordinary `shopGridControlAt` compare for the other three
  families; `app.go`'s release handling (`app.go:1696`) calls it ahead of the cross-family
  `shopGridControlAt` read. Neither the drawn picture nor `dollFigureSlotAt`'s own behaviour for
  every OTHER caller changed: the fix is a second, narrower question, not a widened hit test.
- **Counterexample 2 — the two shipped tests never pushed the production suppression state.**
  `TestDollTapStillUnequipsAfterATremorWithinTheSlot` (`pkg/ui/doll_test.go`) and
  `TestTheApplicationATremorReturningToTheDollIsATapNotADrag` (`pkg/ui/shopdrag_test.go`) both drove
  a press, an in-slot move past `TapSlop`, and a release on the origin, and both asserted the tap
  outcome — but neither fixture ever called `SetDollSuppressedFigure`/`ShopSuppressDoll` in a way
  that moved the mask the compare under test actually reads, so both tests passed against BOTH the
  broken and the fixed code for the doll's own suppressed-mask question; their prior mutation kills
  (reverting the `originSame`/`dest == origin` mechanism entirely) proved the ASSERTION was
  load-bearing, not that the FIXTURE reproduced the state a live drag installs. Fixed: the mission
  test now calls `SetDollSuppressedFigure(sub.ID, 1, suppressedFigurePic, suppressedFigureMask)`
  after confirming the drag armed, with a setup assertion that `dollFigureSlotAt` answers no slot at
  the origin once the push lands. `fakeShopTown` (`pkg/ui/town_test.go`) gained a `suppressSlot`
  field; `ShopSuppressDoll` now records it and `ShopScreen()` builds a suppressed COPY of the
  ordinary mask (the recorded slot's own pixels zeroed) whenever it is set, in place of recording the
  call and never reading it back. Both fixtures' own mutation kills are restated below, now against
  the fix that only a state-pushing fixture can distinguish from the broken code.
  Also under this counterexample: no scenario point form (`doll_slot`, `pack_code`, …,
  `pkg/game/headlesspointer.go`) carried a pixel offset, so no scenario JSON step could express a
  tremor — a move that stays inside the same logical slot while still crossing `TapSlop`.
  `HeadlessPoint` gained `OffsetX`/`OffsetY` (JSON `offset_x`/`offset_y`, `omitempty`), added to the
  resolved pixel by the new `resolve()` wrapper around the renamed `resolveBase()`.
  `scenarios/1005-doll-and-shop.json` (version 4, 87 steps) gained three tremor sequences. The
  mission doll's "hero" member and the mission doll's "join:51" member each gain one `move` step
  (`offset_x: 3, offset_y: 3`) inserted into the file's own PRE-EXISTING tap-unequip press/release
  pair, turning it into press/move/release; the pair's own following assert (equipment slot 1 now
  empty, the item carried) is unchanged and now covers the tremor case in place, since the tap
  outcome the guard fixes is the same outcome that assert already checked. An in-place insertion
  was chosen over a separate press/move/release/assert/re-equip cycle because a first attempt at
  the latter roughly doubled the elapsed simulation time inside the hero's own section and drifted
  `join:51` (entity 43) outside the mission's static camera viewport by the time a later step
  reads `select_member 'join:51'` — `pkg/sim` is deterministic, so added ticks deterministically
  move entity state, and the fix is to add negligible ticks rather than to chase the camera. The
  shop doll gains its own three-step press/move(16,16)/release sequence (steps 20-22) asserting the
  item comes off (worn empty, carried), then a separate two-step drag back onto the doll (steps
  24-26, below) to restore state for the file's own later steps. The move offset was widened from
  (3,3) to (16,16), and `HeadlessShopPoint`'s doll-surface resolution changed to read
  `OrdinaryDollMask` rather than the possibly-suppressed `SlotMask`, at the eleventh pass — see the
  counterexample C1 entry below.
- **Counterexample 3 — this document's own Player input row and its closing line
  ("no known in-scope GAP") were wrong at the pushed sha.** The row read PASS and named the tremor
  guard as an already-closed hole (second and third passes, above) without the evidence that it
  never fired in production; corrected below, in the matrix itself, to name this pass's own defect
  and fix rather than narrate it as a separate finding here — the same rule the eighth-pass section
  followed for `DIV-111`.
- **Counterexample 4 — `spec.md` was not canonicalized to the as-built state.** Corrected in place:
  the `originSame`/`dest == origin` description no longer claims either resolves "regardless of
  whether the tremor armed the drag" for the doll case, the shop section no longer claims
  `shopDollSlotAt` "already names the exact slot released on," and the `app.go:1677` citation —
  stale even before this pass, since the block had already moved — is replaced with the function
  name (`shopReleaseIsOrigin`), which does not drift the way a line number does.
- **Counterexample 5 — a dead duplicate case in `command.go`.** A second `case v.dragCandKind ==
  dragFromDoll:` inside the release switch, below the drag-release case, re-tested a condition
  `originSame` already resolves ahead of it in the same switch and carried the comment "THE PLAIN TAP
  (1005 spec, 'press to take off')" on code that could not itself be reached with a different answer
  from `originSame`'s own case. Removed; replaced with a comment recording why (`command.go:979`).
- **Unproven observation, confirmed and fixed — a release over the HUD toggle strip, the spellbook
  strip, the minimap or the side panel during an armed drag bypassed those surfaces' own gate and
  reached the ground-drop arm.** `command.go`'s two drag-release arms gated the drop on
  `!v.inventoryCaptures(...)` alone, which measures only the doll box, the worn box and the pack bar;
  the other four surfaces are checked only at PRESS, ahead of the block that arms a drag, never again
  at RELEASE. Measured directly with a scratch fixture (press a pack cell, drag past `TapSlop`,
  release over the spellbook strip's own geometry) before the fix:
  `TakeInventoryDrop() = (worn=false idx=1 x=1 y=25 ok=true)` — a live ground-drop request over a
  screen surface the HUD owns; the same fixture over the minimap answered
  `(worn=false idx=1 x=34 y=5 ok=true)`. Fixed with `Viewer.groundSurfaceCaptures`
  (`pkg/ui/inventory.go:1023`) — `inventoryCaptures || hudToggleCaptures || spellbookCaptures ||
  minimapCaptures || panelCaptures` — used at both release arms (`command.go:895`, `:945`) and by
  `HeadlessGroundPoint` (`pkg/ui/headlesspointer.go`), which asked the narrower three-surface
  question before this pass. `TestGroundDropRefusesTheSpellbookStripDuringAnArmedDrag` and
  `TestGroundDropRefusesTheMinimapDuringAnArmedDrag` (`pkg/ui/doll_test.go`) each reproduce the
  measured case and assert `ok=false`. The panel and HUD-toggle strip share the same combined gate
  but carry no dedicated test: neither is reachable by a drag release in the shipped campaign
  missions this build's scenarios drive, so their coverage rests on one function being used
  everywhere rather than on four independently-tested ones.
- **Additional finding, found while building the required tremor scenario — the shop had no way to
  drag an item back onto an unworn doll slot.** The shop's cross-family drag destination
  (`shopGridControlAt`/`shopDollSlotAt`, `pkg/ui/shopscreen.go`) answers a doll release only by
  looking up the released pixel in `ShopScreenView.SlotMask`, a mask that marks a pixel only where
  a WORN item's own layer is drawn; once the tremor-scenario steps unequip shop slot 1, no pixel in
  the composed figure carries slot 1's code, so no release point resolves to it. The mission side
  has no equivalent gap: `dollBox()` names the equip destination by membership in a fixed screen
  box, independent of whether anything is currently drawn there. `ShopDrag`'s own doll-destination
  cases (`pkg/game/shopview.go`) match on `to.Kind == ShopControlDoll` alone and never consult
  `to.Index`, so a mask-independent doll-area answer is sufficient without naming a slot. Fixed by
  extracting the existing geometric refusal logic (name plate, pickers, character-block border) out
  of `shopDollSlotAt` into `shopDollAreaAt(p image.Point) bool` (`pkg/ui/shopscreen.go`), and adding
  a fallback branch in `app.go`'s release handling: when the origin is a non-doll equip-capable
  surface and the release lands inside `shopDollAreaAt` but `shopGridControlAt` answered `!ok`
  (no marked pixel under the release), the release resolves to `ShopControl{Kind: ShopControlDoll}`.
  `HeadlessShopPoint` (`pkg/ui/headlesspointer.go`) and the `Shop.Surface` validation
  (`pkg/game/headlesspointer.go`) both gain a seventh surface, `doll_box`, mirroring the mission
  side's pre-existing `doll_box` point form: it resolves to `shopFigureRect`'s own center, bypassing
  the mask scan. `scenarios/1005-doll-and-shop.json` uses `doll_box` for the shop's re-equip
  destination (steps 24-25); `doll` by slot index remains valid and unchanged for a release onto an
  already-worn slot. This is a genuine production gap, not a scenario-construction artifact: before
  this fix, a player who fully undressed a shop character via drag could not re-dress it by
  dragging from the pack, only by tapping.

### Mutation witness, tenth pass

| # | Line reverted | Result |
|---|---|---|
| M10a | `command.go`'s doll `originSame` case, `dollFigureSlotAtUnsuppressed` reverted to `dollFigureSlotAt` | RED `TestDollTapStillUnequipsAfterATremorWithinTheSlot` — `TakeInventoryDollUnequip = (0,false), want (0,true)` |
| M10b | `shopReleaseIsOrigin`'s doll-origin read, `v.OrdinaryDollMask` reverted to `v.SlotMask` (`shopscreen.go`) | RED `TestTheApplicationATremorReturningToTheDollIsATapNotADrag` — `clicked = [], want [3]` |
| M10c | `command.go`'s two release-arm gates, `!v.groundSurfaceCaptures(...)` reverted to `!v.inventoryCaptures(...)` | RED `TestGroundDropRefusesTheSpellbookStripDuringAnArmedDrag` (`TakeInventoryDrop = (worn=false idx=1 x=1 y=25 ok=true)`) and `TestGroundDropRefusesTheMinimapDuringAnArmedDrag` (`x=34 y=5`) |
| M10d | `app.go`'s `shopDollAreaAt` fallback condition reverted to `false && ...` | RED `TestTheApplicationDragsFromPackToAnUnwornDollArea` — `dragged = [], want [[{Kind:6 Index:1} {Kind:11 Index:0}]]` |

All four run directly (`go test ./pkg/ui/... -run <name> -v`): red on the reverted line, green on
the restored one. M10a and M10b are witnessed only because the fixture fix (counterexample 2) makes
the mask the compare reads actually move; against the pre-fix fixtures, M10a and M10b would have
stayed GREEN on the reverted line, since the fixture never installed the suppressed state the
reverted line would have needed to fail against.

### Reconciliation, tenth pass

No new `docs/DIVERGENCES.md` row. `DIV-085`'s own stated property, "a hit test and a drawn pixel can
never disagree," is unchanged and unaffected: `dollFigureSlotAt`'s behaviour for every existing
caller (the drawn-picture question) is untouched, and the fix adds a second, different question
(origin identity) answered by a function that reads a different mask on purpose. The `doll_box`
fallback likewise adds no new drawn-pixel question: it answers only "is the release inside the
doll's own drawn area," geometric and mask-independent, in the same style as the mission side's
pre-existing `dollBox()` membership test, and it only ever fires where the mask read already
answered `!ok`. Both are defects in this story's own drag machinery, not a mismatch between the
implementation and researched ROM1 behaviour, so no divergence type applies.

### Rule added: `implementation/AGENTS.md`, tenth pass

A second new rule, beside the ninth pass's persisted-record rule: a mutation kill proves the
ASSERTION under test is load-bearing, but proves nothing about whether the FIXTURE that feeds it
reproduces the state production installs. Both M10a and M10b's own tests killed their mutation
before this pass's fixture fix, and killed it again after — the two runs prove different things.
Before the fix, the kill only shows that removing the `originSame`/`dest == origin` mechanism
entirely reddens the test; a fixture that never moves the mask the fixed code reads cannot
distinguish the fixed code from the broken code, so a green test proves nothing about which one is
running. Added to `implementation/AGENTS.md`'s golden-rules register, carrying this pass's own
evidence.

## Round-2 adversarial review, eleventh pass

The review's own verdict on the tenth pass's landing was FAIL on three counterexamples, all in
witness and document, not in production behaviour: the tremor guards, the ground-drop gate, the
seventh-pass fix, the shop economy, hover naming, and campaign/session persistence were confirmed
correct.

### Task 0 — master merge

Master (`6a0c398`, story `1008-save-safety`) merged into this branch as `4e0cc33`. Four files
touched by both branches: `docs/DIVERGENCES.md` (conflict, resolved by hand — both sides' rows kept,
row count 84 (ours) + 83 (master) - 77 (common ancestor) = 90 (merged), verified against no loss and
no duplication), `pkg/game/resume.go` (auto-merged, no conflict), `pkg/game/headless.go` and
`pkg/ui/headless.go` (auto-merged, no conflict).

`resume.go` carries both features without interaction: `cloneCandidateUnits` (1008, clones the
class/body cache for the prepared-load transactional path) reads `f.Units`; `liveDriver` (this
branch's seventh pass, aliases the mission's own party slice via `mapload.NameParty` rather than
`mapload.OwnParty`'s deep copy) reads `ms.Party`. The two functions never read or write the same
variable.

`pkg/game/save_test.go`'s `TestReleasedEnvelopeOneSimulationFormFiftyThreeFixture` broke after the
merge: the committed golden fixture (`releasedSaveFixtureBase64`) was encoded before this branch
added `PartyMember.WeaponMaterialized`, and `gob`'s self-describing wire format sends a type
descriptor once per stream, so a struct field addition changes the byte-exact envelope even where
the field is zero-valued on every record. Regenerated the fixture from the current encoder;
`pkg/sim`'s own hash (`0x084d1cd674bfe3cd`), simulation form 53, tick and purse assertions in the
same test are unchanged, confirming this is an envelope-level change with no `pkg/sim` binary-form
version bump — `pkg/sim/binary_test.go`'s version-named test needed no rename.

`ru`-root scenario step 87 (the mid-mission save latch witness, R1) passes after the merge (below).
The merge picks up `App.HeadlessGameMenuAction` (master `4d44dd2`), which fixes CP866 SAVE/LOAD row
selection on the `ru` root — this closes counterexample C3 by itself; nothing in this pass touches
that path directly.

### C1 — the shop tremor guard had no real scenario witness

**Observed.** Scenario steps 20-22 (claimed witness) never armed a drag: the move offset (3,3
window pixels) never crossed `TapSlop` (4) in frame-pixel space after `WindowToFrame`'s 2.5x
reduction at 1600x1200. Widening the offset alone failed differently:
`HeadlessShopPoint`'s own doll resolution read `shopGridControlAt`, which answers off
`ShopScreenView.SlotMask` — the same mask a live armed drag suppresses at the origin slot, so a
scenario step naming "doll slot 1" while a drag on that slot was armed got back no pixel for the
very slot it was trying to release on.

**Fix.** `pkg/ui/headlesspointer.go`'s `HeadlessShopPoint` resolves a `"doll"` surface against
`ShopScreenView.OrdinaryDollMask` directly, exactly as `shopReleaseIsOrigin`
(`pkg/ui/shopscreen.go:559`) does, rather than through `shopGridControlAt`. `scenarios/1005-doll-and-shop.json`
step 21's move offset widened from `(3,3)` to `(16,16)`, which is 6.4 frame pixels after
`WindowToFrame` — past `TapSlop` (4) with margin, and still inside the doll slot's own drawn area,
so the gesture stays a same-slot tremor and not a drag to a different control.

**Mutation proof, at the SCENARIO level.** `shopReleaseIsOrigin` reverted to read `v.SlotMask`
instead of `v.OrdinaryDollMask`:

```
againrom: headless: step 23 (assert_shop): the shop doll slot 1 = 265, want empty
```

exit 1. Restored, the full 87-step scenario passes, exit 0. The scenario itself, not only a unit
test, now catches a reversion of the production fix.

### C2 — two false or stale witness claims, corrected

`spec.md`'s scenario-redesign paragraph claimed the shop doll's three-step sequence "asserts the
item is still worn"; the item comes OFF (step 23: `worn` empty, `carries` holds it, `doll` empty).
Corrected, with the offset and mask fix above explained in place.

`closure.md`'s **Integration witness** section (above) was stale after the tenth pass's own in-place
step insertions: the step count (77, now 87), every step citation in the five numbered
demonstrations, the mutation-output block's own step numbers, the hero-leg/companion-leg ranges, and
the mutation table's `S2` row. Corrected against a fresh read of `scenarios/1005-doll-and-shop.json`
(87 steps) and a fresh re-run of both R1 and R2's own mutations, verbatim output above and in
"Mutation witness, eleventh pass" below. The substance of every claim survived renumbering; only the
numbers were stale, except the one sentence corrected in `spec.md`.

### TapSlop means a different real distance on the two screens, judged and left open

`TapSlop` (`pkg/ui/command.go:16`) is a single 4-unit constant read by two independent drag
accumulators. The mission map (`pkg/ui/viewer.go`, `command.go`) accumulates raw window pixels, per
the constant's own doc comment ("screen pixels," `0028 FR-1`, "four pixels is an eighth of a cell at
native zoom"). The shop screen (`pkg/ui/app.go:1608-1621`) accumulates frame pixels, taken after
`Placement.WindowToFrame`'s own scale division — 2.5 at a 1600x1200 window over the shop's 640x480
frame (`pkg/render/frame/frame.go`). The shop is 2.5x more forgiving of a hand tremor than the map
is, at that window size.

**Judgement: recorded, not fixed, and not a `DIVERGENCES.md` row.** No research claim states a real
pixel distance for either screen's own drag tolerance in ROM1 — the mission's own derivation ("an
eighth of a cell") is native to this implementation's own drag machinery, not a decoded fact, so
there is nothing to compare against and no ROM1 mismatch to record. A code fix would mean choosing
one pixel space for both screens, which changes felt drag behaviour on whichever screen moves, and
no brief through eleven passes has named which one should. Documented at both accumulators
(`command.go`'s `TapSlop` doc, `app.go:1608-1613`), cross-referencing each other and this section, so
a future pass inherits the fact rather than rediscovering it.

### Two items recorded for the record, neither fixed

**The F1 debug readout sits outside `groundSurfaceCaptures`** (`pkg/ui/inventory.go:1023`). The
function's own five disjuncts (`inventoryCaptures`, `hudToggleCaptures`, `spellbookCaptures`,
`minimapCaptures`, `panelCaptures`) are every ordinary HUD surface a ground drop must refuse; the F1
readout is not among them. It is off by default (`app.go`'s `Readout` field, raised only on F1) and
this story adds no drag path across it, so there is no shipped scenario to check a fix against.
Recorded at the function's own doc comment.

**`shopClear` (`pkg/game/shoproom.go:60`) hands every table place to the CURRENT member's pack**,
regardless of which member's press put it there or whether it was a merchant's own item — unlike
`shopOffTable` beside it, which reads a place's `Mine` bit and routes a merchant's item back to the
shelf. Pre-existing, untouched by this branch. Recorded at the function's own doc comment; not fixed
because no claim or owner directive names which member (or the shelf) a whole-table clear should
return each place to.

### Reconciliation, eleventh pass

No new `docs/DIVERGENCES.md` row. C1 and C2 are witness and document defects in this branch's own
scenario tooling and documentation, not a mismatch between the implementation and researched ROM1
behaviour. The `TapSlop` asymmetry, the F1 readout gap and the `shopClear` quirk are all internal to
this implementation's own code, with no research claim on either side to diverge from — none
qualifies for a typed row.

### Mutation witness, eleventh pass

| # | Line reverted | Result |
|---|---|---|
| M11 (C1) | `shopReleaseIsOrigin`'s doll-origin read, `v.OrdinaryDollMask` reverted to `v.SlotMask` (`shopscreen.go:559`) | Scenario exit 1, step 23: `the shop doll slot 1 = 265, want empty` |
| S1 re-run | M2 (`shopUnequipDoll`'s latch raised on the `viaFallback` arm only) | Scenario exit 1, step 23: `the shop doll slot 1 = 265, want empty` (same message as before the eleventh pass; step number unchanged) |
| S2 re-run | M1 (`liveDriver` reverted to `mapload.OwnParty(party)`) | Scenario exit 1, step 87 (step 77 before this pass's own insertions): `inventory weapon_fallback = true, want false` |

All three re-run directly against the current tree (`AGAINROM_ASSETS=<en root> go run ./cmd/againrom
-headless scenarios/1005-doll-and-shop.json`); the fixed tree passes all 87 steps, exit 0, on both
the `en` and `ru` roots.

## Round-2 adversarial review, twelfth pass

The pushed eleventh-pass sha (`8e17624`) was reviewed and returned FAIL on one blocking
counterexample (C1) with two secondary corrections (C1b, C2) and three documentary corrections
(C3). Master had not moved past `6a0c398` (this branch's own eleventh-pass merge base) between the
eleventh and twelfth passes; no further merge was needed.

### C1 — the mission doll composed from `member.Worn`, not `Carry.Equipped`

**Counterexample.** `buildInventorySubject` (`pkg/game/inventory.go`) composed the doll's figure and
`SlotMask` from `member.Worn` directly — the array as read at the party's last assembly, frozen from
that point on. Every other equipment reader in the tree (`shopEquipFromShelf` and its siblings in
`pkg/game/shoproom.go`, the town screen, `mapload.EquipmentFromParty`'s own callers) prefers
`member.Carry.Equipped` over `member.Worn` whenever a `Carry` exists — set at a prior mission's
finish (`CarryParty`/`CarryRoster`) or at an original-save restore. A member who finished a mission
wearing something different from what he was last assembled with opened the next mission's doll
showing the STALE loadout, disagreeing with the shop and the town screen for the same member.

**First correction.** Extracted `missionDollEquipment(member mapload.PartyMember) (eq data.Equipment,
fallback bool)`, routing the composition through the existing reader,
`mapload.EquipmentFromParty(member)`, rather than adding a fifth inline copy of the Carry-preference
rule, per the brief's own constraint. `buildInventorySubject` calls it in place of the bare array
read; `world.go`'s `openMission` and `switchInventorySubject` call it too, seeding a new tracker,
`mw.invComposedEquipment`, for C1b (below). `scenarios/1005-doll-carry-over-worn.json` (new, 32
steps) witnesses the cross-mission case directly: it unequips armour in mission 20, confirms the
pack over the mission-end notices and the town screen, re-enters a different mission (30), and
asserts the doll opens with the item still off the figure and still in the pack.

**Two pre-existing inline copies of the same rule, found while looking for a fifth, collapsed onto
the same reader.** `townscreen.go`'s `composeShopFaces` and `shopview.go`'s `refreshShopDrag` each
restated the identical two-line `if member.Carry != nil { equipmentFromSlots(member.Carry.Equipped)
} else { equipmentFromSlots(member.Worn) }` idiom inline, one of the four pre-existing copies the
brief's own "no fifth copy" instruction was written against. Both now call
`mapload.EquipmentFromParty(member)` directly (`pkg/game/townscreen.go:289`,
`pkg/game/shopview.go:330`); neither reads `w.Equipped` or takes a world/entity-id argument, since
the shop screen keeps no `sim.World` at all (Inventory/equipment row, above) and has no live state to
prefer over the party record. Pre-existing behaviour, unchanged: both call sites already implemented
the Carry-over-Worn preference correctly before this pass, so this is a deduplication, not a defect
fix.

**The first correction regressed a pre-existing, unrelated scenario.** Running the OFFICIAL
`bash pipeline/check-scenarios.sh` gate — not only the new scenario — against the first correction
failed `scenarios/1005-doll-and-shop.json` (87 steps, pre-existing) at step 87:

```
againrom: headless: step 87 (assert_inventory): figure slot 1 = 265, want empty
```

Step 87 saves and reloads mid-mission (the seventh pass's own R1 witness for `WeaponMaterialized`)
and asserts the doll shows slot 1 retired. **Root cause.** `member.Carry` is nil for the WHOLE of
the mission that is still assembling it — `CarryParty`/`CarryRoster` only set it at that mission's
own FINISH — so `EquipmentFromParty` itself falls back to `member.Worn` for any equipment change made
through the doll and then saved mid-mission and reloaded. The resumed world (`resumeWorld`,
`pkg/game/resume.go:373`, called before `openMission` at `frontend.go:1230`/`1286`) already carries
the correct, live-current equipment by the time any doll composition runs; the party record beside
it does not.

**Second correction, within the same pass.** `missionDollEquipment` widened to
`func missionDollEquipment(w *sim.World, id sim.EntityID, member mapload.PartyMember) (eq
data.Equipment, fallback bool)` (`pkg/game/inventory.go:193`): it now prefers a live read off the
world, `w.Equipped(id)` (through a nil-tolerant wrapper, `equippedIfAny`,
`pkg/game/inventory.go:213`), falling back to `mapload.EquipmentFromParty(member)` only when the
world holds no entity for `id` yet. This agrees with `mapload/start.go`'s own mission-open mint
(lines 873-891), which resolves the identical `p.Carry`-then-`p.Worn` preference into the world's OWN
entity at TRUE fresh open — so a live-world read matches the Carry-preferred answer at open and
additionally stays correct across a resume or a live in-mission change, which a pure party-record
read cannot. `buildInventorySubject`, `openMission`'s `invComposedEquipment` seed
(`pkg/game/world.go:1123`) and `switchInventorySubject`'s own seed (`pkg/game/world.go:1862`, which
now also passes `World: mw.world` in its synthetic one-member `Mission`) all call the same function
with the same three arguments.

**Mutation proof, both corrections, at the SCENARIO level.**

| # | Line reverted | Scenario | Result |
|---|---|---|---|
| M12a | `missionDollEquipment` reverted to `eq = equipmentFromSlots(member.Worn)` entirely (dropping the Carry/live preference altogether) | `scenarios/1005-doll-carry-over-worn.json` | `step 32 (assert_inventory): figure slot 7 = 46863, want empty` |
| M12b | `equippedIfAny` forced to always return `([sim.EquipSlots]uint16{}, false)` (forcing every call to fall back to `mapload.EquipmentFromParty`, reproducing the first correction alone) | `scenarios/1005-doll-and-shop.json` | `step 87 (assert_inventory): figure slot 1 = 265, want empty` |

Each mutation was also run against the OTHER scenario and left it passing (exit 0): M12a does not
fail `1005-doll-and-shop.json`, and M12b does not fail `1005-doll-carry-over-worn.json`. The two
scenarios are complementary, non-overlapping witnesses for the two corrections — the cross-mission
Carry preference and the same-mission live-world preference — neither alone exercises what the other
catches. Both mutations were restored and the full 87-step and 32-step scenarios re-confirmed
passing, exit 0, on both roots (below).

**The reverse direction — `CarryRoster`'s mid-mission joiner — investigated and closed, reachable in
shipped play.** `CarryRoster` (`pkg/mapload/carry.go:200`) is called at a mission's own finish to
build the NEXT mission's party record; its own joiner branch (`w.BoundarySurvivors`, folded in from a
unit a script handed the player, line 211 on) sets a joining member's `Carry.Equipped` from a live
world read (`worn(w, id)`, line 156-159) while `Worn` is left at whatever `roster[id]` (the map's own
`Start.Roster` template) supplied — the identical divergence in the opposite direction: `Carry` is
live-correct, `Worn` is not. Because the joiner's `PartyMember.Carry` is already non-nil by the time
`missionDollEquipment` runs at the NEXT mission's own open, and that open is a TRUE fresh mint (not a
resume), `mapload/start.go`'s own mint seeds the world's entity from `p.Carry` first — so both the
live-world read and `EquipmentFromParty`'s own Carry preference agree, and the joiner's doll opens
correctly under either correction. This path is reachable in shipped play: `scenarios/0159-mission40-join.json`
(pre-existing) exercises a scripted mid-mission join, the same mechanism `CarryRoster`'s own doc
comment cites (`0159 FR-3, FR-4`).

**The `world.go` staleness guard — measured, not reasoned, to need no change.**
`refreshEquipment`'s own guard (`pkg/game/world.go:2415`, `if eq == mw.invFigureEquipment && fallback
== mw.invSubject.WeaponFallback { return }`) compares two LIVE-derived values
(`currentFigureEquipment`'s own return against the tracker it last set) and has no read of
`invComposedEquipment` in its condition at all; `invComposedEquipment` is moved forward at line 2453,
inside the guarded branch, strictly after the guard has already decided to recompose. Its own value
answers the question `missionDollEquipment` asked, not the question this guard asks. Measurement
rather than reasoning: both mutation-kills above (M12a, M12b) exercise this exact codepath in a real
mission — the doll-and-shop scenario's step 87 recomposes through a resume, which runs through
`openMission`'s seed, not through this guard, and neither scenario's outcome depends on this guard's
own comparison changing. No line in the guard was touched; both scenarios pass on the unmodified
guard with the fixed `missionDollEquipment`, confirming the guard needs no change rather than
assuming it from the code's shape alone.

### C1b — `headlessInventory()`'s `Figure` field claimed a derivation it did not use

**Counterexample.** `headlessInventory()` (`pkg/game/headlesspointer.go`) set the headless `figure`
field from `mw.currentFigureEquipment()` — a live re-derivation, independently defined and gated on
`mw.invWeaponEverEquipped` — under a comment claiming this was "the same derivation the doll composes
from." It was not: `buildInventorySubject` composed the doll from its own call, and the two could
disagree (and, before C1's own fix, systematically did — the doll showed the stale `member.Worn` set
while `currentFigureEquipment` read the live, un-widened array).

**Fix, and a first wrong attempt within this same pass.** The first attempt pointed `Figure` at
`mw.currentFigureEquipment()`'s own return with `fallback` folded in by hand — still a live
re-derivation, not the composed subject, and still capable of disagreeing with what the doll actually
drew whenever the two derivations' own gating conditions differ. Corrected: `headlessInventory()`
now reads `Figure: equipmentSlots(mw.invComposedEquipment)` (`pkg/game/headlesspointer.go:332`) — the
new tracker seeded from `missionDollEquipment`'s own return at `openMission` and
`switchInventorySubject`, the SAME call `buildInventorySubject` itself makes, and moved forward
inside `refreshEquipment`'s guarded recompose branch alongside `invFigureEquipment`
(`pkg/game/world.go:2453`). `HeadlessInventoryState.Equipment` is unchanged, still reading
`mw.currentEquipment()` (the raw live array, no fallback) for a caller that wants that instead of the
composed figure.

**Which of the ten pre-existing scenarios' `assert_inventory` calls could depend on the old
(live-re-derivation) semantics of `figure`, checked.** `invComposedEquipment` and
`mw.currentFigureEquipment()`'s own return agree at every point `refreshEquipment`'s guard has ever
recomposed BOTH from (mission open, and any live change since), because `missionDollEquipment` itself
now reads the live world too (C1's second correction) — the two derivations produce the same value
in every case this tree can currently construct, differing only in the one case
`invComposedEquipment` exists to catch (a composition that started wrong and has not yet been
live-recomposed, which cannot occur once C1's own fix is in place). Confirmed by running the full
`check-scenarios.sh` gate (below): all ten pre-existing scenarios pass unchanged under the corrected
`Figure` field, and none of their `assert_inventory` calls' expected `figure` values changed.

### C2 — `mapload/start.go`'s comment overclaimed the zero value's correctness

**Counterexample.** A comment at `pkg/mapload/start.go` (on `PartyMember.WeaponMaterialized`) claimed
the field's Go zero value, `false`, is "the correct answer for every member a save this old could
have written to disk." A reviewer-found counterexample: a member whose weapon fallback had already
resolved into a real item in an EARLIER mission, and who then sold or dropped that item before a
save predating the field was written, decodes with the latch unraised — `false` is wrong for this
member, because his fallback question was already answered and answering it again re-offers a weapon
he no longer owns.

**Fix.** The comment is narrowed to state the true scope: `false` is the correct decode only for a
member who never triggered the fallback before the save was written. Recorded as a typed ledger row,
`DIV-112` (`docs/DIVERGENCES.md`), rather than left as an inline claim with no counterexample to
weigh it against — FIDELITY-DEBT, OPEN, since the field is new state with nothing in an old save to
decode it from.

### C3 — three documentary corrections

**1. `DIV-095` updated.** The row's own claim — that the committed `releasedSaveFixtureBase64`
fixture proves a save an EARLIER build actually wrote still loads under the current one — was
weakened by the eleventh pass's own fixture regeneration (forced by `WeaponMaterialized`'s effect on
gob's type descriptor): the regenerated fixture proves only that the CURRENT build reproduces its own
current output, not that it reproduces an earlier build's bytes. Updated to state this narrowing and
to name the new decode-only fixture (below) as the one that now stands for the older claim.

**2. `closure.md`'s Persistence/save-load matrix row made consistent with the eleventh-pass section.**
The row (line 19, above) now states the same narrowing `DIV-095` states, in the same terms the
eleventh-pass narrative section already used, rather than reading as though the regenerated fixture
alone still witnessed backward compatibility.

**3. A second, separately-named constant restored for the pre-1005 released save fixture.**
`preWeaponMaterializedSaveFixtureBase64` (`pkg/game/save_test.go`), spliced verbatim from `git show
master:pkg/game/save_test.go`'s own `releasedSaveFixtureBase64` (renamed, bytes unchanged) — `master`
at `6a0c398`, before `WeaponMaterialized` existed. `TestPreWeaponMaterializedSaveFixtureStillDecodes`
decodes it, checks the envelope label, the snapshot's gold/mission/offered/won/available fields, the
`pkg/sim` world hash (`0x084d1cd674bfe3cd`), and asserts `WeaponMaterialized` is `false` for every
decoded party member — DECODE alone, not a byte-exact re-encoding, since a byte-exact check would
fail the moment the type descriptor changed and prove nothing about the field this fixture exists to
witness. Both fixture tests pass; `go test -trimpath -run TestPreWeaponMaterializedSaveFixtureStillDecodes ./pkg/game/...`
confirmed green in isolation and in the full suite (below).

### Reconciliation, twelfth pass

`DIV-095` updated in place (no new row: same subject, narrower claim). `DIV-112` is a new row for C2,
FIDELITY-DEBT, OPEN. No divergence row for C1 or C1b: both are implementation defects in this story's
own compositor against this story's own (and every other reader's) Carry-preference rule, not a
mismatch against researched ROM1 behaviour — `mapload.EquipmentFromParty`'s own rule predates this
pass and this pass brings the doll into agreement with it, it does not change what the rule is.

### Gate chain, twelfth pass

`go build ./...`, `go vet ./...`, `gofmt -l $(git ls-files --cached --others --exclude-standard
'*.go')` (empty output) and `go test -trimpath -count=1 ./...` all clean, every package `ok`.
`bash scripts/check-no-game-assets.sh`: `check-no-game-assets: clean (tree scan)`.
`pipeline/check-scenarios.sh` selected 11 scenarios (the ten pre-existing plus the new
`1005-doll-carry-over-worn.json`) and passed 11 of 11 on both the `en` and the `ru` asset roots.
Master had not moved past `6a0c398`; no merge was needed this pass.

### Matrix rows corrected

Inventory/equipment, Campaign/session and Interactions with existing mechanics are corrected above
(Twelve-aspect matrix) to describe both C1 corrections, both scenario-level mutation proofs, and the
reachable reverse-direction finding, rather than only the first (incomplete) correction. No row moves
off PASS: the fix closes a defect in an already-`PASS` aspect's own supporting evidence, not a new
gap.

### Mutation witness, twelfth pass

| # | Line reverted | Result |
|---|---|---|
| M12a | `missionDollEquipment` reverted to `eq = equipmentFromSlots(member.Worn)` (`pkg/game/inventory.go`) | `scenarios/1005-doll-carry-over-worn.json` exit 1, step 32: `figure slot 7 = 46863, want empty` |
| M12b | `equippedIfAny` forced to always return `(zero, false)` (`pkg/game/inventory.go`) | `scenarios/1005-doll-and-shop.json` exit 1, step 87: `figure slot 1 = 265, want empty` |

Both re-run directly against the current tree (`AGAINROM_ASSETS=<en root> go run ./cmd/againrom
-headless <scenario>`); the fixed tree passes both scenarios, exit 0, on both the `en` and `ru`
roots. Each mutation was also confirmed NOT to fail the other scenario, establishing the two as
complementary rather than overlapping witnesses.
