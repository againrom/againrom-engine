# The map editor picks a message's speaker by a click on a unit

## Intent

Each speaker row of the message popup has an "On map" button (RU «На карте»). A press arms the
editor's parameter pick for that card's speaker; the status line reads "Click a unit on the map for
the speaker; Esc cancels." (RU «Щелкните по отряду на карте, чтобы выбрать говорящего; Esc
отменяет.»). Esc or a second press of the same button disarms it. The popup stays shown while this
pick is armed, so the second press can reach the button. Moving the card (Alt+Up, Alt+Down, drag)
moves the armed field with it. Apply, Cancel and the cross of the armed card end the speaker
field and so disarm the pick; the next map click is then the ordinary one (`dropLostPick`, run after
every pointer and key event). The button is as wide as its caption in the font the popup draws it
with (`rowButtonWidths`); the speaker field narrows by that width and the cross keeps its square.

A click on a unit sets the card's speaker to the speaker record that shows this unit at mission
start, in the game's own terms:

- A person placed by its own NPC record (the class sub-id `npcFields` reads) takes that record when
  the record resolves to this person.
- Otherwise a person takes the lowest record id that resolves to it.
- A creature takes the lowest record whose dialogue picture is the picture the creature's info
  window shows at its tier.
- A unit no record shows leaves the speaker as it was and keeps the pick armed; the status line adds
  "No speaker record shows a unit there." (RU «Здесь нет отряда, которого показывает запись
  говорящего.»). A click on no unit is the same miss.

The speaker list's faces, and the person window's NPC list's, are the faces a dialogue shows at
mission start on this map: a record a live placed person answers shows that person's own figure,
dressed as the mission start dresses it. A person placed dead (health 0) and a placement on the
border ring, which the mission withdraws, answer no record: the record shows its own picture and
a click on that unit sets no record. Where a dead and a live unit of the same look can both show a
record, the live one is the one shown and picked. The faces are built again when the map's bytes change.

## Authority

- Tester, through the owner: the most common need when finishing a map is to choose a message's
  speaker by a click on the map, so that the speaker looks in the dialogue as the unit looks in its
  info window.
- Owner: screens are at least 1920x1080; the map editor is our own program, so no research and no
  DIV row; map editor stories do not count toward a release; no owner test build.
- The game's resolution this story reads, not changes: `DLG-SPEAKER-023` (the live-actor predicate
  over the record's `Flags`, `Face` and `Picture`), coded as `speakerCast.resolveNPC` and `matches`
  in `pkg/game/speakeractors.go`. The info window's tier digit is an owner ruling recorded at
  `classPortrait` (`UNIT-PICT-036`: the original's info panel takes no tier, its dialogue panel
  does); the pick follows this build's info window.

## As built

| kind | builder |
|---|---|
| map pick | the parameter pick (`pkg/ui/mapeditorparampick.go`): one more target, `targetSpeaker`, with its row in `paramPickTexts`; `pickFieldTarget` asks `messageFieldTarget` on a message page; `armParamPick` arms it and now disarms on a second press for the same field; the click goes `pickAt` → `paramAt` → `unitValueAt` → `paramPicked` as every parameter pick does. `speakerPicking` keeps the message popup in `objectSpecBase` while it is armed |
| popup row button | `messageSpec`'s card head row (`pkg/ui/mapeditormessage.go`) carries `actPickParam` after the cross, recorded in `o.rowActs` as the script popup records its row buttons; `messageRowButton` routes it to `armParamPick` |
| unit to speaker | `InspectionMessages.UnitSpeaker`, the game host's `mapInspectionEdits.UnitSpeaker` (`pkg/game/mapinspectionnpc.go`); candidates are `npcChoices` (the speaker list's own ids) |
| speaker candidates | the game's cast: `missionSpeakers` over `placedEntities` of the map with its border placements withdrawn (`mapload.WithdrawBorderPlacements`) and the roster `mapload.FromALMRoster` returns, through `newSpeakerCast` over a `speakerWorld` whose `alive` and `worn` are the mission driver's `entityAlive` and `equipmentOf` over the mission-start world. `editorSpeakers.placed` maps an authored placement to its place in the withdrawn map (`mapload.BorderWithdrawn`) |
| resolution predicate | `speakerCast.resolveNPC` (and its `matches`), unchanged |
| mission start world | `mapload.FromALMRoster` at `DifficultyNormal` and `entityTiers`, held in `editorSpeakers` and rebuilt when the map bytes or the table change (`missionStart`) |
| creature picture | `infoPortrait` (`pkg/game/portrait.go`), the info window's flat-picture arm factored out of `unitPicture`, which calls it unchanged; equal `classPortrait` cache addresses are one picture file |
| speaker face | `speakerResolver.SpeakerFace` with the mission-start cast, cut by `ui.SpeakerPortrait`; the inspector keeps its picture caches (`speakerPictures`, `speakerFigures`) |
| speaker list faces | `mapInspectionEdits.npcPortraits`, moved from `MapInspector` so it reads the map; both callers, `MessageFields` and the person window's NPC field, call it |
| status words | `pkg/words/text/{en,ru}/strings.toml`: `editor.param.pickspeaker`, `editor.param.nospeakerunit`; the button reuses `editor.object.pick` |

