# 1333 — FrontEnd return and restore

## Result

The return to town, the mission-entry sequence and the decoding of a native
restore now run on named components and plain values. A won mission is
finished, a town is arrived at and a snapshot is decoded on a bare
`CampaignSession` or from plain values. The mission-entry sequence is a
function over named ports and is driven in tests with no FrontEnd. Player-visible
behaviour, world hashes, SAV bytes and scenario hashes are unchanged; this is a
refactor.

## Intent

Owner direction: FrontEnd's operations move to the components that own their
state. Done when campaign logic is testable without building a whole FrontEnd,
and a new audio service needs no change to mission-transition logic. Passing
the whole `*FrontEnd`, or a disguised equivalent, into a component does not
meet it. A port is the narrowest thing that names one service; none exposes the
FrontEnd or a view of it.

This story continues 1332 (campaign operations, the audio seam, mission-entry
stages) with the return to town, the entry sequence itself and restore decoding.

## As built

### Return to town (`pkg/game/campaignreturn.go` and beside it)

Methods of `CampaignSession`. The install's definitions arrive as one plain
value, `townInstall` (table, body list, campaign, NPC-name lookup), built by
`InstallResources.townInstall()`.

| operation | replaces |
|---|---|
| `finishMission(in, n, party, w, ids, roster, arrival)` | the body of `FinishMissionWithRoster` |
| `carryMissionHome` | `FrontEnd.carryMissionHome` |
| `retainReturnCity`, `retainMissionCity`, `missionCityProvenance` | the same-named FrontEnd methods |
| `arriveInTown(in)` | the campaign half of `FrontEnd.arriveInTown` |
| `addChapterCompanions`, `carryTownCompanion`, `settleCompanionArrival`, `joinOnTalk` | the same-named FrontEnd methods |
| `captureMissionReturn`, `missionObjectSource`, `canLeaveMission` | the same-named FrontEnd methods |

`FrontEnd.arriveInTown` is the screen switch: the campaign arrival, then the
town screen's reset of the remembered world-map position on a non-restored
return, in the original statement order. `FrontEnd.FinishMissionWithRoster`
calls `finishMission` with the arrival callbacks and nothing else.
`originalCitySaveState.captureSession` takes the `CampaignSession` it reads.

### Mission entry (`pkg/game/missionentry.go`, `missionports.go`)

`enterMission`, `startMission` and `settleMission` are functions. `enterMission`
receives `missionPorts`:

| port | what it names | production implementation |
|---|---|---|
| `audio missionAudio` | sound (1332) | `runtimeMissionAudio` |
| `install missionInstall` | map decode, table, roster naming, body loading, driver construction, projectile art, briefing | `installMission` over `*InstallResources` and the marker setting |
| `profile missionProfile` | the stored game options and the retreat cycle | `*PersistenceContext` |
| `session *CampaignSession` | commit of the live map, the quick-spell bindings, leave-mission rule | the front end's session |
| `display missionDisplay` | five plain viewer settings | values copied at entry |
| `art`, `cityBase`, `advance` | viewer art, the open town's city graph, the won-mission continuation | method values of the front end |

`cityBase` and `advance` stay callbacks because each spans the whole game state
(the save snapshot; screens and the opener of the next mission). The entry
sequence reads neither.

Moved onto components so a port is a component method: `canonicalizeJoinedRoster`,
`localizedNPCName`, `modCharacterName` to `InstallResources`;
`applyFreshGameOptions`, `ensureGameOptionApplication`, `cycleRetreat` to
`PersistenceContext`. `seedCurrentCityObjects` is a function of the mission,
the town, the table and `cityBase`.

State-write order in the entry is that of 1332. Two reads moved earlier: the five
display settings and the marker setting are read when the ports are built, at
the start of the entry, instead of in the middle of it. Nothing between writes
them.

### Restore decoding (`pkg/game/resume.go`, `originalsave.go`)

- `decodeRestore(snapshot, townInstall)` is the validation and candidate
  construction of `prepareRestore`, from plain values. It returns the candidate
  and the validated snapshot. `prepareRestore` calls it and then builds the
  mission screen.
- `decodeOriginalCampaign(file, bytes, campaign, quickSpells)` is the campaign
  decode of `restoreOriginal` (difficulty, campaign projection, town, current
  session supplement, quick spells).
- `validateOriginalCityBaseline` is a method of `townInstall`.

## Proof

- `campaignreturn_test.go` runs `arriveInTown`, `addChapterCompanions`,
  `carryMissionHome`, `canLeaveMission` and `finishMission` on a bare
  `CampaignSession` with no FrontEnd, install or screen.
- `campaignsession_test.go` drives a whole mission entry from components: an
  install over a synthetic archive, a bare session, a profile with no stored
  options and a recording audio service. It covers the commit to the session it
  is given, the deferred commit through `activate`, a refusal after audio was
  attached, a refusal by the campaign, and an install port that refuses the
  decode.
- `restoredecode_test.go` runs `decodeRestore` and `decodeOriginalCampaign` from
  plain values: a town snapshot, a mission snapshot with no world, a malformed
  campaign record and a pre-town campaign.
- No-behaviour-change evidence and gate verdicts are in the lane notes and the
  return.

## Measures

Measured with `internal/archtest/cmd/composition` and
`review/story1332-frontend-lifecycle/tools/count.go` (go/ast count of receivers
in non-test files of `pkg/game`).

| measure | before | after |
|---|---|---|
| component fields | 82 | 82 |
| coordinating functions (three or more components) | 17 | 16 |
| methods declared on `FrontEnd` | 190 | 170 |
| exported methods declared on `FrontEnd` | 69 | 69 |
| methods declared on `CampaignSession` | 9 | 22 |
| methods declared on `InstallResources` | 0 | 4 |
| methods declared on `PersistenceContext` | 0 | 3 |

`arriveInTown` and `enterMission` leave the coordinating set. `missionPorts`
joins it: it is the one place the entry's services meet the FrontEnd.

## Open debt

- `restoreOriginal` keeps about 380 lines. Its town branch builds a detached
  copy of the FrontEnd (`draft := *f`) because `restoreHiredMercenaries` builds
  mercenary squads through the town screen, and `townScreen` holds a
  `*FrontEnd`. Its mission branch reads `f` for the table, the archives and the
  mission opener. Only the campaign decode moved.
- `frontWorldMapFilteredMarkers` reads the first-use world-map art cache through
  a copy of the FrontEnd (`markerFront := *f`).
- `continuity`, `leaveMission` and `loadMap` stay coordinators: they open
  screens and the next mission's opener.
- `viewerArt` resolves first-use caches stored in `Presentation`, so the entry
  receives it as a callback instead of reading install values.
- `loadMap` (non-mission rows) keeps its own art setters.
- `townScreen` holds a `*FrontEnd`; screen switching is a coordinator call, not
  a port.
