# A window listing every mod mismatch at LOAD

## Intent

A LOAD refused because the save and the active mods do not match opens a window on the Load page
that lists every difference, one row each, with the mismatched part in red. The window says what
the player can do and closes with OK or Escape, leaving the Load page as it was. Both games show it.
Without a refusal nothing changes.

## Authority

Owner direction: when a LOAD is refused because the save and the active mods do not match, show a
window listing everything that does not match, such as the mods and their versions, mismatches in
red. The design beyond that is this story's. The refusal itself is `DIV-1802`.

## As built

### One list, two renderings

`mod.Compare` (`pkg/mod/digest.go`) returns the differences as data: `mod.Difference{Kind, Mod,
Setting, Saved, Current}` with nine kinds (base, missing, added, version, files, setting not declared
now, setting value, setting not recorded, load order). `mod.Differences` is `Compare` with each
difference's `String`, the prose it returned before, word for word. `TestCompareNamesEachDifferenceAsData`
pins every kind's fields and that the prose is the same list.

`applyModMark` (`pkg/game/modmark.go`) refuses with `*modMarkError`. Its `Error` is the text it
returned before (`mod set mismatch: ...`), it unwraps to `ErrModMark`, and its `ModMismatch` method
gives the same refusal as `ui.ModMismatch{Reason, Rows}`:

| Refusal | Reason | Rows |
|---|---|---|
| a mark of another mod set | `ModMismatchSet` | `mod.Compare(saved, active)` |
| a mark while no mod is active | `ModMismatchSet` | each saved mod as missing |
| no mark, or no readable document, while mods are active | `ModMismatchUnmarked` | each active mod as added |
| an unreadable mark, a digest that does not match its list, a mark that does not fit the save | `ModMismatchMark` | none |

`pkg/ui` may not import `pkg/mod` (`internal/archtest` DAG), so the row kinds are restated in
`pkg/ui/modmismatch.go` and `modMismatchKinds` maps them one to one. The Load page finds the data
with `errors.As` on the seam's error, through any wrapping; no string is parsed. Every text output
(the message line of a build without the installed font, a headless run, the command line) keeps
the error's one-line text.

### The window

`chooseLoad` (`pkg/ui/save.go`) opens the window when the error carries a mismatch and the installed
menu font is present; any other refusal still goes to the message line. The window owns the page's
input until it closes: Enter, a released OK press and Escape close it; the arrow, page, Home and
End keys, the wheel and the shared bar scroll it. The page's list, selection and message line are
untouched, so the page is as it was, including an earlier read error on the message line
(`TestALoadRefusedOverTheModsOpensTheWindowNotTheMessageLine` compares the frames;
`TestClosingTheWindowLeavesTheLoadPagesMessageLineAsItWas` closes the window by Escape, Enter and OK).