No second pick mode, click path, candidate walk, face comparison or portrait composer was added.

## Proof

Focused ordinary tests at 1920x1080:

- `pkg/ui/mapeditorspeakerpick_test.go`, EN and RU words:
  `TestMessageSpeakerTakesAClickOnAUnit` (the button arms and the status line names the pick; a
  second press and Esc disarm; open ground and an unshown unit leave the speaker, keep the pick and
  add the miss words; a shown unit sets that card's speaker only; the second card's button fills the
  second card) and `TestPickedSpeakerAppliesAsTheListsChoice` (pick then Apply writes the same parts
  as the list's choice then Apply).
- `TestCardRowButtonCoversItsCaption`, `TestSpeakerPickFollowsItsCard` and
  `TestSpeakerPickEndsWithItsTarget` (Apply, Cancel and the cross) in the same file.
- `TestSpeakerCastIsTheGamesLiveCast` in `pkg/game/mapeditorspeakerpick_test.go`: a corpse alone, a
  corpse and a live man, and a withdrawn border placement; the fixture's units stand inside the
  border ring.
- `pkg/game/mapeditorspeakerpick_test.go`, over the game's host and a fixture map of a paladin
  placed by his record, a man fighter placed by his Humans row, a woman no record shows and a
  tier-2 goblin: `TestUnitSpeakerIsTheRecordThatShowsTheUnit` (own record 25 over the lower record
  3 that also resolves to the paladin; the fighter's lowest record 12; none for the woman; record 41
  for the goblin, whose face equals the mission driver's `unitPicture` and record 40's at tier 1 does
  not), `TestSpeakerListPortraitsAreTheMissionStartFaces` (each list face equals, pixel for pixel,
  the face `openMission`'s driver shows; record 3's face differs from its bare record picture;
  removing two units changes record 3's face) and `TestPickedSpeakerWritesTheListsMarkup` (the popup
  button, a map click and Apply write `<part=1,npc=12>` as the list's choice does).
- `pkg/game/mapinspectionspeakerportrait_test.go`: `TestPersonWindowNPCListCarriesTheSpeakerListsFaces`
  now expects record 7, a record with no picture of its own that a placed person answers, to carry
  that person's mission-start face.

Golden moved: `pkg/ui/testdata/mapeditor_speaker_portraits.golden`. The "On map" button is as wide
as its caption, so the speaker field narrows by that width and the tip's box, centred on the pointer
at the list row's centre, moves left: EN `(1086,325)-(1193,444)` to `(1055,325)-(1162,444)`, RU
`(1086,325)-(1193,444)` to `(1010,325)-(1117,444)` with the fixture font, whose advance is 8 pixels
per glyph. Both frame hashes change; rows and face pixel counts are unchanged.
The modes golden did not move.

## Open debt

- No MCP command arms or answers the pick; an MCP client reads `UnitSpeaker` only through the host.
- A unit no existing record shows cannot become a speaker: a map-own speaker record would need a new
  record writer and a decision on where it is stored.
- The count of shipped-map units the pick refuses was not measured; it needs the installed maps,
  which a lane does not read.
- The mission start is read at Normal difficulty with no party: a record that answers a party hero
  (`Me`, `MySex`, `MyClass`) or a unit a script adds later is not seen.
- Release witnesses for the next release, EN and RU (`gameversions/en`, `gameversions/ru`):
  `TestReleaseMessagePickerParts`, `TestReleaseMessagePickerPopulation`,
  `TestReleaseMessagePickerCampaignPage`, `TestReleaseTriggerMessageFrames`,
  `TestReleaseTriggerMessageEditsAndUndo` and `TestReleaseMapEditorOverrideFrames`. None asserts the
  speaker list's faces or the popup's row buttons, so none was changed; none ran in the lane.
