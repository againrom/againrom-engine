# Story 1125 — town SAV for a native campaign

## Status

Landed, returned by the seat's adversarial pass, then corrected (one
correction pass, per process). The returning finding: the native SAV path
(`pkg/game/nativecity.go`) returned right after `store.WriteOriginal`, so a
native campaign's SAV was the only file written and no `.ags` existed
beside it, while that SAV silently zero-filled every worn item, carried
item and spellbook (old DIV-889) and the Attack/Defence/Modifier and
stat-word raw blocks (old DIV-879, DIV-880). Before this story the same
SAVE produced an `.ags` that kept all of it, so a player who bought
equipment, learned spells or trained stats and saved in town loaded back a
stripped party.

The correction makes the native SAV **lossless or unwritten**:
`ExportNativeCitySave` now refuses (`*originalCityUnsupportedError`,
`nativeCityLossyMember`) whenever any party member carries a worn item, a
carried item, or a known/present spellbook — exactly the state this
generator has no writer for — and `SaveSeams` falls back to the lossless
`.ags` envelope precisely as it did before this story. Speed, Capacity,
HealthRegen and ManaRegen (part of old DIV-880) are no longer zero-filled:
they are written from the same already-computed `data.Derived` value the
other eight `UnitStatWords` already use. Attack/Defence/Modifier and the
remaining opaque raw blocks (DIV-879) stay zero-filled, now backed by a
traced, Against-code-verified claim that no consumer reads them back for a
native party (Key findings, addition 8 below). scalars[1] is written 1
(was 0), citing research candidate EXP-0305 (unlanded); scalars[6] stays 0.

**Consequence for reach**: chargen unconditionally arms a starting weapon
and packs the campaign-documents access item (`partyInputs.Documents`,
story 1035 B5), so every chargen'd party carries at least one worn and one
carried item from the moment of creation. In practice this means the
native SAV path fires only for a party with no worn/carried items and no
known spells — a real, ordinary chargen'd hero always takes the `.ags`
fallback today. This is the honest, correctness-first outcome of "lossless
or unwritten": the roster/campaign-record architecture below is unchanged
and ready for a follow-on story that adds the missing Item-object writer
(DIV-889), which would let the SAV path reach real parties.

Three release-gated witnesses (`pkg/game/nativecity_release_test.go`, EN
and RU) now cover both shapes: two use a party stripped of items/spells to
prove the SAV path is genuinely lossless where it fires, and a third
chargens a real hero (worn equipment, carried inventory, known spells,
non-default stats) and proves `SaveSeams` refuses the SAV path and the
`.ags` file it falls back to reloads that hero's equipment, inventory,
spells and stats unchanged. Two focused asset-free tests
(`pkg/formats/sav/city_generate_test.go`) are unchanged by this
correction. Twelve divergence rows (DIV-878 through DIV-889) remain filled
in `docs/DIVERGENCES.md`; DIV-879, DIV-880, DIV-888 and DIV-889 were
rewritten by this correction.

`docs/1125/story.md`'s design sections below (architecture decision, key
findings, field mappings) remain accurate as the as-built record for the
roster/campaign-record machinery; the equipment/spellbook and stat-word
bullets were updated by this correction to match the code above.

**Second correction (seat hotfix, docs/HOTFIXES.md)**: a further adversarial
pass (`pipeline/reviews/story1125-pass1.md`) found the equipment/spellbook
gate above did not cover four more player-visible losses, all reachable
through the shipped town shop alone with no chargen edge case needed (sell
everything via `shopUnequipToTable`/`shopFromPack`/`shopSell`, no chargen'd
item survives): a hero or mercenary with no Humans row to bind reloaded
with a generic male-fighter identity in place of the one the player made
(DefRow 0; `restoredDefinition`, R-1); the tavern's permanent
mercenary-enabled set reloaded as the current chapter's own declared list
instead of the session's accumulated unlock history (DIV-886, R-2);
world-map markers, quick-spell bindings, the outstanding offered mission
and world-selection history had no document field at all and reloaded
empty (R-3); and a chapter companion the player dismissed returned, because
`RestoreOriginal`'s town arm calls `addChapterCompanions` unconditionally
(R-4). `ExportNativeCitySave` now refuses on all four in addition to the
equipment gate (`nativeCityMercenarySetMismatch`, `nativeCityDismissedCompanion`,
direct zero-value checks on the four session fields, and `nativeCityDefRow`
returning 0 for any party member); `CityFromData`/`Marshal` failures also
now degrade to the same refusal sentinel instead of failing SAVE outright
(D-3), each with one bounded `log.Printf` line.

