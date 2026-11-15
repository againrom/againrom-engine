# The mod window replaces install files per language

## Result

The mod window has a Files page. It lists the install's pictures, sounds and texts, narrowed by a filter, and marks
each path the mod replaces in the common folder, in English and in Russian. A modder picks a file to replace a path
in one of those folders; a file the game would refuse at play is refused on the page with the reason, and a
replacement can be removed. Every change is undone, redone and saved with the mod's other edits. The page shows
which replacement wins when two enabled mods replace one path, and the published mod plays with its replacements.

## Authority

- Owner card n-modwb-11: a Files page that lists the install's paths, imports a replacement per language checked
  by the install-file rules, and shows what the mod overrides.
- Owner plan of the mod workbench, row 11, decisions D1 to D7 on the recommended answer.
- Owner ruling: a mod shadows install files by path; replacements live in the fixed tree under files.

## Rules

1. The Files page lists the install archives' and sound archive's pictures, sounds and texts by a filter, 40 at once.
2. Each listed path has an import per language folder, common, en and ru; a path the mod replaces names its file.
3. An import is held to the folder's rules: its kind, the 16 MiB bound, UTF-8 text and a language folder.
4. An import for common or the install's language is held to the rule the game serves it by; a refusal names it.
5. An import for another language is held to the folder's rules; that language's install checks it at play.
6. A replacement is removed on the page; a second import of one path asks before it replaces.
7. Every import and removal is one draft step Undo and Redo walk, also past a save; Ctrl+S writes it under files.
8. The page's override lines are the mod check's for the mods the starter enables; a path two mods replace names both.
9. Publish packs the replacements; the packed mod passes the check and the file layer serves each replacement.
10. The starter and the editor read the starter's enabled mods through one reader.

## Code

| kind | its one builder | change |
|---|---|---|
| W page | `pkg/ui/mapeditorworkspace.go` `modScopePages` | the files page is a row, scope `installfiles` |
| workspace fields | `pkg/game/mapinspectionworkspace.go` | scope `installfiles` routed to `mapeditormodfiles.go` |
| install file rule | `pkg/mod/files.go` `installFileAt` | `DraftInstallFile`, `InstallFileKind`, `InstallLanguages` for the page |
| install file serving | `pkg/game/modfiles.go` `serveInstallFiles` | `serveModFiles` serves the context and the override lines, with the sound archive |
| install listing | `pkg/vfs` `Entries` | the archives' and the sound archive's entries of a mod file kind |
| mod file import | `pkg/game/modassetimport.go` `importModFile` | `hostFileBytes` takes the bound of the file's kind |
| editor mod document | `pkg/game/mapeditormod.go` `document` | the snapshot holds every file under files, the strips among them |
| starter settings | `pkg/launch/starterini.go` | `EnabledMods`, `StarterEnabled`; the editor's host asks it |

## Proof

| test | rules |
|---|---|
| `TestFilesPageImportsByTheRulesTheGameServesBy` | 1, 2, 3, 4, 6 |
| `TestFilesPagePlacesByLanguageAndUndoesPastASave` | 2, 5, 6, 7 |
| `TestFilesPageOverridesAreTheChecksLines` | 8 |
| `TestFilesPageModSavesPublishesAndServes` | 7, 9 |
| `pkg/ui` `TestModWindowShowsTheFilesPageOfItsHost` | 1 |
| `pkg/launch` `TestModsFolderHasOneResolver` | 10 |

Release tests at the release: the `TestReleaseModInstallFiles*` set on rom1-en and rom1-ru from an editor-made mod
(owed, not run).

## Open debt

- After an Undo past a save, the draft context and the override lines serve the folder's file until the next save.
- The override lines are made once per context; a change of the starter's list shows when the context is built again.
- The page shows the install-file overrides; the data-kind override lines of the check are not on it.
