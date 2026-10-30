# One World carrier set; absence rules read the World

## Intent

A SAVE's absence rules and the second game's mission seat decided by the
record the mission last loaded (`savedDocument`/`SavedDocument` with a nil
`Document`), not by the World. The SAVE bytes and the seated Scenario bank
therefore depended on which record was loaded last. This story makes one
carrier set for the World state a SAV carries, gives the World a
loaded-record carrier that the SAV LOAD writes, and switches the two rules
that tested the last loaded record to that carrier.

## Authority

- Owner direction: the World carrier set is its own story, ahead of the
  projectile records and the campaign projection (decision item 1 of
  `pipeline/reviews/arch-audit-3c0dbfc3.md`).
- Audit shape: carrier descriptors {home, capture, absence, transform,
  writer}; one loaded-record carrier written to the World at LOAD; keep the
  byte codecs and form wrappers (rank 1 of `arch-audit-3c0dbfc3.md`,
  `arch-audit-91ad8a33.md`, `arch-audit-a5c60f20.md`).
- ROM1 behaviour is not changed. DIV-4175 keeps its text: the SAVE of a
  World loaded without a decoded document leaves the unbound spell effects
  and undriven projectiles out of its World copy.

## As built

### Loaded-record carrier

`sim.LoadedRecord` (`pkg/sim/loadedrecord.go`) is a World field with three
values: `NoLoadedRecord` (a started mission, or a snapshot resume with no SAV
document), `LoadedDocument` and `LoadedWithoutDocument`. It is provenance,
not play state: no tick reads it, the byte form does not carry it, and a
decode carries the receiver's value across, as the ghost template is. The
World hash and the byte form are therefore unchanged for every World.

The one writer is the SAV LOAD restore, `restoreSavedDocumentDecoded`
(`pkg/game/savdocument.go`), at the line that seats `ms.savedDocument`. The
two LOAD stagings that later replace the live World copy it by value
(`savsackobjects.go`, `savplayerpurses.go`); the sim staging round trip in
`originalliving.go` copies it onto its fresh receiver.

### The carrier set

`worldCarrierSet` (`pkg/game/worldcarriers.go`) is an ordered slice of
`worldCarrier` rows {home, capture, decode, absence, transform, seat,
writer, late}. The SAV producer, the LOAD and the mission seat run the rows
through `captureWorldCarriers`, `decodeWorldCarriers`,
`leaveAbsentCarriersOut`, `transformWorldCarriers`, `seatWorldCarriers` and
`writeWorldCarriers`, in row order, at the points where the base called each
hook directly.

| Row (home) | Step | Hook | Run from |
|---|---|---|---|
| World.LoadedRecord | capture, decode | Snapshot copy and SAVE World copy | `FrontEnd.Snapshot`, `currentMissionDocument` |
| saved spell effects and projectiles (DIV-4175) | absence | `leaveUnboundEffectsOut` | `currentMissionDocument` |
| a Pointer's UseAmount | transform | `restorePointerSlots` | `applyOriginalStructuresCurrent` path, `restoreStructureUseMetadata` |
| ROM2 scenario state (Scenario bank) | seat | `campaignService.seatWorld` | `seatWorld` (`missionentry.go`) |
| town spell-effect owners | writer | `finalizeCurrentTownEffectOwners` | `ExportCurrentSave` |
| loaded Token Reference misses | writer | `finalizeLoadedTokenReferences` | `ExportCurrentSave` |
| object identity absence | writer | `finalizeCurrentObjectIdentityAbsence` | `ExportCurrentSave` |
| order spell absence | writer | `finalizeCurrentOrderSpellAbsence` | `ExportCurrentSave` |
| spell book selection | writer | `finalizeCurrentBookSelection` | `ExportCurrentSave` |
| World carrier absence | writer | `finalizeCurrentCarrierAbsence` | `ExportCurrentSave` |
| cell plane residue | writer | `FrontEnd.finalizeCurrentPlaneResidue` | `ExportCurrentSave` |
| World spell-effect Tokens | late writer | `finalizeCurrentWorldEffectTokens` | `ExportCurrentSave` |

### Switched absence rules

- `leaveUnboundEffectsOut` (`savworldeffects.go`) applies when the SAVE's
  World copy holds `LoadedWithoutDocument`. The base tested
  `s.SavedDocument != nil && s.SavedDocument.Document == nil` in
  `currentMissionDocument`.
