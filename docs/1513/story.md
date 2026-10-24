# Tavern talks and quests as town data in a mod

Status: built. With no mod the game is unchanged.

## Intent

A town's tavern talks and quests are rows of the town catalog. A mod's
`data/towns.toml` adds a row, changes a row's mission by its key, or
switches a row off, through `mod.Merge`, as it does for towns. A town a mod
adds has rows of its own. The base first-game town keeps the rows
`scenario.reg` gives it.

## Authority

- Owner direction: BACKLOG T8.
- Seat answers to the design's questions (continuation brief): Q1 is
  measured; a mod's talk text is an install file under `main/text/inn/npc/`
  read by the one text reader; an old save with the mod loads through
  `-mods-accept-unmarked`; LOAD adds no row.
- REG-SCN-064: the tavern's `InnNPC`/`InnMission` arrays pair by position;
  `InnMission` 0 is a talk with nothing to give; the talk text is
  `inn\NPC\npc%02dm%d`. It fixes the meaning of the row's `npc` and
  `mission`.
- TAVERN-023, TAVERN-024: a heard mission is queued and registered in order;
  registration removes the first matching offer and sets the announce flag.
  A mod quest takes this path unchanged.
- TAVERN-TALKPIC-016: a talk cell draws the object of its NPC record, so a
  mod row names a record of the install's NPC table.
- SAV-CAMPAIGN-081, SAV-1112: an accepted pair leaves both live arrays; a
  talk cell follows its array position.

## As built

- `pkg/town`: `Catalog.Tavern []TavernRow` (town, chapter, npc, mission);
  the key is (town, chapter, npc). `Validate` refuses a row of a town the
  catalog lacks, a chapter below 1, an npc or mission outside 0 to 65535 (a
  SAV holds 16 bits) and a chapter naming one NPC record twice over all
  towns. Measured on EN and RU `scenario.reg`: 22 rows each, no chapter
  names an NPC record twice (`TestReleaseTavernRowKeyIsUnique`).
- `pkg/mod`: `ParseTowns` reads `[[tavern]]` tables beside `[[town]]`;
  `mission` and `off` are the settable fields, `town`, `chapter` and `npc`
  the key. `WriteTowns` writes rows too; the base dump writes every
  scenario row.
- `pkg/game/taverncatalog.go`: `scenarioTavern` is layer 0 (each chapter's
  pairs, walking the shorter array, each in the first town that offers its
  mission). `layerTavern` merges the mods' rows over it through `mod.Merge`
  after the towns layer and judges them after each mod: the catalog rules, a
  chapter that is a main mission, a quest mission that is a section of the
  campaign and is offered by the row's town, an NPC record of the install
  and a talk text in the install or a loaded mod. `Campaign.chapter(n)` is
  chapter n with its tavern pairs read from the merged rows.
- A book no mod layers a row over states no rows, and `Campaign.chapter`
  then answers `Campaign.Chapters[n]` itself: layer 0 is read from the one
  scenario reader on every read, so nothing the base does changes.
- The readers that took `Chapters[n].InnNPC`/`Inn` take `Campaign.chapter`:
  the chapter load of the live arrays (`loadMain`), the side-record
  candidates, `Town.ChapterData` without progress, `TavernRoster` and the
  native SAV producer. With several towns, a row a mod added shows in its
  own town (`Town.tavernHere`); every other row keeps the town that offers
  its mission.
- Install files: a text at a new path is admitted under
  `main/text/inn/npc/` alone, named as `TownNPCTextPath` names a talk text.
- Persistence: no new SAV field. A mod row is a pair in the live
  `InnNPC`/`InnMission` arrays and its quest a campaign record; the towns
  kind marks the save, and LOAD without the mod refuses as for every marked
  kind. An old save needs no default.
- LOAD mid-chapter keeps the save's live arrays; a mod row of the loaded
  chapter shows from the next chapter load or a new game.
