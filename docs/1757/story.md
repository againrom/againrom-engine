# A mod's global variables are declared and used in the editor

## Result

The map editor's mod window now has a Globals page for adding, renaming, noting and removing global variables.
Each global shows its name, number, starting value, note and the maps and replies that use it.
The trigger wizard and the Dialog window's replies can compare, set and add to a global, and show it by name.
A choice in the global slot adds a new global on the spot. A global in use cannot be removed.
The page is saved with the mod like every other page.

## Authority

- Owner: «a mod's global variables are declared and used in the editor»; the editor shows names, not numbers.
- Owner: a number is given once and never again while a map names it.
- `DIV-5108`: the globals, their file, the three nodes and the extended map they need.

## Rules

1. When the mod window opens, the Globals page lists New and each global's name, number, start, note and users.
2. When New is pressed, the global takes the first number no declaration and no map of the mod names.
3. When a name or value is bad, repeated or out of range, the edit is refused and writes nothing.
4. When a map node or reply uses a global, even a node no trigger names, Remove is refused naming the map and the place.
5. When a global is unused, Remove takes it out; every edit is one undo step.
6. When the mod is saved, the globals file holds the page's rows, equal to the hand-written mill file.
7. When a trigger or reply row is added in a mod, it offers a global compared, set and added to; outside a mod, none.
8. When the global slot is opened, it lists the mod's names and a choice that adds a global and fills the slot.
9. When a global is renamed, its number stays, so every map's rows still play.
10. When the first global action is applied on a plain map, the extended map question is asked first.

## Code

| kind | its one builder | change |
|---|---|---|
| mod window page | `pkg/ui/mapeditorworkspace.go` modScopePages | a Globals row, with its game row in `workspaceScopes` |
| page edit step | `pkg/game/mapeditormodterrain.go` kindFiles.editRows | globals edits through the kind's reader and writer |
| held file kind | `pkg/mod/kinds.go` dataKinds | globals kind drafted whole; draft reader `draftGlobals` |
| wizard item | `pkg/ui/mapeditorwizard.go` wizConditions, wizActions | three rows, one global parameter |
| script parameter kind | `pkg/game/mapinspectionscriptauthor.go` scriptParameterKind | a global kind for slot 0 |
| script target list | `pkg/game/mapinspectionscriptchoices.go` scriptTargets | the mod's globals and the new one |
| what names a global | `pkg/game/mapcheckglobals.go` scriptGlobalUses | the map check and the page's users share it |
| choice that adds a record | `pkg/ui/mapeditorobject.go` chooseObjectChoice | the new global choice beside the new message |

## Proof

| test | rules |
|---|---|
| `TestModGlobalsPageBuildsTheMillFile` | 1, 3, 5, 6 |
| `TestModGlobalsPageRefusesRemovingAUsedGlobal` | 1, 2, 4, 5 |
| `TestGlobalsDeclaredInTheEditorPlayAfterARename` | 6, 7, 8, 9 |
| `TestGlobalActionAsksToExtendThePlainMap` | 8, 10 |
| `TestTriggerWizardItemListsOpenWholeOnTheSmallestScreen` | 7 |
| `TestModGlobalsPageCountsNodesNoTriggerNames` | 2, 4 |
| `TestTriggerWizardOffersNoGlobalRowOutsideAMod` | 7 |

Release tests at the release: none; the plays run on the fixture front in EN and RU words.

## Open debt

- The second game's editor has no globals; a global in a second-game map is not offered.
- A global's number is not changed on the page; a rename keeps it.
- The slot's new-global choice adds the global in place; it does not open the page, which would drop the wizard's rows.
