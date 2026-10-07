# 1014 — closure

Branch `1014-menu-accelerators`, base `fa13230`. Research pin `02a1403` (unchanged; no story
boundary crossed, per SDD S-5 mid-story freeze — this story opened and closes within one pin).

Gate: `go build ./...`, `go vet ./...`, `gofmt -l` over tracked and untracked Go files, and
`go test -trimpath -count=1 ./...` all pass with no game install present (57 packages, all `ok` or
`[no test files]`). `bash scripts/check-no-game-assets.sh` reports `clean (tree scan)`. The deletion
set against base is empty — no file under the repository was deleted.

## Result

`MENU-KEY-013`'s decoded CP866 fold is implemented on both halves it requires: the label-marking
lowercase step (`gameMenuLower`, `pkg/ui/gamemenu.go`) and the keyboard-side encode
(`chooseGameMenuAccelerator`, `pkg/ui/save.go`, plus the `encodeMenuKey` seam threaded from
`pkg/game/frontend.go` through `App.SetWords`). A pre-existing byte/rune dispatch bug in
`stepGameMenu`'s `Typed` case is corrected, without which no multi-byte Cyrillic keystroke could
have reached an accelerator regardless of the fold. `docs/DIVERGENCES.md`'s `DIV-006` moves to
`DIVERGENCES-CLOSED.md`. `DIV-140` and `DIV-141` are both spent as `UNKNOWN` rows in "Authored where
research is silent" (see `spec.md`'s "Divergence disposition").

## The twelve-aspect matrix

| Aspect | Verdict | Evidence |
|---|---|---|
| Data | N/A | No archive, registry or table format touched; the install's own `dialogs.txt` bytes are read exactly as `0168` already reads them. |
| Runtime state | N/A | No new persisted or session field; `flow.encodeMenuKey` is a per-`App` function value, not serialized state. |
| Simulation | N/A | `pkg/sim` is untouched; `go build`/`go vet`/`go test` confirm no file under it changed. |
| Player input | PASS | The keyboard-to-accelerator path is the story's second half: `TestRUKeyboardAcceleratorReachesItsLabel` (both lowercase and uppercase Cyrillic reach the RU Save row), the real-install witness below (a real keystroke through `HeadlessType`, the same call a windowed frame makes). |
| AI | N/A | No unit decision is reached. |
| UI / HUD | PASS | The whole subject: label fold plus keyboard fold, both tested and witnessed against real installs. |
| Triggers / scripts | N/A | Not touched. |
| Inventory / equipment | N/A | Not touched. |
| Persistence / save-load | PASS | The real-install witness exercises a real save write (through the existing, unchanged `SaveStore`/`OriginalStore` seam, to a scratch directory — never the install) as the RU Save row's own action; no serialized field or format changed. |
| Campaign / session | N/A | Not touched. |
| Shipped content | PASS | `cmd/menuaccelcheck labels`, run against both preserved installs, reproduces `EXP-0162`'s own evidence file's byte values independently, from the live install rather than only from the research repo. |
| Interactions with existing mechanics | PASS | EN regression checked directly: `cmd/menuaccelcheck key` against `gameversions/en` with `-key s` still reaches Save (below); `TestAcceleratorOverAnInstalledLabel` and the pre-existing EN-only tests are unchanged and pass. |

No in-scope GAP.

## Integration witness

All four runs below are against the two preserved installs
(`<seat>\gameversions\en`, `...\ru`), using the new tool `cmd/menuaccelcheck`
(`internal/archtest/dag.go` allow-map row added: `pkg/game`, `pkg/ui`, `pkg/formats/res`).

**Label bytes, both roots** (`menuaccelcheck labels`). RU output (selector 1):

```
MenuSave             7e 91 ae e5 e0 a0 ad a8 e2 ec 20 a8 a3 e0 e3  marked byte 0x91 at offset 1
MenuLoad             7e 82 ae e1 e1 e2 a0 ad ae a2 a8 e2 ec 20 a8 a3 e0 e3  marked byte 0x82 at offset 1
MenuGameOptions      7e 8e af e6 a8 a8 20 a8 a3 e0 eb  marked byte 0x8e at offset 1
MenuSoundOptions     7e 87 a2 e3 aa ae a2 eb a5 20 ae af e6 a8 a8  marked byte 0x87 at offset 1
MenuQuestObjectives  87 7e a0 a4 a0 ad a8 a5  marked byte 0xa0 at offset 2
MenuEndQuest         87 a0 7e aa ae ad e7 a8 e2 ec 20 ac a8 e1 e1 a8 ee  marked byte 0xaa at offset 3
MenuReturn           82 a5 e0 7e ad e3 e2 ec e1 ef 20 aa 20 a8 a3 e0 a5  marked byte 0xad at offset 4
MenuAbort            7e 82 eb e5 ae a4 20 a8 a7 20 a8 a3 e0 eb  marked byte 0x82 at offset 1
```

