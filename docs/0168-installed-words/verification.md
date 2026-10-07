# 0168 — verification

Branch `0168-installed-words`, base `2e662e6`, research pin `7747b9d`.

## The gate

Run in the lane worktree on the committed tree.

```
go build ./...                                                    clean
go vet ./...                                                      clean
gofmt -l $(git ls-files --cached --others --exclude-standard '*.go')   prints nothing
go test -trimpath -count=1 ./...                                  all packages ok
```

```
scripts/check-no-game-assets.sh        PASS
scripts/check-doc-budget.sh            PASS
scripts/check-hotfix-ledger.sh         PASS
scripts/check-sdd-audit-selftest.sh    PASS
scripts/check-sdd-audit.sh             PASS (note: no tasks.md)
```

`git log --format='%h %(trailers:key=Co-Authored-By)' 2e662e6..HEAD` prints the
hash with an empty trailer field: no commit carries one.

The note/warning **count** from `check-sdd-audit.sh` is not comparable from a
worktree, which has no `builds/`. Only the FAIL set is, and it is empty.

## The result someone can point at

Eleven program-chosen words resolve from the install. Measured on both preserved
roots:

```
$env:AGAINROM_ASSETS='<seat>\gameversions\en'
$env:AGAINROM_ASSETS_RU='<seat>\gameversions\ru'
go test -trimpath -count=1 ./pkg/game/ -run TestInstallWordsOverALawfulInstall -v
```

```
en: selector 0, main.txt 274 lines, dialogs.txt 166 lines, 11/11 words resolved
ru: selector 1, main.txt 274 lines, dialogs.txt 166 lines, 11/11 words resolved
words identical across the two roots: 0 of 11
```

The two line counts are an independent agreement with the pin: `TEXT-STRTAB-023`
gives `main.txt` 274 lines from its base arithmetic, and `TEXT-NAMETAB-026`
measures `dialogs.txt` at 166. This reader, walking the payload itself, reports
the same numbers on both roots.

What each word resolves to, read out of both `MAIN.RES` archives with
`cmd/restool`:

| Field | Line | English root | Russian root |
|---|---|---|---|
| notice button | `main.txt` 77 | `Ok` | `Принять` |
| mission won | `main.txt` 140 | `Mission Completed` | `Миссия выполнена` |
| mission lost | `main.txt` 141 | `Mission Failed` | `Миссия провалена` |
| menu save | `dialogs.txt` 34 | `~Save Game` | `~Сохранить игру` |
| menu load | `dialogs.txt` 35 | `~Load Game` | `~Восстановить игру` |
| menu game options | `dialogs.txt` 36 | `Game ~Options` | `~Опции игры` |
| menu sound options | `dialogs.txt` 37 | `Sou~nd Options` | `~Звуковые опции` |
| menu quest objectives | `dialogs.txt` 38 | `~Quest Objectives` | `З~адание` |
| menu end quest | `dialogs.txt` 39 | `~End Quest` | `За~кончить миссию` |
| menu return | `dialogs.txt` 40 | `~Return to Game` | `Вер~нуться к игре` |
| town menu abort | `dialogs.txt` 77 | `Abort Game` | `~Выход из игры` |

Both roots start:

```
$env:AGAINROM_ASSETS='...\gameversions\ru' ; .\againrom.exe -check
againrom: 62 map rows, 8 of 8 buttons have a mask region; hero Body 43, ...
$env:AGAINROM_ASSETS='...\gameversions\en' ; .\againrom.exe -check
againrom: 66 map rows, 8 of 8 buttons have a mask region; hero Body 43, ...
```

**Nothing was watched on a screen.** No windowed game was launched for this
story. What is witnessed is the resolution, the drawing contract and the paint
path under test, not a photograph of the menu. The build and its README are in
`builds/0168-installed-words/`; the visible check is one Escape press on the
Russian root and it has not been performed.

## The script-gap census

```
go build -o /tmp/mr ./cmd/missionrun
AGAINROM_ASSETS=<en root> /tmp/mr -mission 10 -trace -ticks 1 | grep -c UNSUPPORTED   13
AGAINROM_ASSETS=<en root> /tmp/mr -mission 20 -trace -ticks 1 | grep -c UNSUPPORTED   11
```

`pipeline/milestone-baseline.txt` carries `en m10 cannot run 13 x instant op 2`
and `en m20 cannot run 11 x instant op 2`. Both are **unchanged**, which is the
expected result: this story adds no script instant. Its result is the visible one
above, not a falling census number.

## FR