This supersedes the "fires only for a party with no worn/carried items and
no known spells" framing above: because every valid party carries exactly
one starting hero and a hero's `MercenaryType` is always 0, the DefRow-0
refusal now fires for every party this writer is asked to save, equipped or
not. The native SAV path is unreachable in practice today, not merely
narrowed to a bare party; `SaveSeams` always falls back to the lossless
`.ags` envelope, exactly as it did before story 1125 entirely. The two
existing bare-party witnesses were narrowed and renamed to assert refusal
and `.ags` fallback instead of SAV success
(`TestReleaseNativeTownSaveBarePartyRefusesForSessionStateAndFallsBackToAGS`,
`TestReleaseNativeTownSaveRosterChangeAcrossAGSFallbackSaves`); the
equipped-party witness is unchanged
(`TestReleaseNativeTownSaveEquippedPartyFallsBackToCompleteAGS`); a new
fourth witness drives the review's own shop-baring route through chargen,
`Town.Won`, `arriveInTown`, `addChapterCompanions`, `markWorldSelected` and
the production shop-room methods, then asserts refusal and a field-by-field
`.ags` round trip of hero identity, mercenary set, `Town.won`,
world-selected history, quick spells, `Offered` and gold
(`TestReleaseNativeTownSaveShopBaredSessionRefusesAndAGSFallbackRoundTrips`).
All four pass on both EN and RU. DIV-887 and DIV-889 were amended to match;
see Open debt below.

## Player result

A campaign that never imported an original save (`f.Town.progress == nil`,
the ordinary case for a game started in Againrom) presses SAVE in town and
gets a real `Asg&` SAV file **only when that would be lossless**: no worn
item, no carried item, no known spell on any party member. A fresh process
loads that file and shows the same roster names/stats/skills (including
Speed/Capacity/regeneration), gold, journal (documents), current chapter,
available side missions and mercenary pool/hired state.

Whenever any party member carries a worn item, a carried item or a known
spell — the ordinary shape of a real, played party — SAVE instead writes
the native `.ags` envelope, exactly as it always did before this story.
That file already carries equipment, inventory and spellbooks losslessly;
a release witness
(`TestReleaseNativeTownSaveEquippedPartyFallsBackToCompleteAGS`) proves the
fallback fires and the file reloads complete.

## Architecture decision

`originalCitySaveState`/`bindOriginalCity`/`.marshal()` (originalsave.go) is
the REFUSAL-TABLE apparatus for import continuity: it binds once at SAV
import time (`captureSession`) and refuses to export whenever a native
mutation happened that it does not understand. A native (never-imported)
campaign has no baseline to diverge from, so that machinery does not apply
and is NOT touched by this story.

Instead: a new, independent path builds a complete `sav.CityData` FRESH from
current live state on every SAVE call, converts it with the EXISTING
`sav.CityFromData` (already validates budget/graph/round-trip), then calls
the EXISTING `(*CityProvenance).Marshal(update)` once for final bytes. Since
the roster and campaign record are rebuilt from `f.Carried`/`f.Town` every
time, hiring/dismissal/mercenary arithmetic (item 3) need NO special-case
code — they are automatically reflected because the source is live state,
not a cached baseline.

