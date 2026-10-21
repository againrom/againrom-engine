# Install files in a mod

## Intent

A mod replaces an install's picture, sound or text by carrying a file at the same install path,
with one text per language. The game reads the mod's file wherever it reads the install's. A save
made with a mod that only replaces pictures and sounds loads without that mod.

## Authority

Owner direction T5 of the mod track (`pipeline/BACKLOG.md`): install files in a mod.

## As built

### Folder layout and language rule

`pkg/mod/files.go` reads `<mod>/files/<folder>/<archive>/<path>`. The folders are `common`, `en`
and `ru`. `ReadInstallFiles` holds every folder to the folder rules and returns, for the install's language
(`modrt.LanguageFor`), that folder's files plus the `common` files whose path the language folder
lacks. Kinds: `.bmp` picture, `.wav` sound, `.txt` UTF-8 text; each at most 16 MiB. A file outside
a language folder, an unknown folder, a file naming no archive, another extension, a non-UTF-8 text
and a non-regular file are refused naming the mod and the file. `modrt.Load` reads them before the
script runs and returns them as `Result.Files` with the mods' precedence.

### One boundary

`pkg/game/modfiles.go` holds `modFiles`: one fork of the install's containers (and of the sound
bank's), the precedence, and the owner of each served path. `Archives.mods` keeps it, so every owner
serves through the same copy:

- `moditems.go` (item icons and figure layers), `modbodies.go` (the body choice file) and
  `modmission.go` (mission maps and texts) call `modFileLayer()` and `serve`/`serveBytes` in place
  of their own `Fork`/`Overlay`.
- Install files are served by `NewModFrontEnd(root, files)` into a fresh, uncached install share
  before any eager decode (menu art, statics, units, sounds, words). The text readers
  (`installtext`, `questobjectives`, `secondgametext`) read the served bytes through the archives
  unchanged. `NewFrontEnd(root)` is `NewModFrontEnd(root, mod.FileData{})` and keeps the shared,
  cached install.

`serve` decides a path served twice: one mod keeps its later file; of two mods the one that loads
after the other (`requires`, `load-after`) wins and the override is recorded
(`FrontEnd.ModFileOverrides`); without an order the launch is refused naming both. A composed file
(the body choice) carries no mod and is not checked.

Before a file is served: the install must hold the path; a picture must decode with the install's
picture's bit depth (8 or 24) at its size; a sound must decode as WAV; a text must hold only
characters the install's alphabet writes (`ModTextDropped`). A text is served as the install's
bytes (`EncodeInstallTextFile`), and on a converting second-game edition as the Windows code page
the reader converts back (`TextCode.FileBytes`, `vfs.DOSCyrillicToWindows`).

### Mod mark