| id | Witness |
|---|---|
| FR-1 | `TestSplitTextTableIsTheDecodedWalk` — CRLF, the skipped byte, the unterminated tail, and high bytes returned unchanged |
| FR-2 | `TestTextTableAbsent` — four absent shapes and one present line, plus the nil table |
| FR-3 | `TestInstallWordsWithoutTablesAreAuthored` (four sources, including nil); `NewFrontEnd` reads it once beside the language selector, `pkg/game/frontend.go` |
| FR-4 | `TestInstallWordsResolveEveryDecodedIndex`, `TestInstallWordsResolvePerIndex`, `TestWordsReachEveryViewerAndTheMenu` |
| FR-5 | `TestOutcomeTextFollowsTheWordSet`, `TestMissionOutcomeTextComesOffTheViewer` |
| FR-6 | `TestSetWordsRewritesOnlyTheButtonWord`, `TestAuthoredLayoutsStateTheAuthoredButtonWord`, `TestTownDialogueButtonComesFromTheWordSet` |
| FR-7 | `TestGameMenuRowsComeFromTheWordSet` — both surfaces, both sources, in screen order |
| FR-8 | `TestAcceleratorOverAnInstalledLabel` — a CP866-marked label yields that byte and column 1, and `lowerASCII` leaves it alone |
| FR-9 | `spec.md`'s category-(c) table; the *What stays English* section below; `builds/0168-installed-words/README.md` |
| FR-10 | `TestChargenTextRowsRetainRawBytes` and `TestLoadChargenAssets` over the CRLF fixture — the generator resolves `Character name:`, `Accept`, `Reset`, `Back` unchanged |

## AC and P

| id | Result |
|---|---|
| AC-1 | PASS. `A\r\nB\r\n` → `A`, `B`; `A\rXB\r\n` → `A`, `B` |
| AC-2 | PASS. `80 9f a0 ff` returned byte for byte |
| AC-3 | PASS. −1, 3, 400 and the empty line 1 all absent; line 2 present |
| AC-4 | PASS after a spec correction made at Phase 5. The original wording added "and the front end constructs", which is false of a source stating no `main.txt`: `LoadChargenAssets` requires eleven slots of its own and refuses without them, and that rule predates this story. `spec.md` AC-4 now states the word set alone, which is what is witnessed: a source stating neither table resolves to `AuthoredWords()` exactly |
| AC-5 | PASS. Fixture with 140/141 gives them; fixture without gives `MISSION COMPLETE` / `MISSION FAILED` |
| AC-6 | PASS, both layouts, and every other field of both layouts compared unchanged |
| AC-7 | PASS. Mission 7 rows and town 5 rows, authored and installed |
| AC-8 | PASS. `TestInstallWordsResolvePerIndex` and the *per row* subtest |
| AC-9 | PASS. Accelerator `0xa0`, column 1, drawn label `87 a0 a4`; no ASCII letter equals `0xa0` |
| AC-10 | PASS with a **changed fixture**. The generator's strings are unchanged, but the fixture is not: it joined rows with a single NUL and now terminates each with CRLF (plan DD-2). The old fixture was readable only by the splitter this story deletes, so "the same fixture" was not available; the shipped file is CRLF |
| AC-11 | PASS for the panel: `TestAuthoredPanelCaptionsAreUnchanged` pins all 32 captions in order. The other category-(c) surfaces are unchanged by inspection of the diff — no file under `pkg/game/shop*.go`, `pkg/game/townscreen.go` (beyond the button word), `pkg/ui/readout.go`, `pkg/ui/hudtoggles.go` was touched |
| AC-12 | PASS. 11/11 on both roots, 0/11 identical between them |
| P-1 | PASS. `LoadInstallWords` is called once, in `NewFrontEnd`. No draw path reads an archive: `paintGameMenu`, `SetWords` and `OutcomeText` take values already resolved |
| P-2 | PASS. `git diff 2e662e6..HEAD -- pkg/sim` is empty; no constant named `formatVersion` or any serialized field is touched |

## DD and SC