Wiring point: `pkg/game/resume.go` `SaveSeams()`, the `s.Mission == 0 &&
f.originalCity != nil` branch. Add a sibling branch for `f.originalCity ==
nil` that calls the new `f.ExportNativeCitySave(s, label)`; on any
`*originalCityUnsupportedError` it falls through to the existing `.ags`
path exactly like the import branch does. The existing branch is unchanged.

## Key research/code findings that make this safe

1. `actorLoadState()` (pkg/formats/sav/holdings.go) always returns
   `Present: true` for any parsed Human/Unit record — it is not a "was this
   saved mid-mission" signal. `actorBasisFields` (actorbasis.go) requires
   only correct RAW BLOCK LENGTHS (UA6=24, UBE=22, U114=24, UD4=64,
   U154=180), not any specific content.
2. `originalActorBasis()` (pkg/game/originalholdings.go) sets
   `sim.SourceActor.Class = 2` for ANY `"Human"`-class actor unconditionally
   (only `"Unit"` maps to 1) when decoding an IMPORTED original mission SAV.
   Combat-derive NUMERIC fidelity from zero-filled Attack/Defence/Modifier
   is a bounded, MISSION-ENTRY-ONLY concern (mission-side saves are out of
   scope per the brief), NOT a town-state or crash risk. Town state
   (Hero/Saved/Carry.SkillXP) is read by `RestoreParty` directly from
   Stats/SkillLevels/SkillXP/Experience, an entirely separate path from
   Attack/Defence/Modifier, so it is unaffected by zero-filling those
   blocks. Finding 8 (added by the correction pass below) traces the
   mission-entry side precisely: for a NATIVE party this Class=2 decode
   path is never reached at all, because nothing populates
   `Carry.LiveLoad`/`OriginalHuman` for a session that never imported one.
3. `restoredDefinition()` (pkg/game/originalparty.go:542) gracefully
   defaults (`Profile{}`, ManFighter, face 1, not mage) when `DefRow` is 0 or
   out of the installed Humans range — it does not error. `RotationSpeedBase`'s
   own doc block establishes "a full hero corresponds to no Humans row at
   all" as EXISTING, already-shipped chargen policy. So a generated hero
   gets `DefRow = 0` (semantically correct: this hero really has no row, not
   an approximation), and a generated mercenary gets
   `DefRow = data.FindHumanByType(f.Table.Humans, int32(member.Class))`
   (0 if not found), matching the codebase's own accepted legacy-fallback
   pattern for the same lookup elsewhere (pkg/mapload/loadout.go:162).
   Downstream, `hero.Body/.Reaction/.Mind/.Spirit` get overwritten from
   `c.Stat(...)` regardless of what the DefRow template supplied, so a
   generated hero's actual stats are unaffected by DefRow — only the
   cosmetic figure/profile/mage flags fall back to defaults (open debt).
4. City state store (`&YA1`, parsed by `parseStateStore`/`cityStateShape` in
   pkg/formats/sav/city_state.go) has an EXACT required shape: 7 directories,
   22 total records (7 dirs + 15 leaf values). `pkg/formats/sav/city_test.go`'s
   `cityTestState`/`cityTestSource`/`cityTestHuman`/`cityTestToken` already
   hand-build this shape from scratch, structurally validated by an existing
   passing test (`TestCityProvenanceStructurallyRoutesIdentityKeyedRosterAndRemints`).
   This story's generator follows that exact proven shape, parameterized by
   live values instead of test constants.
5. `sav.CityCharacter.Hero` is not a stored per-unit bit; it is derived by
   comparing a unit's own identity to the Player object's `fixed[43:47]`.
   The generator marks the hero by writing that member's own identity there.
