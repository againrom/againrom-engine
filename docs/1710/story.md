# The mod window's Companions page edits companion join conditions

## Result

The mod window has a Companions page beside Towns and Bodies. For each chapter whose town grants a companion, a modder
adds, changes and removes a row: the chapter, the companion and the tavern conversation that makes the companion join.
An empty page shows one button, Add a companion. Every edit is one undo step. Apply writes the mod's companions file.
Save mod refuses a row the campaign cannot run, naming it. No game rule, hash or save byte changes.

## Authority

- Owner: the mod workbench epic, plan story 13; no description text stands in a window, an explanation is the field's
  hint; the editor is Againrom's own and has no divergence.

## Rules

1. When the mod holds no companions file, the page shows only the Add a companion button and nothing is written.
2. A row added takes the first companion a town grants that has no row, and its chapter's first tavern conversation.
3. When a row is added or changed, the companions file reads back as the rows set.
4. When a chapter is changed, a companion or conversation the new chapter lacks gives way to its first.
5. When a change would give one companion of one chapter two rows, the edit is refused naming the row.
6. When an edit is undone, the project returns to what it was before the edit.
7. Save mod refuses a row naming a chapter, companion or conversation the campaign lacks, naming the row.
8. A file the page made loads under the release test on rom1-en and rom1-ru.

## Code

| kind | its one builder | change |
|---|---|---|
| companions writer | `pkg/mod/kinds.go` dataKinds, `writeRows` | a write column and the held-draft mark |
| mod window page | `pkg/game/mapinspectionworkspace.go` workspaceScopeRows, `pkg/ui` modScopePages | one row each |
| page fields | `pkg/game/mapeditormodcompanions.go` | fields and setter on the kind step |
| list page layout | `pkg/ui/mapeditormod.go` modListPages | Companions joins |

## Proof

| test | rules |
|---|---|
| `TestCompanionsWriterRoundTripsRowsThroughItsReader` | 1, 3 |
| `TestModCompanionsPageEditsJoinRows` | 1, 2, 3, 5, 6 |
| `TestModCompanionsPageOffersCampaignChoicesAndGatesSave` | 4, 7 |
| `TestReleaseModJoinWaitsForTheTavernTalk`, lane run, rom1-en and rom1-ru | 8 |

The release test ran over a mod whose companions file the page made.

## Open debt

- The page offers the tavern only, as the kind does; a shop or school conversation cannot start a join.
