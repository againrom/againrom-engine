# The mod editor reads its drafts through one view and routes its pages through one table

## Result

Nothing a player sees changes. The mod editor now reads every unsaved change of a mod through one view,
which treats two spellings of a name that differ only in letter case as the same file. The speakers, terrain,
strings, item sprites, install files and test play all read through that view. The mod window's own pages are one
table on the game side, so a new page is one row there and one row in the window.

Each kind of file the editor drafts is one row of the mod data kinds table, and the editor's open, run context,
save and page edits read the kind from that row. One small change a modder can see: a speaker picture changed
alone now shows in the run context at once, where before it waited for the next rebuild.

## Authority

- Owner, on the architecture audit's rows 1, 3 and 5: «Как рекомендовано»; BACKLOG items N44, N64, N63.

## Rules

1. When a draft is keyed in another letter case, every reader of the drafts reads the same bytes and names.
2. A draft that deletes a file removes the install file it replaces; a draft under files adds or replaces one.
3. A page scope the mod window shows is one row of the game's scope table; its fields and its edits go through it.
4. Each mod page scope stands in the window's page rows and the game's scope table, and the game names it nowhere else.
5. Every page, draft, rebuild and save behaves as before, but for rule 7: the focused sets pass unchanged.
6. A drafted kind is one data kinds row that open, rebuild, save drop, overlay, page edit read; speakers: Open debt.
7. A draft of a kind the run reads, a speaker picture too, rebuilds the context; strings only when cited.

## Code

| kind | its one builder | change |
|---|---|---|
| draft overlay | `pkg/game/mapeditormod.go` `projectView` | one folded map; `Draft` and `Drafted` answer by it |
| install file overlay | `pkg/mod/files.go` `OverlayInstallFiles` | takes the view; keeps only the language-layer rule |
| speakers read | `pkg/mod/kindread.go` `ReadSpeakers` | asks pictures of a lookup, the view's `Draft` |
| run context draft | `pkg/game/mapeditormodcontext.go` | terrain, items and their strings read through the view |
| page scopes | `pkg/game/mapinspectionworkspace.go` `workspaceScopes` | `WorkspaceFields` and `SetWorkspaceField` read rows |
| drafted kinds | `pkg/mod/kinds.go` `dataKinds` | `Draft` column; `draft`, `lookup` readers in `draftkinds.go` |
| page edit | `pkg/game/mapeditormodterrain.go` `editKind` | one undo step; held files only; `modStripFile` names a strip |
| guards | `internal/archtest` `draftview.go`, `scoperows.go`, `draftkinds.go` | second routes, scope rows, kind rows |

## Proof

| test | rules |
|---|---|
| `TestDraftKeyedInAnotherCaseReadsTheSameThroughEveryOverlay` | 1, 2 |
| `internal/archtest` `TestDraftReadersReachTheDraftsThroughProjectView`, `TestDraftViewSecondRoutesAreFound` | 1 |
| `TestWorkspaceScopeAddedAsARowShowsAndSetsItsFields` | 3 |
| `internal/archtest` `TestModWindowPageScopesAreOneRowOnEachSide`, `TestModWindowPageScopeBranchesAreFound` | 4 |
| `TestEveryDraftedKindRowIsReadByEachReader` | 6, 7 |
| `internal/archtest` `TestDraftedKindIsOneDataKindsRow`, `TestDraftedKindBranchesAreFound` | 6 |
| `pkg/game`, `pkg/mod` focused set `Mod Draft Workspace Terrain Texts Items InstallFile Speaker` | 2, 5 |
| `pkg/ui` focused set `Mod Workspace` | 5 |

Release tests at the release: the `TestReleaseMod*` set on rom1-en and rom1-ru (owed, not run).

## Open debt

- `document()` keeps its own speakers load and the speakers-picture drop, from before this story; the speakers
  row is not read through them.

Evidence: `review/story1668-mod-kind-rows/RESULT.md`.