| id | Where it landed |
|---|---|
| DD-1 | `pkg/game/installtext.go` holds the only splitter in the package; `chargenTextRows` is deleted and `chargenTextAt` takes a `*TextTable` |
| DD-2 | `SplitTextTable` is the cursor walk; the discriminating case is the second subtest of `TestSplitTextTableIsTheDecodedWalk` |
| DD-3 | `Global` and `Dialogs` are separate methods; `TestGlobalAndDialogsAreDifferentIndexSpaces` shows the same number reaching two different lines and `Global(274)` refused |
| DD-4 | `ui.Words` is a struct; `TestWordsHoldsExactlyTheDecodedEleven` fails on a twelfth field and on an empty authored value |
| DD-5 | `TestInstallWordsWithoutTablesAreAuthored`; no error path exists in `LoadInstallWords` |
| DD-6 | `pkg/game/frontend.go`, the `words :=` line above the language selector |
| DD-7 | `App.SetWords` → `flow.words`; `flow.enter`'s `v.SetWords(f.words)`; `TestWordsReachEveryViewerAndTheMenu` covers a map opened after the call and a viewer already open when it lands |
| DD-8 | `missionOutcomeText(v, o)` reads `v.Words()`; `pkg/game`'s `MissionWonText` and `MissionLostText` are aliases of the `ui` values and the test asserts they have not drifted |
| DD-9 | `gameMenuRow` is unchanged; the accelerator helpers were not modified, and FR-8's witness is a test over them |
| DD-10 | `AuthoredWords` keeps the upper-case labels; `TestGameMenuRowsComeFromTheWordSet`'s two subtests show both cases; the README states the visible consequence |
| DD-11 | `TestInstallWordsOverALawfulInstall`, skipping on an absent `AGAINROM_ASSETS` and logging rather than assuming a second root |
| DD-12 | Every fixture is built in test code; the one CP866 string is written as `[]byte{0x87, '~', 0xa0, 0xa4}` |
| DD-13 | `pkg/sim` untouched (P-2) |
| DD-14 | This file's *What stays English*, `spec.md`'s category-(c) table, and the README's own paragraph |
| SC-1 | No category-(c) word was bound to an index |
| SC-2 | No `Diplomacy` row was added; `dialogs.txt` 76 is read by nothing |
| SC-3 | `lowerASCII` is unchanged and the test asserts it leaves `0xa0` alone |
| SC-4 | No index in 85..89, 129, 142..149, 204..209 or 221..226 is read |

## Beyond spec: the menu's font

One change is in the plan's shape but was not in the spec's requirement list, and
it is stated here rather than left to be found.

`paintGameMenu` drew its rows with `ebitenutil.DebugPrintAt`, which is an ASCII
font. Resolving a Russian label into that path would have produced a row of
replacement glyphs — the words would have moved with the install and nobody
could have read them. `paintGameMenu` now takes a `*text.Font` and draws with it
when one is supplied; the front end supplies the font it already loads, which
carries the install's language selector, and that is where CP866 is converted
(`TEXT-DOM-010`). A nil font keeps the debug-font path exactly as before, which
is what `cmd/mapview` and every test flow use.

Two consequences are visible on the English root and are not defects: the rows
are drawn in the game's own font rather than the debug font, and the accelerator
underline is measured with that font's advances rather than a fixed cell of 6.

`TestThePanelPaintsInsideItsOwnRect` now runs both paths, including a row whose
label is CP866, one with an empty label and one with an escaped `~~`.

## What stays English, in the player's units

On a Russian install these are Russian: the in-game menu's rows, the button on
the dialogue box and on the outcome box, and the two sentences the outcome box
states.

These stay English on both installs:

| Surface | Distinct strings | Counted by |
|---|---|---|
| Unit information panel captions | 32 | `TestAuthoredPanelCaptionsAreUnchanged` |
| Character skill and school names | 11 | `CharacterSkillName`: `General` plus two banks of five |
| Panel always-hits mark | 1 | `panelAlwaysHits` |
| Developer readout captions | 15 | `grep -c 'Label: "'` over `AuthoredReadoutLayout` |
| Developer readout state words | 6 | `readoutStopped`, `readoutExtended`, `readoutArmed`, `readoutPatrol`, `readoutMarch`, `readoutFar` |
| Shop screen sentences | 12 | distinct `Msg:` in `pkg/game/shoproom.go` and `shopview.go` |
| Shop shelf names | 4 | `shopRoomShelves` |
| Town screen sentences | 4 | distinct `Msg:` in `pkg/game/townscreen.go` |
| Map-list notice after a won mission | 1 | `TownNotBuiltMessage` |
| HUD toggle letters | 4 | `hudToggleLabels` |
| Chargen preview placeholder | 1 | `pkg/ui/chargen_page.go` |

91 strings. They stay English because no research claim names an index for them.
`main.txt` lines 15..46 and 60..82 read like the panel's and the shop's words,
and that resemblance is not evidence: `MENU-STRTAB-008` states its own coverage
as 5 of 47 owners of `[L04369]`, and those surfaces are in the unresolved
remainder. Binding a word to an index because the English happens to match is
exactly what this story does not do.

One further limitation, in the same units: on a Russian install a menu row's
keyboard accelerator is a Cyrillic byte, which no key on an ASCII keyboard
produces. Those rows are chosen with the pointer. The row is still underlined
where the label marks it, and the English root is unaffected.
