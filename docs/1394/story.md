# A hand-over keeps the player's key and an original keyless Sack keeps its Token

## Intent

Two SAV round trips stop changing state. A script hand-over no longer gives the
player's own Player the key of the Player the actor left, so the owner kit
`owner-sav-story1225/game9253.sav` played and cycled loads to the World that
saved it. An original Sack whose cell names no Sack binds at LOAD and keeps its
`RuntimeID` and Identity across SAVE. Base: public main `05b24704` (pin k235),
reconciled with main `49141cdc` and `db226edb`; the knowledge pin stays k235
(`d929f29a`).

The story set out to make the milestone-2 leg pass with no failure. It does
not: the eleven `TestMilestone2*` failures on `2026-10-07/saveorcsdontgo.sav`
and that file's two unbound Sacks are owned by a separate story (Open debt).

## Authority

- Owner ruling (AGENTS.md, Boundaries): SAV is the only save format; a state
  loaded from an original SAV and a state the engine created are one kind of
  state, written by one producer from the current state.
- Owner decision on this story: one kind of World from mission start is a
  separate story; this story lands the game9253 round trip and the keyless
  original Sack.
- `PARTY-JOIN-025` step 3 and `UNIT-OWNER-009`: the hand-over writes the
  destination Player into the actor's owner Reference (actor+0x14).
- `SAV-TOKEN-034` (High): a Sack's Token, `RuntimeID` and Identity included,
  is written from the Sack itself. `SAV-SACKREMOVE-591` (High): removal clears
  a cell's Sack slot without testing which Sack it names.
  `SAV-SACKCALLER-593` (High / Medium / Unknown): lookup reads the cell slot
  only. No claim states how an original file comes to hold a Sack whose cell
  names no Sack (Unknown).

## As built

### Hand-over reference (`pkg/sim/script.go`)

`handOver` first runs `handOverReference`. When the actor's Reference scalar is
known and its owner changes, the Reference becomes the destination slot's key:
a saved Group's reference to that Player, else the Reference most of the slot's
other actors carry, the lower key on a tie. With no key the Reference stops
being known. Before, the Reference kept the source Player's key (`0x51000060`,
slot 5) after the owner became slot 1; the writer votes a Player's `This` per
slot, four handed-over actors outvoted the hero, and after LOAD the hero's
Reference was `0x51000060`.

### Removed body's terminal basis (`pkg/sim/nativebasis.go`)

A body torn down before it reached bones (Decay below DecayBones at removal)
retains the terminal order and action words SAVE writes for it
(`nativeRemovedBasisNow`), not the raw words. Before, the retained row held
`U50=11 U54=0`, SAVE wrote 16/16, and LOAD restored 16/16, so
`world.removedNativeBases` differed after one cycle.

### Keyless original Sack (`pkg/game/savsackobjects.go`, `savitemobjects_project.go`, `pkg/sim/savedobjectsworld.go`)

`savedSackCellAdmits` is the cell join of a Sack binding: a cell whose Sack
slot names the Sack's Identity admits it, and so does a cell with no record or
an empty slot; a slot naming another key refuses. The caller already requires
a unique source Sack on the cell and a unique Identity. Original import,
v1 validation and v2 projection validation use it.
`sim.SavedSackBinding.Keyless` states a keyless join, and
`World.ImportSavedObjects` then requires the cell to name no Sack; an exact
binding still refuses a missing cell node.

### Not built

Option B (World-built SAVE joins an unbound Sack to the loaded document's sole
unbound root on its cell) is parked unapplied in the evidence directory as
`loaded-sack-join-wip.patch`. A LOAD-time binding of unregistered current-path
Sacks broke `TestCurrentMergedSackFromLegacyFixture` (live World unbound, cold
LOAD bound, World hash differs) and was removed.

## Proof

Evidence directory: `review/story1394-m2-zero/`.

- Focused, failing before their fix: `TestHandOverWritesTheDestinationPlayerReference`,
  `TestHandOverToAPlayerWithNoKeyForgetsTheOldReference`,
  `TestRemovedTerminalBasisRetainsItsTerminalOrderWords`,
  `TestSavedSackKeylessBindingNeedsACellNamingNoSack` (does not build before),
  `TestReleaseSackWithoutCellKeyKeepsItsToken` (`oldsaves7/game0006.sav`, Sack
  at 71,116; `focused-sackkey-before.txt`).
- game9253 gate before and after: `gate9253-base.txt`, `gate9253-fix.txt`.
  The other names the gate log listed (`world.actorTraversal.len`,
  `world.areaCostLive`) are listing noise: once one hash-relevant field
  differs, `savGateWorldFields` reports every later copied field that differs.
- Final commit: see the results table below.

| check | root | result | file |
|---|---|---|---|
| ordinary `go test` of `pkg/sim`, `pkg/game`, `cmd/againrom`, `internal/archtest`, `internal/storyguard`, `internal/gatedtests`, `internal/divledger` | - | PASS | `final-ordinary.txt` |
| `TestReleaseSackWithoutCellKeyKeepsItsToken` | RU, EN | PASS | `final-sav-{ru,en}.txt` |
| `TestRelease.*(Sack\|Handover\|HandOver\|Join\|Corpse\|Terminal\|Ground\|ItemObjects\|Handoff\|Dead\|Removed)` | RU, EN | 63 PASS each, no fail or skip | `final-rel-{ru,en}.txt` |
| `TestReleaseSecond*` | rom2-en, rom2-ru | 52 PASS each; `TestReleaseSecondPhysicalSyntheticCityAppRoute` is a ROM1 test and passes on RU and EN | `final-rom2-{en,ru}.txt`, `final-rom2-physical-{ru,en}.txt` |
| `scripts/check-milestone2-acceptance.sh` (includes `TestSAVRoundTripGateNewGameKits*` and `TestSAVWriterCensusChangedWorlds*`) | EN, RU | M2-RESULT | `final-m2.txt` |

No golden, trace or witness hash moved.

## Open debt

- One kind of World from mission start (separate story, owner decision): a
  World built from mission start holds no group, motion, cell-plane,
  spell-graph or session-clock carrier, and LOAD of the SAV it writes drops the
  same carriers (`restoreCurrentPolicy`). Eleven `TestMilestone2*` instruments
  fail on `saveorcsdontgo.sav` (`DIV-2885`), and its two Sacks at 20,106 and
  70,38 stay unbound and take new Tokens at each SAVE, two `sack|missing
  original` rises in `TestSAVWriterCensusChangedWorldsPart6` (`DIV-2884`).
  Evidence: `m2-corpus1-keep.txt` (a policy keeping the carriers after import
  turns only Buildings to PASS on this file), `m2-corpus2-base.txt` (three
  current-engine new-game saves, missions 10, 20 and 80, fail the same family),
  `m2-corpus3-stripped.txt` (with the supplement stripped the original path
  refuses the file: late-dead stage 4 HP -601, a dead actor with contents).
- Unknown: where a keyless Sack comes from in the original (`DIV-2506`).