6. Campaign record consumption was verified method-by-method in
   `pkg/game/campaignprogress.go`: `Main.Payment/ShopMin/ShopMax/AddHero/EnableMercenary`
   feed `chapter()` (real gameplay: rewards/shop bounds/heroes granted) and
   MUST be populated correctly. `AutoGetMission`, `LastMission`,
   `MissionTime` are decoded, stored and re-encoded but are NOT read by
   `record/chapter/lowerMainBlocked/selectedMarkers/advanceMain/complete/
   takeAddHeroes/selectMission/announce/take` — confirmed safe to zero-fill
   (matches the brief's own treatment of scalars[1]/[6]). `Announced`
   (per-record) is exactly native `Town`'s `t.available[mission]` (both mean
   "taken from a building, not yet won" — confirmed by reading
   `newTownFromCampaignProgress`'s seeding: `if r.announced { t.available[r.mission]
   = true }`, and `Town.Take()`'s native branch: `t.taken[ref]=true;
   t.available[m]=true`). `InnMission/TCMission/ShopMission` (still-offered
   candidates) are the chapter's own candidate lists with entries already
   consumed via `t.taken[offerRef{...}]` removed, matching what native
   `Offers()` computes live and what `campaignProgress.chapter()` returns
   directly post-import.
7. `progressRecordFromCampaign(c Campaign, mission int)` and
   `candidateSides(ch Chapter)` (both unexported, both in
   pkg/game/campaignprogress.go, same package as the new generator) are
   REUSED DIRECTLY rather than reimplemented — they are the exact functions
   `advanceMain` itself uses to build a fresh chapter's records from the
   registry.
8. Added by the correction pass. `originalHumanSpawn` (pkg/mapload/originalhuman.go)
   reads a party member's Attack/Defence/Modifier/raw-block state through
   exactly two sources: `p.Carry.LiveLoad` (when its `Inventory.Source.Class
   == 2`) or `p.OriginalHumanState()`. Neither is ever populated for a
   NATIVE party. `CarryParty` (pkg/mapload/carry.go) sets
   `out[i].OriginalHuman = nil` for every member on every mission return,
   unconditionally. `Carry.LiveLoad = e.CurrentActorLoad()` is likewise set
   generically on every mission return, but `Entity.CurrentActorLoad()`
   (pkg/sim/actorload.go) returns nil unless `e.ActorLoad.Present`, and
   nothing sets that field true except an original-SAV-derived restore
   (`originalActorLoad`, pkg/game/originalparty.go:484, called only from the
   original mission-SAV import path). So for a session that never imported
   one, `Carry.LiveLoad` stays nil and `p.OriginalHuman` stays nil forever;
   `originalHumanSpawn` therefore always returns `ok=false` and every caller
   (`PartySpawn`/`PartySpawnWithTable`) falls through to the fully generic
   recompute (`partySpawn`, driven by Body/Reaction/Mind/Spirit/skills/
   equipment) at every mission entry. The raw per-Human SAV blocks
   (DIV-879) are consulted by NOTHING, for ANY native party, regardless of
   equipment — a stronger and more precise claim than finding 2's original
   "bounded gap" framing, and the one `docs/DIVERGENCES.md`'s DIV-879 now
   cites.

## Campaign record field mapping (native generator)

- `Main` = `progressRecordFromCampaign(f.Campaign, f.Town.Chapter()).savRecord(false)`,
  `Announced: true` (being in town at all implies the main mission is known).
- `Children` = for each mission in `candidateSides(chapterData)` that is not
  `t.won[m]`: `progressRecordFromCampaign(f.Campaign, m).savRecord(true)`
  (`Age` forced to 0 — DIV row), `Announced: t.available[m]`.
- `Mercenaries` = `PermanentMercenaries` = `chapterData.Mercenaries` (chapter's
  own declared list; a native campaign's cross-chapter permanent-unlock
  history is not separately reconstructed — DIV row, `validateMercenaryTypes`
  accepts the shared list either way).
- `InnNPC/InnMission/TCMission/ShopMission` = chapter's own paired candidate
  lists with entries already consumed via `t.taken[offerRef{chapterData.Mission,
  building, index}]` filtered out, preserving pairing/order.
- `Documents` = from `t.documents` (same construction `snapshotTown`'s
  existing restored-path branch already uses).
- `MercenaryWorking/Pristine/Hired[1..15]` = `t.mercPool[i]`/`t.mercCapacity[i]`/`t.mercHired[i]`
  (same construction `snapshotTown`'s existing restored-path branch already
  uses — verbatim reuse of live Town fields).
- `SelectedMission` = `uint32(f.Town.Chapter())` (always validates: equals
  Main). `AutoGetMission = 0`, `LastMission = 0`, `FirstMapPoint = false`,
  `MissionTime = 0`, `Markers = nil` — all DIV rows, confirmed unconsumed by
  current gameplay logic (point 6 above) or already matching native's own
  default (FirstMapPoint/Markers/`selectedMarkerMissions()` already return
  false/nil for a native Town regardless of this story).
- scalars[1] and scalars[6] (city_campaign.go, `applyCityCampaignProjection`
  already never touches them; set once in `nativeCityData`'s own
  `sav.CityData` literal, before that projection ever applies) — scalars[1]
  written 1 (DIV-888, correction pass; research candidate EXP-0305,
  unlanded, claim SAV-599: 27/27 walked owner saves store 1, consumed
  through the neutral `.alm` arm of UNIT-GATE-012/AI-DIFF-016/UNIT-GATE-013),
  scalars[6] written 0 (EXP-0305 claim SAV-600: no located consumer).

## Roster/Human object mapping (native generator)

Per live `mapload.PartyMember`: identity = small sequential placeholder
(reminted by `Marshal` regardless), Name, DefRow (see finding 3 above),
Stats (all 14 `UnitStatWords` from `Hero.Body/.Reaction/.Mind/.Spirit`,
`Saved.HP/MaxHP/.../Mana/MaxMana`, and — correction pass, DIV-880 —
Speed/Capacity/HealthRegen/ManaRegen from the same `data.Derived` value the
other eight words already use, at the same `sav.Stat*` indices
`cityHumanState`/`cityHumanUpdate` use for the imported path; Weight/Load
stay 0, now correct rather than approximated because the lossless-or-refuse
gate below guarantees no equipment/inventory exists whenever this generator
actually runs), SkillLevels from `Hero.Skill[6]`, SkillXP from
`Carry.SkillXP` (0 if `Carry == nil`), Experience from total XP.
Attack/Defence/Modifier/ManaFloor/Sight/MoverSpeed and all other opaque raw
blocks (RawBE, Raw114, RawD4, Raw154, Raw158, Scalar1, ScalarTail,
Words15c/178/158) are explicit zero-fill — DIV-879, provably not lossy for
a native party per finding 8. Equipment/Container/Spellbook item refs are
always empty by construction, because `ExportNativeCitySave` now refuses
(`nativeCityLossyMember`, DIV-889, correction pass) before this generator
ever runs for a party member whose live `Worn`/`WornItems`/`Carried`/
`CarriedItems` (via `mapload.MemberItemEquipment`/`MemberCarriedItems`) or
`KnownSpells`/`SpellbookPresent` are nonempty — so the always-empty refs
this generator writes never diverge from what that party member actually
has. `Hero` flag is carried by writing that member's identity into the
Player's `fixed[43:47]`, not a per-unit field (finding 5).

## Divergence rows

All 12 numbers in the reserved range are used, none retired unused. DIV-879,
DIV-880, DIV-888 and DIV-889 were rewritten by the correction pass; the
others are unchanged:

- DIV-878: state-store leaf defaults (hero name, `-1`-filled quick-spell
  shortcuts).
- DIV-879 (correction pass): opaque per-Human raw blocks zero-filled,
  provably unread for a NATIVE party by any consumer at any mission entry
  (finding 8) — not merely bounded, as the original text said.
- DIV-880 (correction pass): Speed/Capacity/HealthRegen/ManaRegen now
  written from the same `data.Derived` value the rest of Stats already
  uses (closed as a zero-fill); Weight/Load stay 0, correct because
  DIV-889's gate guarantees no equipment/inventory whenever this generator
  runs.
- DIV-881/882/883: AutoGetMission/LastMission/MissionTime written 0, each
  confirmed unread by every method in `campaignprogress.go`.
- DIV-884: FirstMapPoint/Markers written false/nil, matching native Town's
  own existing default regardless of this story.
- DIV-885: side-mission offer Age written 0 (not separately tracked).
- DIV-886: Mercenaries and PermanentMercenaries both written from the
  chapter's own declared list (cross-chapter permanent-unlock history not
  separately reconstructed).
- DIV-887: hero DefRow 0 (correct, not an approximation, per finding 3);
  mercenary DefRow best-effort via `FindHumanByType`.
- DIV-888 (correction pass): campaign scalars[1] written 1 (research
  candidate EXP-0305, unlanded, claim SAV-599), scalars[6] written 0
  (EXP-0305 claim SAV-600); not waited on for landing.
- DIV-889 (correction pass): equipment/inventory/spellbook contents remain
  unreconstructed, but the row's meaning changed from a silent fidelity
  debt to the exact, enforced fallback condition — `ExportNativeCitySave`
  refuses whenever any of the three is present, so the generator never
  ships a document that drops them.

## Proof

- `go build ./...`, `gofmt -l .` (clean on the candidate), `go test
  -trimpath -count=1 ./...` (all packages pass, including
  `internal/gatedtests`'s checked-in population scan after adding the third
  release test name — 170 gated tests now, was 169).
- `pkg/formats/sav/city_generate_test.go`: two asset-free tests exercise
  `NewCityStateData`/`CityFromData` at the exact DTO boundary
  `nativeCityData` uses (not the lower-level path `city_test.go`'s existing
  test already covers) — one full valid round trip, one "hero identity
  matches no roster member" rejection. Both caught real defects before this
  landed: `CityFromData` requires the Player's Slot/SlotAgain fields
  populated and nonzero (playerAt, `campaign.go`), and requires
  `/SpellBook/Shortcuts` filled with `-1` words rather than 0 (0 is a live
  `originalBookIDs` index, so an all-zero Shortcuts value made every quick
  spell slot resolve to the same ID and fail `validateQuickSpells`'s
  duplicate check). Both are fixed in `nativecity.go`/`city_generate.go`.
  Unchanged by the correction pass.
- `pkg/game/nativecity_release_test.go`, run against both EN and RU lawful
  installs (three tests, all pass on both):
  `TestReleaseNativeTownSaveWritesSAVAndFreshProcessRestoresState` and
  `TestReleaseNativeTownSaveReflectsRosterChangeAcrossSaves` now use
  `nativeCityBareMember` (correction pass) to strip a chargen'd hero's
  worn/carried/spell state before exercising the SAV path — the only party
  shape it can prove lossless — then check the same structural fields as
  before (roster/money/label/SelectedMission/journal/mercenary pool via
  `sav.Open`/`.CityProvenance()`/`.Campaign()`, and a fresh-process
  `SaveSeams`/`store.WriteOriginal`/`RestoreOriginal` round trip for hero
  presence/gold/chapter) plus, for the roster-change test, that dismissing a
  party member between two `ExportNativeCitySave` calls changes only the
  roster (item 3, no special-case code needed).
  `TestReleaseNativeTownSaveEquippedPartyFallsBackToCompleteAGS` (added by
  the correction pass) is the returned story's own required witness: a real
  chargen'd hero (worn weapon and armor, one carried item, non-zero
  `KnownSpells`, non-default stats — no shopping needed, chargen already
  produces this) makes `ExportNativeCitySave` return
  `*originalCityUnsupportedError`, drives the real `SaveSeams`/`save`
  closure and confirms the published file name is NOT `IsOriginal` (took
  the `.ags` branch, not SAV), then reloads it in a fresh `FrontEnd` via the
  real `SaveSeams`/`load` closure and confirms the restored hero's `Worn`,
  `Carried` length, `KnownSpells` and `Hero` stats all equal the pre-save
  values exactly, plus gold and chapter.
- Known limitation found during the original implementation, not a defect,
  unaffected by the correction: `RestoreOriginal`'s own town-arrival path
  can grant chapter companions on a first-ever visit to a restored chapter
  (`addChapterCompanions`, gated on `Town.restoredCampaign()`) —
  pre-existing behaviour of every town restore, independent of save origin.
  The SAV-path round-trip witnesses check the saved hero's presence and
  gold/chapter, not exact post-reload roster length, for this reason; the
  new `.ags`-fallback witness is unaffected (native `.ags` restore does not
  go through `addChapterCompanions`) and does compare Worn/Carried/
  KnownSpells/Hero exactly.
- Milestone census (mission 10/20 UNSUPPORTED script-node count via
  `cmd/missionrun -trace -ticks 1`, EN root): 0/0 on the candidate,
  remeasured directly for this correction. Master (`pipeline/milestone-baseline.txt`'s
  own script-node census) is a different measurement, but the same
  reasoning holds for this comparison: the diff (`pkg/game/nativecity.go`,
  its release test, `docs/DIVERGENCES.md`, `docs/1125/story.md`,
  `internal/gatedtests/testdata/population.txt`) touches no file under
  `pkg/sim`, `pkg/mapload`'s script/pathing code, `cmd/missionrun` or
  `scripts/`, so 0/0 is unchanged from base `5ba297ec` and from master.
- `pipeline/check-release-tests.sh` (seat script, run with `AGAINROM_IMPL`
  pointed at this worktree): 8 packages, 170 gated tests, 170/170 ran and 0
  lacked a subject on both the EN and RU lawful roots.

## Open debt

DIV-889 (equipment/inventory/spellbook contents, and — after the session-
refusal hotfix below — session-state fields, cross-chapter mercenary-unlock
history, dismissed-companion state, and per-member identity/DefRow) is
unreconstructed, but is no longer a silent gap: `ExportNativeCitySave`
refuses whenever it would matter, and `SaveSeams` falls back to `.ags`.

A seat hotfix (docs/HOTFIXES.md; review `pipeline/reviews/story1125-pass1.md`
R-1..R-4) widened that refusal past equipment: a hero or mercenary with no
Humans row to bind (DefRow 0), a tavern-enabled-mercenary set that differs
from the current chapter's own declared list, a dismissed chapter
companion, or any of four session-only document fields (pending world-map
return, quick spells, an outstanding offered mission, world-selection
history) now also refuse. Because every valid party's starting hero always
resolves DefRow 0 (`MercenaryType` 0), the native SAV path is now
unreachable for any party at all, not merely one with no worn/carried items
and no known spells — this narrows and replaces the prior correction's own
open-debt claim above. DIV-887's residual debt (a hired mercenary whose
TypeID collides with the wrong nonzero Humans row, cosmetic-only) is
unaffected by this hotfix and remains open.

Items 1-3 and the money/journal/mission part of item 4/5 (from the original
brief's five in-scope items) remain complete and unaffected; the
roster/campaign-record machinery is exactly as useful once a follow-on
story adds the missing Item-object writer, a per-member identity/Humans-row
writer, and the session-field and cross-chapter campaign-history writers
this hotfix's refusals name (reuse targets: story 1096 spellbook records,
story 1109 item weight, story 1115 `SavedObjects`; the imported-city path's
own item handling (`originalcity_sales.go`) turned out to be a
replay-validator over bytes captured at import time, not a general
item-object constructor, so it is not itself reusable — a fresh writer is
needed). World writer, mission-side saves and the original-process write
boundary remain out of scope, as the brief states.
