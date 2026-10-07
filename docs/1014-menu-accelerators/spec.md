# 1014 — spec (as-built)

## Subject

The in-game menu's own keyboard accelerator: the lowercase step that resolves a marked label byte
to a matchable key, and the keyboard-side encode that turns a typed character into a byte the fold
can match. `MENU-KEY-013` decodes both halves of the original's own routine; before this story only
the label-marking half was built (`0168`).

## Label-side fold

`gameMenuLower(c byte, selector int) byte` (`pkg/ui/gamemenu.go`) is the decoded routine's own
lowercase step. It lowercases ASCII `'A'-'Z'` unconditionally, then — only when `selector` equals
`text.SelectorConverting`, the install's own Russian selector — folds CP866 Cyrillic uppercase by
the claim's exact two offsets:

- `0x80..0x8f` (CP866 uppercase А-П) → `+0x20`
- `0x90..0x9f` (CP866 uppercase Р-Я) → `+0x50`

Every other byte, and every selector but the Russian one, is returned unchanged. A byte already in
the CP866 lowercase ranges (`0xa0..0xaf`, `0xe0..0xef`) is therefore also unchanged, since neither
fold clause reaches it — this matches the claim, which describes only the two uppercase ranges.

`gameMenuAccelerator(label string, fallback byte, selector int) byte` (previously
`gameMenuAccelerator(label string, fallback byte) byte`) now takes the install's own selector and
routes both the label's own marked byte and the fallback immediate through `gameMenuLower` before
returning. `gameMenuLabelText`, the `~~` escape walk, and the marked-byte search are unchanged from
`0168` — the brief's premise that this half needed no rework is correct.

## Keyboard-side encode

`chooseGameMenuAccelerator` (`pkg/ui/save.go`) changed from taking a raw `byte` to taking a `rune`:

```go
func (f *flow) chooseGameMenuAccelerator(r rune) bool
```

It resolves an encoder from `f.encodeMenuKey` (falling back to `defaultMenuKeyEncode`, ASCII-only
passthrough, when nil), encodes the rune, folds the result with `gameMenuLower` at the flow's own
selector (`f.menuSelector()`, read off `f.menuFont.Selector`, `0` when no font is set), and compares
against each row's own folded accelerator (`gameMenuAccelerator`) in `menuRows()`'s own order,
returning on the first match.

`encodeMenuKey func(rune) (byte, bool)` is a new field on `flow` (`pkg/ui/flow.go`). It crosses the
architecture DAG as a function value rather than an import: `pkg/ui` does not have
`pkg/formats/textinput` on its own allow-map row in `internal/archtest/dag.go`, and widening that
row was not needed, because `pkg/game/chargen.go` already carries the identical pattern for
character-name entry — a higher tier (`pkg/game`, which can import the leaf) builds a closure and
hands it down as a plain function value. `App.SetWords` (`pkg/ui/app.go`) gained a third parameter,
`encode func(rune) (byte, bool)`, stored on `flow.encodeMenuKey`. `frontend.go`'s `App()`
construction supplies `func(r rune) (byte, bool) { return textinput.EncodeRune(r, selector) }`,
`selector` read off `f.Font.Selector` (`0` when `f.Font` is nil, matching the FontErr-carried,
never-fatal convention this package already uses elsewhere).

`defaultMenuKeyEncode` (`pkg/ui/save.go`) is the nil-encoder fallback: ASCII `0x20..0x7e`
passthrough, nothing else. It is what `cmd/mapview` and every hand-assembled test flow that does not
set `encodeMenuKey` gets, and it reproduces the pre-story EN-only behaviour exactly.

## The dispatch bug this exposes

`App.stepGameMenu`'s `Typed` case (`pkg/ui/app.go`) previously ranged `in.Typed` — a UTF-8 string
from `ebiten.AppendInputChars` — as bytes, calling `chooseGameMenuAccelerator` once per byte. A
Cyrillic character is two bytes in that string, so neither byte alone is a valid rune, and the call
could never have matched a Cyrillic accelerator regardless of the fold. The loop now ranges runes:

```go
for _, r := range in.Typed {
    if a.flow.chooseGameMenuAccelerator(r) {
        a.syncViewerLayout()
        break
    }
}
```

