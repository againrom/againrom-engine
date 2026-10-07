# 1332 — FrontEnd lifecycle operations

## Result

New campaign, mission entry, save-restore commit and live-map activation now run
on the components that own their state. The mission-entry sequence reaches
sound through one interface and the viewer's art through one resolved value.
Player-visible behaviour, world hashes, SAV bytes and scenario hashes are
unchanged; this is a refactor.

## Intent

Owner direction: FrontEnd's operations move to the components that own their
state, starting with the game lifecycle. The game-state change is separate from
audio attachment, image preparation and screen switching. Coordination stays on
FrontEnd where a transition spans systems, and it calls named operations
instead of editing each system's internals. A component receives plain values
and narrow callbacks, never the whole `*FrontEnd`.

Done when campaign logic is testable without building a FrontEnd, and a new
audio service needs no change to mission-transition logic.

## As built

### Campaign operations (`pkg/game/campaignsession.go`)

Methods of `CampaignSession`, or functions of plain values:

| operation | replaces |
|---|---|
| `newCampaignCandidate(campaign, units, level)` | the candidate construction inside `prepareNewGameWith` |
| `admitMission(town, n, resuming, level)` | difficulty, restored-progress and mission-number refusals inside the mission opener |
| `(*CampaignSession).clear(campaign)` | the body of `resetSessionForNewGame` |
| `(*CampaignSession).adopt(candidate)` | the state writes of `installCandidate` |
| `(*CampaignSession).endLive()` | the live-triple clear in `installCandidate` |
| `(*CampaignSession).activateLive(mw, n, party)` | the state half of `liveDriver`; returns the replaced map |
| `(*CampaignSession).nextParty(fallback)` | the carried-party half of `NextParty` |
| `(*CampaignSession).recordWin(campaign, n, ..., townArrival)` | the win recording and routing half of `FinishMissionWithRoster` |
| `(*CampaignSession).completeFame(campaign, ...)` | `FrontEnd.completeFame` |
| `detachFameObserver`, `bindFameObserver` | moved from FrontEnd to `CampaignSession` |

`recordWin` reaches the town screen, the shop and the companion grant through
`townArrival`, two function values the caller supplies. `FinishMissionWithRoster`
keeps the party carry (`carryMissionHome`) and the return-city retention, which
read the live map, the install table and the original-city provenance.

### Mission entry (`pkg/game/missionentry.go`)

The 320-line closure in `missionOpenerMode` is now `enterMission(request,
ports)` over named stages: admit, decode, attach audio, `startMission`,
`seatWorld`, `viewerArt().apply`, `openMission`, `settleMission`, commit,
install seams. Statement order of every state-changing step is unchanged. The
exceptions are three plain viewer setters (`SetMissionCutscene`, `ShowReadout`,
and the art setters) that moved relative to each other.

### Audio seam (`pkg/game/missionaudio.go`)

`missionAudio` has four operations: `attach`, `release`, `wireReplies`,
`wireEffects`. `runtimeMissionAudio` is the production service, holding
`*RuntimeServices` and `*InstallResources` and reading both when each operation
runs. `enterMission` and `loadMap` call only the interface. `unitReplies` moved
to `runtimeMissionAudio`.

### Image preparation (`pkg/game/missionart.go`)

`FrontEnd.viewerArt()` resolves the words, fonts, panels, pane art and sack
frames into a `viewerArt` value; `apply(viewer)` is plain setters.

### Screen switching

`installCandidate` is `adopt`, `resetTownSurface`, then either the town arrival
(`restoreWorldMapReturn`) or the mission commit. The town-screen calls sit in
those two methods and nowhere in the campaign operations.

## Proof

- `campaignsession_test.go` runs `newCampaignCandidate`, `admitMission`,
  `adopt`, `clear`, `activateLive`, `endLive`, `nextParty` and `recordWin` on a
  bare `CampaignSession` with no FrontEnd, install or screen.
- The same file drives a whole mission entry through a recording `missionAudio`
  that is not the production service: one attach, one `wireReplies`, one
  `wireEffects` and no release on success; one attach and one release on an
  entry refused after attach; no call on an entry refused by the campaign.
- No-behaviour-change evidence and gate verdicts are recorded in the lane
  notes and the return.

## Measures

Measured with `internal/archtest/cmd/composition` and a go/ast count of
`FrontEnd` receivers in non-test files of `pkg/game`.

| measure | before | after |
|---|---|---|
| component fields | 82 | 82 |
| coordinating functions (three or more components) | 17 | 17 |
| methods declared on `FrontEnd` | 188 | 190 |
| exported methods declared on `FrontEnd` | 69 | 69 |
| methods declared on `CampaignSession` | 0 | 9 |

The coordinating-function count is unchanged: `missionOpenerMode` is replaced
by `enterMission`, which reaches the same components. The FrontEnd method count
rose by two because the closure became coordinator stage methods that still
read several components.

## Open debt

- Return to town is half moved. `carryMissionHome`, `retainReturnCity`,
  `arriveInTown` and `addChapterCompanions` stay on FrontEnd; they read the
  live map, the install table, the original-city provenance and the town
  screen together.
- Save restore decoding (`prepareRestore`, `restoreOriginal`, which copies
  `*f` into a draft) is untouched. Only its commit moved.
- `startMission` and `settleMission` are FrontEnd methods. They need the
  install table, the town and the map world together.
- `loadMap` (non-mission rows) keeps its own art setters; only its audio uses
  the seam.
- `townScreen` holds a `*FrontEnd`; screen switching is a coordinator call, not
  a port.