`mod.SetEntry.StateFree` is set by `modrt` for a mod whose script set no rule and added no data
file and which serves no text file (`mod.FilesChangeState`; the install files that apply on the
install's language count). A text feeds the definitions: the body list (`HeroPictureAddress`) sets
the hero's class in every first-game mission start World hash and in the SAV. A picture or a sound
keeps a mod state-free; the readers that decide this are listed under "Picture and sound readers". `mod.Set.Marked()` drops those mods; `currentsave.go` writes the mark from it and
`modmark.go` judges a loaded save's mark against it. The flag is not in the canonical form, so a set
digest does not move. A mod with a rule or a data file is marked and refused as before.

`applyModMark` compares a saved mark with the active marked mods plus the active state-free mods
the mark names, so a save written under a mod that is state-free now loads while that mod is
active; a mark naming a mod that is not active still refuses. A language folder is read in any
case (`files/EN`); two spellings of one folder in one mod are refused naming both.

### Picture and sound readers

Searched: every non-test call of `bmp.Decode*`, `audio.DecodeWAV`, `audio.DecodeTrackWAV` and
`wav.Parse` in `pkg` and `cmd`, and every reader of the archives that reaches `pkg/sim`,
`pkg/mapload`, `pkg/data` and `pkg/formats/sav`. None of those packages reads an install `.bmp` or
`.wav`. The readers, all presentation or input:

- `pkg/game/portrait.go` `loadPortrait` (info window and structure portraits): drawn only.
- `pkg/game/screenart.go` `readInstallBMP`, `readInstallMask` (screen art and its 8-bit hit
  masks): drawn and hover or click regions; no command is created from the mask data.
- `pkg/game/spellicon.go` `loadSpellAtlas` (spell book icons): drawn only.
- `pkg/render/terrain/tileset.go` `LoadTileset` (terrain strips): drawn only; passability comes from
  the cost table and the map, not the pictures.
- `pkg/game/sound.go`, `pkg/game/speech.go`, `pkg/game/music.go` (`DecodeWAV`, `DecodeTrackWAV`):
  played only; no sample length reaches the simulation. `music.res` is not served.
- `pkg/game/modfiles.go`: the decode that checks a served file before it is served.

Verdict: no reader on the World-hash or SAV path reads an install picture or sound, so these
two kinds keep a mod state-free. Existence checks of sprite sheets (`.256`) in `modbodies.go` are
not served kinds.

### Launcher

`cmd/againrom/main.go` builds the front end with `modRun.Files`, prints
`againrom: install files=<n> state-free=<ids>` and one `againrom: override:` line per override.

## Proof

| proof | test | install | loss control |
|---|---|---|---|
| picture in the headless frame, title and briefing per language, the mod's sound, SAV and World equal without state | `TestReleaseModInstallFilesDrawPlayAndPrintTheModsFiles` | EN, RU | two sessions without a mod compared first; the diff must name exactly the six changed parts |
| picture-mod SAV loads without the mod; a rule mod's SAV is marked and refused | `TestReleaseModInstallFilesSaveLoadsWithoutAStateFreeMod` | EN, RU | the rule mod is the same pictures and sound plus one rule |
| a mod serving only `main/text/heropicture.txt` changes the hero's class, is in the mark, its SAV is refused without the mod and loads with it | `TestReleaseModInstallFilesTextModIsMarkedAndItsSaveNeedsTheMod` | EN, RU | with text counted state-free again the test fails at "is state-free true" on both |
| a mark naming a now state-free mod loads while that mod is active, refuses when it is not | `TestMarkNamingAStateFreeModLoadsWhileItIsActive` | none | the not-active case asserted |
| language folder in any case; two spellings refused naming both | `pkg/mod/files_test.go` | none | refusal asserted on a map file system |
| two mods on one path: unordered refused, ordered later wins and the override is named | `TestReleaseModInstallFilesOfTwoModsFollowTheLoadOrder` | EN, RU, ROM2-EN, ROM2-RU | the refusal and the winner are both asserted |
| second game: menu picture, home title, mission 10 briefing, sound | `TestReleaseSecondModInstallFilesDrawPlayAndPrintTheModsFiles` | ROM2-EN, ROM2-RU | the row without the mod must show none of them |
| folder rules, language selection, mark set | `pkg/mod/files_test.go`, `pkg/modrt/files_test.go` | none | refusal texts asserted |
| one path in mod order, picture check, text bytes, mark under a state-free mod | `pkg/game/modfiles_test.go` | none | each refusal asserted |
| `-check` counts, refusals and overrides | `TestCheckTakesAModsInstallFilesAndRefusesWhatTheGameCannotUse` | synthetic | exit codes and stderr asserted |

The fixture mod is generated in each test from the install it runs on; no install byte is in the
repository. Existing mods' release tests run unchanged in the release gate.

## Divergences

None. Install files are an Againrom mod facility; with no mod the game reads the install as before.

## Open debt

- Not served: `speech.res`, `music.res`, loose files beside the archives, and pictures in `.256`,
  `.16a`, `.16`, `.pal`, `.reg`, `.alm` form.
- A text is checked against the install's alphabet, not against the glyphs of the font that draws
  it.
- State-free is decided by the script and the kinds of file served: a mod that adds a data file
  (the base dump's towns file) or serves a text stays marked even when that file changes nothing.
- The rule is by kind, not by path: a text that no definition reads (`npcnames.txt` left the World
  hash and the SAV unchanged in review) still marks the mod.