This is a pre-existing defect exposed by this story rather than introduced by it: before this
story no accelerator this build resolved was ever non-ASCII, so the byte/rune distinction was
inert. `HeadlessType`'s own doc comment, which stated `Typed` was read on no screen but chargen, is
corrected to note `stepGameMenu` also reads it now (`pkg/ui/chargen_headless.go`).

## Which real shipped bytes exercise which clause

Measured directly against both preserved installs with `cmd/menuaccelcheck labels` (below). RU
marked bytes: `MenuSave` `0x91`, `MenuLoad` `0x82`, `MenuGameOptions` `0x8e`, `MenuSoundOptions`
`0x87` — all four in the uppercase fold ranges, so the label-side fold is exercised on shipped data
by these four rows, and the matching keyboard-side fold is exercised by typing the corresponding
LOWERCASE Cyrillic letter (the natural keystroke, no Shift). `MenuQuestObjectives` `0xa0`,
`MenuEndQuest` `0xaa`, `MenuAbort` `0x82`, `MenuReturn` `0xad` are already in the lowercase ranges
(`0xa0..0xaf`), left unchanged by both fold clauses — these are the rows `MENU-KEY-013` itself
describes as deliberately marked away from the first letter, on the mission menu, to avoid a
first-letter collision.

## Mutation evidence

`TestGameMenuLowerFoldsExactlyMENUKEY013sTwoRanges` (`pkg/ui/words_test.go`) asserts both range
boundaries and both range interiors, plus that a byte outside either range and every selector but
the Russian one pass through unchanged. Swapping the `+0x20`/`+0x50` offsets in `gameMenuLower`
(backed up, mutated, restored) turned four tests red: this one directly, plus
`TestRUKeyboardAcceleratorReachesItsLabel`, `TestRUTownAcceleratorCollisionFirstMatchWins` and
`TestAcceleratorOverAnInstalledLabel`, confirming the fold offsets are load-bearing rather than
vacuously covered.

## Real-install witness (independent of `EXP-0162`'s own evidence file)

`cmd/menuaccelcheck labels -assets <root>` reads each install's own resolved `Words.MenuXxx` fields
and prints every marked byte with its offset. Run against both preserved installs, this
independently reproduces `EXP-0162/evidence/labels.txt`'s own byte values (RU marks: `0x91`, `0x82`,
`0x8e`, `0x87`, `0xa0`, `0xaa`, `0x82`, `0xad` for the eight `MenuXxx` rows in declared order) and is
the source for the "which real bytes exercise which clause" statement above — read from the live
install file, not only from the research repo's own recorded evidence.

`cmd/menuaccelcheck key -assets <root> -save @first -key <rune>` drives the full production
dispatch against a real loaded original save: `App.HeadlessKey("load")` → `HeadlessActivate("@first")`
→ `HeadlessKey("escape")` → `HeadlessType(<rune>, false)`, printing the resulting screen and message.
Evidence in `closure.md`.

## Cut list

None. `MENU-KEY-013`'s decoded fold and the keyboard path it requires are both built in full. The
two residuals below are properties of the shipped data and of this build's own architecture
boundary, not omissions from the claim.

## Divergence disposition

`DIV-006` moves to `DIVERGENCES-CLOSED.md`: both halves of `MENU-KEY-013` (the label fold, built at
`0168`; the keyboard fold and encode, built here) are now implemented and mutation-tested. `DIV-140`
and `DIV-141` are both spent, added to "Authored where research is silent" as `UNKNOWN`:

- `DIV-140` — `MENU-KEY-013` decodes the fold applied to a byte the original's own keyboard handler
  has already produced; it does not name which physical key produces that byte. This build reads a
  rune `ebiten.AppendInputChars` has already resolved through the operating system's own
  keyboard-layout mapping; it chooses none of that mapping itself.
- `DIV-141` — the RU town menu's `MenuLoad` and `MenuAbort` both mark byte `0x82` (their shared
  first letter, В). `MENU-KEY-013` shows the original's own mission-menu labels were placed away
  from the first letter specifically to avoid this shape among rows 5-7, but does not establish
  whether the original's own runtime matcher (as opposed to the decoded label-construction routine)
  resolves the same collision in the town menu the same way. This build's own matcher, walking rows
  in declared order and returning the first match, makes Abort unreachable by the `в` key in the RU
  town menu; Abort remains reachable by pointer, as it already was for every row before this story.
