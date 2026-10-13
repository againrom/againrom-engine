# A mod's map passes the map editor's rules when the mods load

## Intent

One rule table judges a map wherever a map enters the game from a mod or the
editor. Before this story the table `mapRules` (`pkg/game/mapcheck.go`, 14
rows) was read only by the editor's report; the mod load checked the mission
number, the install map and that the map parses, so a map the editor reports as
an error loaded when a mod named it. The story started at public main
`638142e9` (pin k289).

## Authority

Owner decision 1 on architecture audit 80fd0d53, row 4. The original loads a map
without such a check; the refusal is ours: `DIV-3256` (`mods-missions.md`).
`DIV-3257` to `DIV-3263` are returned unused. No research was needed.

## As built

- `MapInspector.CheckModMap(mission, source)` is the entry both callers use.
  The editor's report (`mapInspectionEdits.Report`) and the mod load both call
  the same `report` over the same `checkMapPages`; no rule has a second list or
  a copied predicate. A map the editor's model cannot open is one
  `mission-build` error.
- `SetModMissions` now returns `(warnings []string, err error)`. After the
  existing per-mission checks it runs `checkModMaps`: an inspector over the
  front end's own tables (`modMapInspector`: its archives, its definition table
  after the mod's tables, the installed `structures.reg`), then `CheckModMap`
  for every mission, guest and replacing, with the mission's address as the
  source so the script compile sees the mission number.
- An error finding refuses the mission as every other refusal does:
  `mod "<id>": <file>:<line>: map <map>: error <rule> <record> [cell]: <text>`,
  all error findings joined by `; `.
- A warning refuses nothing. `cmd/againrom` prints each as
  `againrom: warning: mod "<id>": <file>:<line>: map <map>: warning <rule> ...`
  on standard error. The starter's check runs `againrom -check` and shows its
  standard output and standard error together, on a pass and on a failure, so
  the warning reaches the player on the route the refusal already takes.
- The rule `mod-message-text` reads the texts the mod ships: `modPages` holds
  the mission entry and its texts folder files by name. The editor fills it from
  the mod folder on disk (`editorPages`); the load fills it from the
  `TextFiles` read when the mod loaded (`missionPages`). One predicate reads
  either.

## Proof

- `TestSetModMissionsRunsTheMapRuleTable` (fixture mod, guest and replacing
  route): a unit outside the map is refused with the mod, file, line and rule;
  the same map with the unit inside loads with no warning; two ground sacks on
  one cell load and return one `sack-duplicate` warning line. Loss control: with
  the `checkModMaps` call removed the error map loads (both routes returned
  `<nil>`).
- `TestSetModMissionsRuleTableReadsTheTextsTheModShips`: a character the base
  cannot show in a shipped text file gives the `mod-message-text` warning; the
  file without it gives none.
- `TestReleaseModMissionRoutesRefuseAMapTheRulesReport`, EN and RU: the
  reniesta guest mission and a replacing mission 20, assembled from the install
  under test, are refused when a unit lies outside the map and load unedited.
- `TestReleaseModMapRuleCensusRefusesNoInstalledMap`, EN and RU: all 28 numbered
  scenario maps, as a replacing mission with no texts of its own, give no error
  finding. Warnings, EN: `drop-cell` 1, `mod-message-text` 4, `unit-terrain` 5;
  RU: `drop-cell` 1, `mod-message-text` 3, `unit-terrain` 5.
- The existing guest and replacing release tests (`TestReleaseModMission*`,
  `TestReleaseModReplacesACampaignMission`) now pass through the rules on both
  installs. The editor's report, Save As and save into a mod keep their tests.

## Open debt

- The structure-kind and item rules read the installed tables. The item table
  the load holds includes the mod's items; a structure class a mod adds is not
  known to `structure-kind`, so a map placing it is refused. No mod adds a
  structure class today.
- A warning reaches the player on standard error and in the starter's check
  output only; the game window shows none.
