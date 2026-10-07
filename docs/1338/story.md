# 1338 — FrontEnd services and ports

## Result

The three remaining method-value ports of `FrontEnd` and the town screen's
`RuntimeServices` and `Presentation` components are replaced by named ports
and services. The mission transition rule, the city-base projection and the
town screen's audio, draw and art reads each run over an interface or function
type that names what they need. Each is tested with a fake and no FrontEnd.
Player-visible behaviour, world hashes, SAV bytes and scenario hashes are
unchanged; this is a refactor and ships no divergence.

## Intent

Owner direction (maintenance item 13): FrontEnd's operations move to the
components that own their state. Done when campaign logic is testable without
building a whole FrontEnd, and a new audio service needs no change to
mission-transition logic. Passing the whole `*FrontEnd` into a component does
not meet it. This story closes the slice list of 1332, 1333, 1336 and 1337.

## Authority

No ROM1 behaviour is touched. No divergence row is added or changed. DIV-2318
and DIV-2319 are reserved and unused.

## As built

### Mission transition rule (`missiontransition.go`)

`continueMission(t missionTransitions, n, ms, advance)` is the former
`(*FrontEnd).continuity` body. `missionTransitions` has five operations:
`leave`, `finishWon`, `route`, `opener`, `returnToTown`. `frontTransitions`
is the production implementation and the only place the rule meets the front
end. `advanceFrom` builds the mission-entry `advance` port from a rule.
`(*FrontEnd).continuity` remains a one-line call for existing tests.

### City base (`missionports.go`, `currentsave.go`)

`openCityBase(src gameSnapshotSource, table, party)` replaces the method.
`gameSnapshotSource` is `Snapshot(onMap bool)`. The dependency is named as it
is: the projection reads the whole game state as a SAVE would capture it, so
the port is the snapshot and is not narrowed to a part. `cityBaseFrom` builds
the existing `cityBaseFunc` port from a source and the table. The body of
`(*FrontEnd).currentCityBase` is the free function `cityBaseDocument(table, s)`.

### Town screen services (`townservices.go`)

| field | type | replaces | reads |
|---|---|---|---|
| `sound` | `townAudio` | `rt` | sound, speech and ambient devices, ambient seed |
| `draws` | `townDraws` | `rt` | animation clock, five bounded draw sources |
| `art` | `townArt` | `pr` | menu frame, character panes and corners, shop, tip, document font, world map |
| `latches` | `*townAmbientLatches` | `pr` | bird delay latch and value, star terminal counter |

The three latch fields moved from `Presentation` into one `townLatches` field
(`townAmbientLatches`), so the screen holds the latches and not the component.
`openMission` has the named type `missionDoor`. The screen still holds `sess`,
`in` and `pc` whole.

### Picker map art (`missionart.go`)

`loadMap` (non-mission rows) dresses its viewer through `pickerArt`, resolved
by `pickerArtSource` and applied in the setter order the function had before:
font, card font, readout off, attack pointer, command panel, bottom HUD,
character panes, sack frames, sack boundaries. It is not `viewerArt.apply`.

Visible difference kept: a mission viewer also receives the install words
(`SetWords`), the dialog frame (`SetDialogFrame`) and the character pane
filler (`SetCharacterPaneFillerArt`); a picker map receives none of the three,
so the debug map picker shows the viewer's own defaults for them. Moving
`loadMap` onto `viewerArt.apply` would change that, so it does not.

## Differences from the base

Statement order, error strings, nil guards and hashed state are unchanged.
The screen asks its services at the moment it plays or draws, as it read the
components before. A nil device, clock or draw source behaves as before.

## Proof

- `serviceports_test.go` builds no FrontEnd. It runs `continueMission` over a
  fake rule for every route (mission, town, ending, stay), a loss, abandon,
  restart and a world-less mission; `openCityBase` over a fake snapshot
  source; and the town screen over a fake audio service, a fake draw service
  and a fake art service.
- `TestTownScreenPlaysThroughASubstitutedAudioService` swaps the audio
  service, shows the crowd loop stopped and the effects and speech devices
  taken from the fake, and runs `continueMission` beside it unchanged. The
  mission transition and the town logic have no audio parameter.
- `TestResetForNewGameDropsExactlyTheGamePopulation` classifies the new screen
  fields as kept across a load.
- Unchanged EN and RU release tests, the milestone-2 acceptance family and the
  scenario hashes carry the no-behaviour claim. Gate verdicts are in the lane
  return.

## Measures

| measure | before | after |
|---|---|---|
| component fields | 82 | 80 |
| coordinating functions (three or more components) | 15 | 15 |
| `CommentBytes` baseline | 8620735 | 8618593 |
| `townScreen` fields typed `*RuntimeServices` or `*Presentation` | 2 | 0 |

The field count falls by two because three `Presentation` fields became one.
The storyguard baseline falls by 2142 bytes; the retired narrative of the old
`continuity` doc is replaced by a shorter one on `continueMission`.

## Open debt and item 13

Item 13's two done criteria are met. Campaign logic (the win transition, the
city base, the town screen's shop, tavern and campaign operations) runs with
no FrontEnd, over fakes. A new audio service is a `townAudio` implementation
for the town and a `missionAudio` implementation for the mission entry;
`continueMission`, `enterMission` and the town logic take no change.

What remains is wiring and not logic:

- `frontTransitions`, `cityBaseFrom(f, ...)` and `(*FrontEnd).MissionOpener`
  still close over a `*FrontEnd`. They are the composition seam; the callees
  do not depend on it.
- `gameSnapshotSource` is the whole game state. It is a named dependency, not
  a narrow one.
- The two audio interfaces (`townAudio`, `missionAudio`) are separate. A new
  audio service implements both.
- The screen holds `sess`, `in` and `pc` as whole components.