EN output (selector 0, unchanged from before this story): eleven-of-thirteen marked (`MenuAbort`
carries no mark, unaffected). Both outputs match `EXP-0162/evidence/labels.txt` exactly.

**RU real-install keystroke** (`menuaccelcheck key -assets <ru root> -save @first -key с`, lowercase
Cyrillic es, U+0441 — chosen because `MenuSave`'s marked byte `0x91` is in the fold range, so this
run exercises the `+0x50` fold clause on real shipped data, not only the already-lowercase rows):

```
loaded "@first": screen = map
game menu open: 7 rows [{Сохранить игру true} {Восстановить игру true} ...]
typed "с" (U+0441): screen = gamemenu, message = "saved as save-20260818-133459.ags"
```

The typed lowercase с reached and chose the Save row, through the exact production dispatch a
windowed keystroke uses (`App.HeadlessType` → `stepGameMenu`'s rune loop →
`chooseGameMenuAccelerator` → `textinput.EncodeRune` → `gameMenuLower`'s `+0x50` fold →
`gameMenuAccelerator`'s own folded comparison), and wrote a save to the scratch directory this run
supplied, never to the install. Confirmed separately: no file under either `gameversions/en` or
`gameversions/ru` carries this run's timestamp, and `pipeline/check-preserved-installs.sh` reports
`ok — 162 file(s), both roots as recorded` after the run.

**EN regression** (`menuaccelcheck key -assets <en root> -save @first -key s`):

```
loaded "@first": screen = map
game menu open: 7 rows [{Save Game true} {Load Game true} ...]
typed "s" (U+0073): screen = gamemenu, message = "saved as save-20260818-133522.ags"
```

Unchanged from the pre-story ASCII path (selector 0, `gameMenuLower` returns `c` unchanged for
every byte outside the fold ranges, which every ASCII byte already was before this story).

**What this does not witness.** `-key с` exercises the fold's `+0x50` clause on the RU Save row; no
run in this closure exercises the `+0x20` clause (`0x80..0x8f`) against a real install keystroke —
`TestGameMenuLowerFoldsExactlyMENUKEY013sTwoRanges` covers it directly, at the unit level, not
against a real save. The town-menu collision (`DIV-141`) is witnessed by
`TestRUTownAcceleratorCollisionFirstMatchWins` with byte fixtures cross-checked against this
closure's own `labels` output (`MenuLoad`/`MenuAbort` both `0x82`), not by a live town session — the
mission-menu witness above already exercises the identical production dispatch, and driving a town
session adds no additional code path this story changes.

## Added at the landing

The adversarial review found that no test witnessed the dispatch this story fixed. Reverting
`stepGameMenu`'s rune walk to the byte walk it replaced -- the story's own defect -- left the whole
suite green, because every accelerator test called `chooseGameMenuAccelerator` directly and entered
the pipeline below the loop. `TestATypedCyrillicRuneReachesTheMenuThroughTheStep`
(`pkg/ui/headless_gamemenu_test.go`) drives the same rune through `a.step`, and reddens under that
revert while the two direct tests stay green. Production behaviour was correct throughout; the gap
was in the witness.

## Research reconciliation

- `MENU-KEY-013`'s two fold ranges are implemented exactly as decoded and mutation-tested
  (`spec.md`, "Mutation evidence"). No claim is refuted.
- The claim's own note that RU marks are placed away from the first letter "where the first letters
  would collide" (mission rows 5-7) is read narrowly: it is evidence the original's LABEL AUTHORS
  avoided a specific collision shape in the mission menu, not evidence about the original's own
  RUNTIME accelerator matcher, and not evidence about the town menu. The town menu's own shipped
  labels (`MenuLoad`, `MenuAbort`) carry the same collision shape the mission menu avoids, measured
  directly from both installs' `dialogs.txt` in this closure. Recorded as `DIV-141`, typed `UNKNOWN`
  rather than `FIDELITY-DEBT`, because the original's own runtime matcher for this exact case is not
  decoded by any claim read for this story.
- No claim was refuted by this story.

## Script-gap census

Unaffected. This story touches no script opcode or trigger path; measured directly, from this
worktree:

```
go build -o /tmp/mr ./cmd/missionrun
AGAINROM_ASSETS=<en> /tmp/mr -mission 10 -trace -ticks 1 | grep -c UNSUPPORTED  ->  0
AGAINROM_ASSETS=<en> /tmp/mr -mission 20 -trace -ticks 1 | grep -c UNSUPPORTED  ->  0
```

Both match `pipeline/milestone-baseline.txt`'s current values (0 and 0). This story does not move
the census; it was not expected to — the change made is to `pkg/ui`'s own input dispatch, which the
census does not measure.
