# Mod screens and the screen registry

## Intent and authority

A mod declares screens in data and the game shows them: an entry in the main menu or the in-game menu opens an info page, a list or a table of rows, in the game's own frame, font and colours. Authority is owner direction (`review/mod-milestone/PROPOSAL.md`, "UI extension and new screens" and item 11; `review/mod-milestone/MOD-LAYOUT.md`), built on `docs/1267/story.md`, `docs/1272/story.md` and `docs/1273/story.md`. A mod ships no screen code. ROM1 differences are rows `DIV-1894` to `DIV-1897` in `docs/divergences/mods.md`. `DIV-1898` and `DIV-1899` are returned unused. Without a mod every menu frame, screen, hash, SAV byte, scenario and release witness is unchanged.

## As built

Declaration form. One form: `data/screens.toml`, loaded by `game.data.add("data/screens.toml")` in the mod's `init`, the route items already use (`pkg/modrt/data.go`, `pkg/mod/screens.go`). `[[screen]]` tables:

| key | meaning |
|---|---|
| `key` | identifier, `a-z0-9_`, unique in the file |
| `kind` | `info`, `list` or `table` |
| `title`, `menu` | text keys: the page heading and the menu entry's label |
| `place` | `main`, `game` or `both` |
| `text`, `items`, `rows` | the body of the kind: paragraphs, bulleted lines, or pairs `["label key", "value key"]` |

Every text is a key of `text/<lang>/strings.toml`, resolved in the language of the base with the English fallback (`DIV-1865`). Refusals name the mod, file and line (`mod "m": data/screens.toml:7: ...`): unknown kind, key, place or table, a body key of another kind, a kind with no body, a missing text key, a text outside its length bounds, a duplicate key, and an entry that does not fit a menu (3 main menu slots and 2 in-game menu slots, counted across all mods in load order). A screen file loads once. `-check` prints `againrom: screens=N main-menu=M game-menu=G` when a mod declares screens. The screens reach the application through `ui.App.SetModScreens(game.ModScreens(res.Screens))`; `cmd/againrom` calls it after `front.App`, so the front end gained no field and the archtest ratchets did not move.

Registry (`pkg/ui/screenhandler.go`). `ScreenHandler{Step, Compose, Draw}` is keyed by `Screen`. `App.step` and `composeScreen` look a screen up first, and `Draw` does too; a nil member falls back to the switch the screen was not moved out of. Migrated into handlers: the three dispatches of the main menu, the picker, the town, the in-game menu, the load window, the save dialog, the documents panel and the ending; Compose and Draw of the cutscene library, the credits and character generation; Step of the new `ScreenMod`. Not migrated: Step of the map screen (its arm is the mission input path) and of character generation, and the pre-switch routes of the cutscene library and the credits (`stepMedia`) and of the cutscene overlay. `screenRegistry` keeps its hand-written census metadata (composer names and witnesses); `TestScreenHandlersAgreeWithTheCensusRegistry` ties it to the handlers (a composer is named exactly where a Compose handler exists). The registry is a package table of `pkg/ui`, not a `FrontEnd` field.

`ScreenMod` is declared outside the const block of the shipped screens, so `cmd/screencensus` output is unchanged; a mod screen is registered by its handler and tested in `pkg/ui/modscreens_test.go`.

Menus. The main menu draws each entry as a town shell text box stacked up from the bottom left corner after the brooch and the version label; a press and release on it opens the screen (`DIV-1895`). The in-game menu root appends one row per entry after the shipped rows, with the action `gameMenuModScreen`; the town panel grows downward to hold them, the mission panel already has the room (`DIV-1896`). Both use the install font through the existing row and label paths.

Page (`pkg/ui/modscreens.go`). Menu background, the install's large panel, centred title, body lines in the install font and the shell letter colour, a scroll bar when the body is longer than the page, and a control captioned with the install's OK word. Up, Down and the wheel scroll; the control, Enter and Escape return to the opener (`DIV-1897`). From the in-game menu over a mission the screen holds the world as the menu does.

Example mod (`pkg/modrt/testdata/mods/heavy-armor`, test data): `data/screens.toml` declares the info screen "About heavy armour" for both menus, with English and Russian text. The seat copies the mod into `builds/current/mods/`.

## Proof

- Unit: `pkg/mod` (every kind, each refusal with file and line, the slot bounds), `pkg/modrt` (the example mod in both languages, refusals naming mod, file and line, slots shared across mods, one load per file), `pkg/ui` (handlers against the census, the unmodded menu frame equals the brooch and label with a loss control, entry hover, press and release, Back, Escape and Enter, a press dragged off the entry, game menu rows and window, the grown town panel and the decoded panels without mod rows, scrolling, wrapping, refusals), `cmd/againrom` (`-check` line, refusals).
- Release, EN and RU, one root at a time (`pkg/game/modscreens_release_test.go`, gated by `AGAINROM_ASSETS`):
  - The unmodded main menu frame equals the install's brooch bitmap pixel for pixel; the modded frame differs only in the bottom left corner and carries the entry label drawn in the install font (template match of the font's own glyph pixels); the label is absent from the unmodded frame (loss control).
  - The entry opens the screen by the production pointer route; the title and the first words of every paragraph are matched as install font glyph pixels, and absent from the menu frame; Back and Escape return to a menu frame byte-equal to the one before.
  - Over a mission, the in-game menu has 8 rows against 7 unmodded, the shipped rows equal, the panel differs only on the eighth row's rectangle, and its label is drawn in the install font. The row opens the screen with the world hash unchanged over 40 steps and on return; with the menu closed the same steps change the hash (loss control).
  - Offscreen screenshots, never skipped, written to `AGAINROM_SHOT_DIR` or a temporary directory: `review/story1277-mod-screens/menu-with-entry-<root>.png`, `mod-screen-<root>.png`, `game-menu-with-entry-<root>.png`.

## Open debt

- Menu entries open a page only; a screen with controls, a town room from data, and a key shortcut for an entry are not built.
- The main menu entry box is drawn over whatever the brooch art has in the bottom left corner; a base whose menu art differs was not drawn.
- `screenRegistry` metadata is still hand-written beside the handlers; it is not generated from them.
- Step of the map screen and of character generation, and the media pre-switch routes, stay in `App.step`.
- The town in-game menu with a mod row is covered by unit tests only; the release witness drives the mission menu.

## Gates

- `go test -trimpath -count=1 -p 2 ./...` passes, including the archtest field cap and composition ratchets (`FrontEnd` unchanged; `flow` gained the single field `modUI`, pinned in `flow_test.go`).
- `pipeline/check-release-tests.sh` on the EN and RU roots with 8 shards: 690 gated tests, both roots fail only in `TestReleaseCurrentActionBookSAV` and `TestReleaseCurrentObjectSAVContinuation` (four subtests), which read the untracked `engine/saves/*.ags` files absent from the lane worktree. The three new release tests pass on both roots.
- `cmd/screencensus` output against engine main: identical except the two new test lines and the selected count (688 to 690).
- Ratchets: `internal/gatedtests/testdata/population.txt` lists the two new release tests; `internal/storyguard/baseline.go` `CommentBytes` is 8419934, the rise belonging to the new code and tests named in its note.
- `gofmt -l` is empty; `scripts/check-no-game-assets.sh` is clean.
- Version bump of `againrom` (MINOR) is left to the landing.
