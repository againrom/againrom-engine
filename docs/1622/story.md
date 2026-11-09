# Story 1622: one minimap plan producer

## Intent

One function produces every `minimapPlan`. What differs between the installed picture and the picture of a viewer with no paint data is a value it reads. A mod terrain group's colour reaches every viewer that draws a minimap. No frame of a map without mod terrain, hit area, press behaviour, SAV byte, RNG step or World hash changes.

## Authority

Owner direction on the architecture audit: "Берем те 7 видов, пытаемся достигнуть 13". Kind R49 of `pipeline/reviews/arch-audit-5d474c65.md`, row 14. Earlier owner answer on the same kind: one minimap producer for the game and the editor, the editor takes the mod groups' colours. Architecture story: no DIV row. No player-visible change on any map without mod terrain, so no CHANGELOG line; a mod author's minimap colour on the picker map is not a player-facing change.

## As-built behaviour

`minimapProducePlan` (`pkg/ui/minimap.go`) is the one producer. It reads a `minimapRules` value: colour source, fog rule, mark rules (`minimapMarkRules`: owner colour, unit gate, structure gate, unit extent), outline rule and outline mode. `minimapRulesFor` picks the value: `minimapInstalledRules` or `minimapSymbolicRules`, by `minimapOriginalActive`. The two rule values replace the two producers `minimapOriginalPlan`/`minimapOriginalMarks` and `minimapWindowed`/`minimapMarks`/`composeMinimapLayer`.

Windowing, mark translation and the outline are in the producer. `minimapPlanMarks` is the one mark walk. `minimapTerrain` is the one terrain builder; it is the one place a mod group's colour applies, and `minimapTerrainColours` and `minimapOriginalTerrain` pass it their colour source and shade rule.

Kinds and builders: minimap plan: `minimapProducePlan`. Minimap picture: `composeMinimapPlan`, one call from `minimapCompose`. Terrain colours: `minimapTerrain`. Marks: `minimapPlanMarks`, loot still `minimapLootMarks`. Mod terrain rows: `minimapGroupColours`, `Table.ModTerrain`. Player colour (R27, owner ruling): `minimapOwnerColour` and `mapeditorplayertint.go` not merged.

Rules left as they were, now as data: the symbolic outline is clipped to a window with a blocked margin and unclipped on a whole-map window; the installed one is never clipped. The mark order in the plan differs from before (loot, structures, units in both pictures); the composer's three passes make the painted order the same, and the witness records the painted order.

Mod colours. `minimapGroupColours` now fills rows with their own colour without a lookup; a row without its own colour keeps the lookup's colour of its land kind, or stays out with no lookup, which keeps the sampled tile colour. `missionMinimapPaint` gives the mod groups' colours whenever a table has them, and the lookup and owner shades when the install has those; with neither it is nil, as before.

The surface that took the colours is the picker map viewer: `pickerArtSource.resolve` takes them through `pickerMinimapPaint`, and `pickerArt.apply` calls `SetMinimapPaint`. Its paint holds only the group colours, because `flow.enter` gives every viewer the installed crystal, and a paint with the lookup would turn the picker's picture into the installed one. `SetMinimapPaint` rebuilds the symbolic terrain when the group colours change.

Map editor: a search of `pkg` and `cmd` for `mini ?map|overview|miniature|MiniMapData` (case-insensitive, `*.go`) finds no file with `editor` or `mapedit` in its path, and no match in `pkg/editorapp` or `cmd/againrom`. `Viewer.drawFrame` returns on `editorView` before the minimap (`pkg/ui/viewer.go`). The editor draws no minimap, so there is no editor surface for the colours; the debt line in `docs/1573/story.md` names a surface that does not exist. Editor test play starts the game program with the test mod (`pkg/game/mapedittestplay.go`, `TestPlayArgs`); a mission there is a mission viewer dressed by `viewerArt`, whose `missionMinimapPaint` reads the mod terrain rows of `Table`. This is read from the code; no test play was run.

## Proof

Witness: `pkg/ui/minimapplan_witness_test.go` with `pkg/ui/testdata/minimap_plan.golden`, 500 lines of numbers only, from synthetic inputs, recorded at the base before any production change (commit `746565547` holds the base form of the test). Inputs: five viewer dressings (none, crystal alone, lookup without crystal, paint with crystal, paint with no lookup under the crystal), ten maps below, at and above one pixel per cell with and without a blocked margin, ten fog and option states (no plane, each fog state, mixed three-state plane, `minimapHeld`, the loot option, fog reveal, a camera at the far corner, level plane with player colours and local mark colour), and every mark kind (unit over each owner, corpse stage and life; structure of a fixed class, variable class, unknown class, no owner; sack with a good, nil and missing frame). Each line holds the geometry, the content hash, the framed picture hash and every mark's rectangle and colour in the composer's paint order. After the producer became one the golden is byte equal. `TestMinimapPlanWitnessFailsAPlantedFault` checks the comparison; three planted production faults, each run once and reverted, fail the witness: the symbolic outline unclipped, the installed unit extent zero, the window filter applied to sized marks.

Mod colours: `TestAModGroupColourReachesAViewerWithNoPaintData` (`pkg/ui`, pixels of the viewer with no paint, with and without crystal, loss control with no paint) and `TestModGroupColoursReachAViewerWithNoPaintData` (`pkg/game`, the paint and the picker dressing, loss control with no mod terrain).

Guard: `internal/archtest/minimapplan.go`, `TestMinimapHasOnePlanProducer`: one `minimapPlan` composite literal in `pkg/ui` production code, in `minimapProducePlan`, and one call of `composeMinimapPlan`, in `minimapCompose`.

Tests moved to the producer with the same inputs and assertions: `composeMinimap` is now a test helper over `composeMinimapPlan` and the one symbolic fog rule; the signature test checks the new functions as well. No test or assertion left; `pkg/ui/fogwalk_test.go` names the one walk.

## Open debt

- A viewer whose install lacks a lookup and owner palettes, playing a mod with a row without a `minimap` colour, keeps that row's sampled tile colour (brief).
- Installed release evidence for mod terrain on a picker map: Unknown; the two focused tests are synthetic.