Rows are engine words (`load.mods.*` in `pkg/words/text/{en,ru}/strings.toml`). In a word,
`{mod}`, `{setting}`, `{saved}` and `{current}` are the row's values and text between `[` and `]` is
the mismatched part, drawn in the alert ink (236, 76, 60); the rest is drawn in the list's ink. A
save without a mark has a lead row before the active mods, and an unreadable mark is one row. The
hint is `load.mods.hint` (start with the save's mods and settings), or `load.mods.hint_unmarked`
(start with `-mods-accept-unmarked`, or without mods) for a save without a mark.

### One builder for the kind

The candidates were read and none fits a menu page:

- `noticedialogue.go` is the in-game ROM1 notice: the dialogue frame, a portrait pane, justified
  single-ink paragraphs, laid out from the install's notice layout over the map.
- `goldmodal.go` is the purse amount editor; `itempopup.go` an item's card over the inventory.
- The save dialog's confirmation and the Load page's delete confirmation replace the list with
  single-ink lines inside their own page and have no window of their own.

None draws runs of two inks or a list of rows in a window over a menu page. `pkg/ui/widgetnotice.go`
is therefore the builder of the menu notice window, in the widget kit: a window frame
(`windowFrame`, the page's frame art), a title, the shared list well (`listWell`) with rows that
wrap at spaces into indented lines, the shared vertical bar (`drawVScrollBar`, `scrollBarInput`),
a hint and one push button (`drawPushButton`, `buttonLatch`). The window's body is the page panel
less 12 pixels, so it covers the page's own buttons on both games; the list takes the rows the body
leaves. The list scrolls by wrapped lines and never draws outside the well
(`TestALongListScrollsInsideTheWindow`). A word wider than a line breaks at the width the new line
has after its indent (`TestALongVersionWrapsInsideTheRowWithEveryCharacter`, 120 characters).

### Both games

The Load page is the shared `SavePage` builder, so the second game's page shows the same window over
its own panel. Only the load role opens it; the save page never loads.

## Proof

Unit (`pkg/ui/modmismatch_test.go`, `pkg/game/modmark_test.go`, `pkg/mod/compare_test.go`):

- `TestALoadRefusedOverTheModsOpensTheWindowNotTheMessageLine`: a refusal of every kind opens the
  window with nine rows in the expected words; the message line stays empty; a key does not reach
  the list; Escape, Enter and OK close it; the frame after closing equals the frame before. Loss
  control: with the window branch of `chooseLoad` disabled the test fails with
  `no window; the message line says "b.sav: load b.sav: mod set mismatch: fixture prose"`.
- `TestTheWindowNamesTheRefusalsWithoutADifferenceList`: the unmarked lead row and hint, the
  unreadable-mark row, and an ordinary refusal that keeps the message line.
- `TestTheWindowDrawsTheMismatchedPartsInTheAlertInk`: each visible line has alert pixels exactly
  when it has an alert run, and plain pixels exactly when it has a plain run; no alert pixel lies
  outside the list.
- `TestALongListScrollsInsideTheWindow`: 40 rows; Down, PageDown, End, PageUp, Home, Up and the bar's
  bottom arrow move the first line as stated; no pixel outside the window changes; no wrapped line
  is wider than the well.
- `TestClosingTheWindowLeavesTheLoadPagesMessageLineAsItWas`: a first save leaves a read error; a
  second opens the window; Escape, Enter and OK each leave the same message and the same frame.
- `TestALongVersionWrapsInsideTheRowWithEveryCharacter`: with the Load panel's geometry, a version of
  120 characters wraps into lines none wider than the row's text width, indent included, and every
  character remains.
- `TestLoadRefusalsCarryTheirDifferencesAsData`: `applyModMark` on a fixture SAV gives the expected
  reason and rows for no mods, another version, another setting, an extra mod, a mod fewer, an
  unmarked save and an unreadable mark, and still wraps `ErrModMark`.

Release (fixture mods written by the test; no install byte in the repository):

- `TestReleaseLoadPageListsEveryModMismatch` (EN, RU): a mission 10 SAVE under alpha 1.0 and one
  without mods, each loaded on the production Load page (`ConfigureSaveSeams`) under alpha 1.1,
  no mods, alpha with beta, alpha with `skill_cap=120`, alpha with twelve more mods, and the unmarked
  save under alpha. Each opens the window with the expected rows and red pixels in its list; the
  twelve-mod case needs scrolling and End reaches the last line; Escape returns to the Load page.
  Under alpha 1.0 the marked save loads into the mission.
- `TestReleaseSecondGameLoadPageListsModMismatches` (`rom2-en`, `rom2-ru`): a SAVE on the second
  game's save page under alpha 1.0 is refused under alpha 1.1 with both versions in red.

Frames (headless renderer, outside the repository): `review/story1494-mod-mismatch-window/frames/`
`{en,ru}/{version,missing,extra-state-mod,setting,unmarked,many,many-scrolled}.png` and
`{rom2-en,rom2-ru}/load-page-version.png`.

A LOAD with matching mods and with no mods is unchanged: the existing release tests of the mod
mark and the LOAD route pass, and no World hash or SAV byte is written by this change.

## Divergences

None new. The original game has no mods, so the window appears only under a refusal `DIV-1802`
already holds; that row now names the window. Without mods no frame, World hash or SAV byte changes.
The reserved IDs `DIV-3893` to `DIV-3900` are unused.

## Open debt

- The window's geometry (body inset, alert ink) is this build's own; no owner frame review yet.
- A refusal raised after the mark is accepted (a mod mission without its guest record) is not a
  mark refusal and still uses the message line.