- `TavernRoster` of a restored chapter reads the registry form its live
  arrays were written against (`Campaign.chapterForms`: the merged rows,
  then the scenario's pairs when a tavern layer changes the chapter). It
  takes the first form whose rows hold the live arrays as an ordered
  sub-list and whose kept speakers the save shows accepted (a campaign
  record, the available list or a finished mission); else the first form
  that holds the live arrays; else the merged rows. A save made without the
  mod therefore shows the same talk cells with the mod as without it. A
  game in progress (no restored record) reads the merged rows alone, and a
  chapter without a tavern layer has one form, so the base is unchanged.
- A towns layer that switches off a talk a companion join waits for in that
  chapter is refused at launch (`checkJoinTalk`, in `layerTavern`'s
  per-mod check), naming the towns mod, file, line and row and the
  companions mod, file and line. The companions kind hands its joins over in
  `modTownInstall.Joins`. `setModCompanions` reads the chapter's talks
  through `Campaign.chapter`: it runs before any tavern layer, so the answer
  is the scenario's pairs as before, and every tavern-pair reader now takes
  the one accessor.
- `tavernTalkTextPath(npc, mission, chapter)` is the one choice of the text a
  tavern row speaks (the mission's leaf, or the chapter's for mission 0);
  the town screen's `townTextPathFor` and the launch check `checkTavernRow`
  call it. `internal/archtest` (`CheckTavernTalkText`) refuses another
  production caller of `TownNPCTextPath` beside it and `isTavernTalkLeaf`.

## Kinds and builders

| kind | builder |
|---|---|
| town catalog | `town.Catalog` / `Validate`; rows are a field of the catalog |
| mod layer | `mod.Merge`, called by `layerTavern` from `layerTownBook` |
| towns file | `mod.ParseTowns`, `mod.WriteTowns` |
| scenario read | `LoadCampaign` / `readChapter`, unchanged; layer 0 is built from it |
| tavern offer list | `Town.chapterOffers`, `TavernRoster`, through `Campaign.chapter` |
| tavern roster | `TavernRoster`, its loop moved into `rosterOn` and run over `Campaign.chapterForms` |
| tavern talk text choice | `tavernTalkTextPath`, called by `townTextPathFor` and `checkTavernRow` |
| companions kind | `setModCompanions`, its talk check through `Campaign.chapter` |
| launch refusals | `layerTavern`'s per-mod check through `mod.Merge`; `checkJoinTalk` added there |
| town offer rule | `townOffersMission`, now shared by `offeredHere`, `checkTowns` and layer 0 |
| install files | `modFiles.serveInstallFiles` |
| town dialogue, part reader | unchanged |
| SAV producer and LOAD | unchanged |

## Proof

- Unit (`pkg/game/tavernrows_test.go`, fixture campaign): add, change and
  switch off; the scenario's chapters unchanged without a tavern row; the
  refusals (unknown town, unknown row switched off, a key a mod may not set,
  a side chapter, an unknown mission, no talk text, no NPC record, a quest
  in a town that does not offer it, an NPC record twice in a chapter, two
  mods without an order); the override with an order; East's rows in East.
- Parse (`pkg/mod/towns_test.go`): the table, Apply and the parse refusals.
- Release, EN and RU (`pkg/game/tavernrows_release_test.go`): the key's
  uniqueness; the `tavern-rows` fixture on chapter 50 shown, saved, loaded,
  heard, accepted, saved, loaded and opened, and refused without the mod;
  the `east-tavern` fixture over `east-town`; a save with no mod mark
  loaded without the mod, refused with it, and loaded with
  `-mods-accept-unmarked` keeping its offers; the same in chapter 50,
  which the fixture changes, comparing offers, roster and talk cells with
  and without the mod; and chapter 50 entered by winning 40 after a LOAD in
  chapter 40, showing the mod's rows (`TestReleaseTavernModOldSave`).
- Unit: a towns layer switching off the talk of a companion join is
  refused naming both mods, and a mission change or another talk switched
  off is accepted (`TestTavernLayerRefusesSwitchingOffAJoinTalk`); the
  chapter load writes the live arrays from the merged rows with a side
  record for the mod quest (`TestTavernLayerChapterLoadTakesMergedRows`);
  `TestTavernTalkTextHasOneChooser` in `internal/archtest`.
- Loss controls of the correction, each run on a temporary edit and
  restored from commit `941ed60fb`: the roster reading the merged rows
  alone fails `TestReleaseTavernModOldSave` on EN (the cells gain npc 32
  mission 31 and npc 52 mission 51); the join check switched off fails the
  join refusal test; `checkTavernRow` choosing its own text fails the
  archtest scan; `loadMain` reading `Chapters[main]` fails the chapter load
  test (live npc `[22 90]`).

## Open debt

- A companion join (`data/companions.toml`) names a conversation of the
  install's rows: the companions kind applies before the towns kind, so it
  cannot name a mod row. The opposite direction is refused: a towns layer
  cannot switch off a talk a join waits for.
- A restored chapter whose live arrays both registry forms hold, and whose
  kept speakers the save does not show accepted in either (a side quest won
  before the save, its record gone), shows the merged rows' kept speakers.
  Inferred from the selection rule; no fixture measures it.
- With two towns that list no missions, a row a mod adds shows in its own
  town only, where a scenario row shows in both.
- A mod talk text has no voice file: mods do not serve `speech.res`. That
  the talk then plays no voice is inferred from that rule, not measured.