- `secondCampaignRules.seatWorld` (`campaignsecond.go`) seats the campaign's
  Scenario bank on a fresh mission or when the World does not hold
  `LoadedDocument`. The base tested `ms.savedDocument == nil ||
  ms.savedDocument.Document == nil`.

In production the two tests agree for every LOAD: the carrier is written
where `ms.savedDocument` is, and a Snapshot copies both at one capture. They
differ only where the World and the mission's record disagree, which the
focused witnesses construct.

### Guard

`internal/archtest/worldcarriers.go` names, for each hook of the set and for
`sim.World.SetLoadedRecord`/`LoadedRecord`, the production functions that
may reference it: the set itself; for the setter also the LOAD restore; for
the reader also the two switched rules. A direct call of a hook elsewhere is
a finding. The reference-owner check is the one `CheckSaveCapture` uses,
factored into `checkReferenceOwners`.

### Carriers left

These keep their own capture, absence or LOAD derivation outside the set.
None is a second set: each is one call chain the set does not yet run.

| Carrier | Files | Why left |
|---|---|---|
| document capture chain (`projectSavedActorValues` through `projectSavedWorldEffects`) and its LOAD validators | `savdocument.go` | capture order and LOAD validation order differ; a row per family needs a stage order per step |
| kept from the loaded record: `retainDeadLoad`, `retainDeadBook`, `retainDeadEffects`, `loadedDeadObject` | `currentdeadconstruct.go` | read the loaded record's content, not its kind; a World home changes the World byte form |
| identity joins and effect-key reservation from the loaded record | `currentworldbuild.go` | as above |
| Player purse seat (`ms.savedDocument.PlayerPurses == nil`) | `missionentry.go` | reads a record part, not the record kind |
| application state from the loaded document | `savapplication.go`, `missionentry.go` | reads the document's content |
| script slots in World form 129 and the current-actions program | `pkg/sim/scriptslotbinary.go`, `currentscriptprogram.go` | byte codecs kept; only the LOAD derivation is in the set |
| runtime ids, body words, effect Tokens capture and restore, kept actors, Player colour, world policy, projectile counter, walk claims, passive bodies, mover radius | `pkg/sim/runtimeids.go`, `savbodywords.go`, `worldeffecttokens.go`, `currentworldpolicy.go`, `projectilecounter.go`, `walkclaim.go`, `passivebody.go`, `moverradiusbinary.go` | not reached in this story |

## Proof

- Loss control, the SAV corpus round-trip gate on the first game's EN root
  (`TestSAVRoundTripGateCorpusPart[1-4]`, `AGAINROM_SAVE_CORPUS` 146 files):
  base `1f4f0865` and candidate both pass with town 44/44, resave 84/85
  (1 unreadable), played 17/17, dying 67/67; the one case
  `2026-08-02/game0003.sav` passes before and after.
- `TestTheLoadWritesTheWorldsLoadedRecord`: a second-game mission file LOAD
  holds `LoadedWithoutDocument`; the cold LOAD of our SAVE holds
  `LoadedDocument`.
- `TestTheSavesAbsenceReadsTheWorldNotTheLoadedRecord`: the SAVE's World
  copy drops an undriven projectile when the World holds
  `LoadedWithoutDocument` under a record with a document, and keeps it when
  the World holds `LoadedDocument` under a record without one; the live
  World's hash is unchanged.
- `TestSecondSeatWorldKeepsTheEngineDocumentsLiveBank`: two added cases put
  the World and the record in disagreement; the seat follows the World. The
  base rule gives the opposite bank in both.
- `TestASaveKeepsTheLiveWorldAndAResaveItsBytes`: on the mission file
  fixture a SAVE leaves the live World's hash unchanged and the next tick
  equals the next tick of a twin that did not SAVE; our SAV loaded and saved
  again writes equal bytes.

## Open debt

- The first resave of a SAVE taken after a second-game mission file LOAD
  writes `OfferLabels.Chapter` 0 where the first SAVE wrote 10 (fixture
  `secondMissionFile`, 27063 bytes each). It is the campaign projection's
  town chapter, not a World carrier, and no DIV row names it (searched
  `docs/divergences/*.md` for `OfferLabels`). The later resaves are equal.
- The carriers left above.
