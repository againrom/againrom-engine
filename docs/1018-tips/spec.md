# Spec — 1018-tips

Canonicalized to the code as built at this story's landing. `contract.md` records what was asked
and what the pin knew before implementation; this document records what the build does.

## Result

The town square, the shop, the school, the tavern and the character generator's pre-create page each
show that screen's own shipped tip text in a floating panel, drawn from three shipped art nodes and
composed over the screen's own content. The panel carries a close control and a labelled toggle.
Closing the panel dismisses it for the rest of the session on that screen; the dismissal does not
affect any other screen's panel. The toggle suppresses every screen's panel and is persisted to a
store beside `saves/`, read back at startup, defaulting to on.

## Behaviour 1 — the panel widget

`ui.TipPanelView` (`pkg/ui/tippanel.go`) is the shared view type each of the five screens builds
and each of `app.go`'s four call sites reads: a rectangle, the resolved text, the toggle's own current state, the resolved art
(`*ui.TipPanelArt`) and a font. `Showing()` requires all four of `Art`, `Font`, `Text` and a
non-empty `Rect`; a screen whose text is empty, whose art failed to load, or that has not built a
font draws nothing.

`ui.TipPanelArt` holds three pictures, all resolved by `pkg/game`'s `LoadTipPanelArt`
(`pkg/game/tipart.go`) from three shipped nodes read for the first time in this tree:

- `interface/t_back.bmp` (160x240, 24-bit BI_RGB), tiled opaquely across the panel's interior by
  `tileBlitSrc`.
- `interface/t_border.bmp` (88x108). **Ships 8-bit paletted, not 24-bit.** `LoadTipPanelArt`
  originally read it with `readChargenBMP`, the same 24-bit-only reader `t_back.bmp` uses; against
  both preserved installs this failed with `"terrain: bmp is 24 bpp, want 8"` (`t_border.bmp`'s own
  header: `bpp=8`, palette at byte 54, pixel data at offset 1078 = 54 + 256*4). Fixed by reading it
  with `readChargenMask` (`terrain.DecodeBMP8`, the reader this tree already uses for `mask.bmp` and
  `PathMap.bmp`) and a new helper, `paletteRGBA`, that resolves the palette to RGBA via
  `image/draw`'s `draw.Draw` — the first 8-bit BMP in this tree read for its colour rather than its
  index. The result is keyed on pure black (`keyBlack`) before use, on the contract's own reading of
  the picture as "an ornate gold frame ... on a keyed black field," drawn as a nine-patch
  (`drawNinePatchBorder`) with corners one quarter of the source's own 88x108 (`tipPanelBorderCornerW
  = 22`, `tipPanelBorderCornerH = 27`). **This corner fraction is authored, not decoded**: no claim
  or photograph gives the real corner size.
- `interface/radiob.256`, six frames (24x24 round off/on, 24x24 square off/on, 16x16 small-square
  off/on). Frames 4 and 5 (the small-square pair) are read as the toggle's unchecked and checked
  pictures — **authored**, the closest shape to a checkbox beside a line of text; no claim names
  which of the three pairs the original toggle uses.

The pairing of these three nodes as a panel's fill, frame and toggle gem is inference from the
contract's own reading of the owner's photographs, not a decoded fact: no claim names any of the
three addresses. `LoadTipPanelArt` is cosmetic on `LoadTownSquareArt`'s own rule — a missing or
mis-sized node returns an error naming its own address, and the front end that calls it
(`FrontEnd.tipArt`, cached the way `shopArt` is) falls back to drawing no panel rather than making
the game unusable.

`ComposeTipPanel` draws, in order: the tiled fill, the nine-patch border, the wrapped text (measured
and broken by the existing `wrapShopTip`, `DLG-WRAP-009`'s own line pitch), the close control
(`drawTownShellBox`/`drawTownShellText`) and the toggle (the gem for the current state plus its
label). The two captions are installed words: `TOWN-206` joins ids `0x7f` and `0x80` to
`text/main.txt` lines 127 and 128. `InstallWords.Words` resolves those lines into `ui.Words`, and
the town and character-generator paths carry the selected EN or RU strings in `TipPanelView`.
An empty hand-built view uses the exact EN strings as its visible fallback. `TipPanelControlAt`
hit-tests a point against a
showing panel: any point inside `Rect` is consumed (`consumed=true`), and a point inside
`TipPanelCloseRect`/`TipPanelToggleRect` additionally names which control. All four call
sites in `pkg/ui/app.go` (`stepTownAt`'s `inSurface`, `inShop` and town-square branches, and
`stepPreCreate`) resolve the panel's hit test **before** their own room's controls and return
immediately whenever `consumed` is true, whatever `kind` is — a showing panel swallows every press
and release inside its own rect, on all five screens, the way `inShop` (`app.go:1703`) has done
since round 1. There are four sites and five screens because `inSurface` serves both the tavern and
the school. Close and Toggle take input before the screen beneath it because nothing in the rect
reaches the screen beneath it while the panel is showing.

**A swallowed press or release also clears the covered screen's own press state** (hotfix `66ee8bf`,
2026-08-19, folded in here at the landing; pass-3 review's P-1). Returning early is not sufficient by
itself: three of the four sites return before the block that arms or resolves their own screen's
latch, so a gesture crossing the panel's boundary — press on a live control and release inside the
rect, or the reverse — used to leave that latch exactly as the last un-swallowed event left it, and a
later event acted on it. `inSurface` clears `townSurfacePress`, `inShop` calls `clearShopDrag` (the
armed flag, the accumulated distance and the captured icon; the per-frame doll suppression follows,
because the block that pushes it reads `shopDragArmed` unconditionally), and `stepPreCreate` clears
`chargenPress`, which also keeps a swallowed gesture out of the choice controls' double-click window.
The town-square site sits inside `if in.PrimaryReleased` and holds no press latch, so it is
unchanged. `pkg/ui/tippanel_crossing_test.go` witnesses the three with a press at one point and a
release at another, which is the gesture `tippanel_overlap_test.go`'s same-point `clickAt` cannot
express.

**Round 2 narrowed this guard at the four non-shop call sites to `consumed && kind !=
TipControlNone`, and round 3 reverts it (round-3 review, P-1, DIV-162).** Round 2 widened each
non-shop rect past its own researched width to wrap the shipped text into fewer lines, then let a
press or release elsewhere in the (now wider) rect fall through to the room's own hit test, on the
reasoning that the widened rects left a smaller residual overlap with each screen's controls. Driven
through the real dispatch against both preserved installs, that let a click land on the panel's own
PAINTED surface — its gold border, its own wrapped text — and reach whatever control the room's own
hit test found under that pixel: a release on the town square's own left border chose a door and left
the square (`Choose(3)` → `enterWorldMap()`), a click on the generator's own tip text committed a
class the player had not selected, a click inside the school's own tip text spent a skill's price
with nothing on screen to show it. This is story 1005's own shipped defect shape: a hit test
answering for a pixel the player cannot see.

Round 3 restores the bare-`consumed` guard at all four call sites and separately restores each
non-shop rect's own **researched width and top-left corner**: `TownTipRect` `(328,0)-(640,H)`
(`TOWN-165`/`TOWN-184`), `TavernTipRect` `(0,0)-(312,H)` (`TOWN-015`), `SchoolTipRect` `(0,0)-(456,H)`
(`TOWN-021`/`TOWN-188`), `ChargenTipRect` `(0,280)-(312,480)` (`TOWN-187`, whole and unmodified).
Since every press and release inside the rect is now swallowed regardless of what it visually covers,
overlap costs the player nothing while the panel is showing, and the only deviation this story owns
is `H` on four of the five rects: each height, grown past its researched 200px only as far as that
screen's own shipped tip text needs to draw whole at the researched width, on the worse of the two
preserved roots, plus a small margin. The generator's rect takes no growth at all — its researched
200 already holds both branch texts. `cmd/tippanelcheck` (new, replacing the uncommitted scratch
measurements round 2 relied on) searches and prints, per screen and per root, the rect, the minimum
fitting height, the spare rows at the chosen height, and what the rect covers:

| screen | rect | minimum fit (en / ru) | chosen H | spare |
|---|---|---|---|---|
| town square | `(328,0)-(640,373)` | 295 / 363 | 373 | 10 (worse root) |
| tavern | `(0,0)-(312,322)` | 295 / 312 | 322 | 10 (worse root) |
| school | `(0,0)-(456,271)` | 261 / 210 | 271 | 10 (worse root) |
| generator | `(0,280)-(312,480)` | 160 / 160 (mage low 148 ru) | 200 (researched) | 40 |
| shop | `(164,162)-(476,340)` | 176 / 142 | 178 (unchanged, `SHOP-TIP-045`) | 2 (en) |

Round 3 shipped `ChargenTipRect` as `(0,310)-(312,480)`, 30 rows lower and 30 rows shorter than
`TOWN-187`'s own `left=0, top=0x118, right=0x138, bottom=0x1e0`, while this section, `closure.md`,
`DIV-162` and the rect's own comment all said the top-left was researched. It was the one rect of the
five smaller than the original popup, a direction no document disclosed (round-3 adversarial review,
D-1). The rect is `TOWN-187`'s own at the story's landing.

**What each rect covers, per control.** A live-pixel total is stated in the format's own vocabulary
rather than the reader's, so `cmd/tippanelcheck` prints each covered control as a percentage of that
control's own hit-testable area. The figures below are identical on both roots and come from a fresh
front end with no campaign loaded, so a control the room disables answers no hit and is absent — they
are a lower bound on what a mid-campaign player sees covered, not a fixed property of the screen.

| screen | covered controls |
|---|---|
| town square | statue/menu 65.2% (3647 of 5592 px), door 2 50.8% (7477 of 14733), door 1 4.4%, doors 0 and 3 under 0.2% |
| tavern | cell 0 50.0% (4536 of 9072 px) |
| school | cell 0 100% (1820 px), cell 1 100% (2007 px), cell 2 91.3% (1554 of 1702) |
| shop | shelf pick 0 and 1 71.9% each (13965 of 19425 px), merchant 71.6% (9576 of 13376), table cells 2/3/4 46.2% each, table cell 1 16.2% |
| generator | choice 0 97.9% (17431 of 17812 px), choice 2 13.1% (2162 of 16458) |

Two of the school's three skill cells are completely covered while the tip shows. A rect that matches
its own researched size and still covers a control is the original's own layout, not this build's
regression: the generator's researched rect covers 97.9% of choice 0's own fighter portrait before
any height is added, and the shop's own rect — sourced, never re-sized in round 2 or round 3, since
it already held the shipped text at its 136-row height plus `tipPanelChromeH` — covers the largest
area of the five. No control on any screen is entirely unreachable: the panel is dismissible per
visit from its own Close control, and every call site swallows the click while it shows.

**`TOWN-185`'s own child rects were available and are not used.** The claim carries three child
controls' ids, sizes and rectangles inside the original popup at High confidence. This build authors
its own close, toggle and text rects from `tipPanelInset`, `tipPanelButtonRowH`, `tipPanelCloseW`,
`tipPanelGemSize` and `tipPanelGemGap` instead, because the chrome those rects position is this
story's own authored chrome (`DIV-161`) rather than the original's two `STRINGTABLE` controls. The
outer rects are researched; the inner geometry is not (round-3 adversarial review, D-3).

`TipPanelFits` (exported) reports whether a view's wrapped text is drawn whole inside its own text
rect; all five rects were sized so that every one of the twelve real non-shop combinations — five
non-shop screens' own shipped tip texts (six counting both generator branches), `en` and `ru` — draws
whole, measured directly against both preserved installs' own font via `cmd/tippanelcheck`.
`TestReleaseTipPanelTextsDrawWholeOnBothClasses` (`pkg/game/tippanel_release_test.go`) is the
standing regression witness for the fit. `pkg/ui/tippanel_overlap_test.go` is the round-3 witness for
the hit test, rewritten from round 2's four per-site regression tests into five per-screen
enumeration tests — `TestTownSquareTipPanelSwallowsEveryControlItCovers`,
`TestTavernTipPanelSwallowsEveryCellItCovers`, `TestSchoolTipPanelSwallowsEverySkillIconItCovers`,
`TestShopTipPanelSwallowsEveryControlItCovers`, `TestPreCreateTipPanelSwallowsEveryPortraitItCovers`
— each enumerating every control its own screen's hit test can name, asserting through the real
`app.go` dispatch that a press or release anywhere inside the panel's rect activates none of them
while the panel is showing, and that Close and Toggle both still work. `ShopTipRect()`
(`pkg/ui/shopscreen.go`) keeps its own sourced 136-row height (`SHOP-TIP-045`) unchanged and adds
`tipPanelChromeH` (42 rows: two insets, the button row and its gap) below it — the close/toggle row
is additional floor space, never carved from the 136 rows DIV-133 already measured as an exact,
no-spare-line fit for the shipped EN shop text before this story.

## Behaviour 2 — each screen's own text on entry

`pkg/game/tips.go` adds four address constants (`TownTipPath`, `SchoolTipPath`, `TavernTipPath`,
already-existing `ShopTip1Path` from story 1011) and two for the generator
(`ChargenFighterTipPath` = `chrgen1f.txt`, `ChargenMageTipPath` = `chrgen1m.txt`). **The two
generator files are class text, not sex text**, despite sharing the `1f`/`1m` letters with
`graphics.res`'s `equipment/` gender convention: `chrgen1f.txt` reads "one of the five weapon
skills" and `chrgen1m.txt` "one of the five magic spheres," and `TOWN-187` (corrected at research
`8990406`, an ancestor of this story's pin) reads the generator's own selection as
`+0x18c & 0x2`, the mage/non-mage class split — the same bit `TOWN-149` and `HERO-FIGURE-059` read,
distinct from the sex bit at `+0x18c & 0x4`.

`townScreen.loadTip(room)` reads one room's own node into its own field
(`townTip`/`schoolTip`/`tavernTip`/`shopTip`) and is called on every entry into that room: at
`TownScreen()`'s own construction (`roomSquare`), from `Choose(i)` for every room `townDoors`
resolves to, from `talk(i)` and `Back()` on returning from a conversation, and from `Back()`'s own
return to the square. `TipsOff()` is read once per call, before the file is opened at all — a
suppressed front end never reads the node, rather than reading and hiding the result. A room whose
node the install does not ship, or a suppressed front end, both resolve an empty string, which
`tipView` turns into a zero `TipPanelView` (`Showing()` false).

For the generator, `ChargenSetup` (`pkg/game/chargen.go`) resolves BOTH branches through the same
pure function, `chargenTipPath(classStart int) string`: `0` resolves to `ChargenFighterTipPath`,
any other value to `ChargenMageTipPath`. **Round 1 resolved only the fighter branch, once, at
`ChargenSetup()` construction** (`TipText: ReadShopTip(src, chargenTipPath(choices[chargenChoiceClass].Start))`),
and `choices[chargenChoiceClass].Start` is never seeded by any caller in this tree (confirmed by
grep across `pkg/game`, `pkg/ui`, `cmd/`), unlike the skill row's own `-skill` precedent — so every
fresh chargen resolved to the fighter tip regardless of what portrait the player went on to pick,
and the mage branch was unreachable through any player action (round-2 review correction,
DIV-162). `ChargenSetup` now resolves both `ChargenFighterTipPath` and `ChargenMageTipPath` once
each, into `ui.ChargenSetup.TipText` and the new `ui.ChargenSetup.TipTextMage`.
`ui.Chargen.TipPanel()` picks between the two using `preChoiceParts(c.preChoice)`'s own class half
— the same field `Forward` already reads to commit the choice row — so the shown text tracks
whichever portrait the player currently has selected on the pre-create page, live, rather than a
class this package cannot otherwise observe; `TipTextMage` empty falls back to `TipText`, so a
hand-built setup that sets only `TipText` (every setup built before this field existed) draws
exactly what it drew before. This is still a snapshot read, not a continuous binding to
`TOWN-187`'s own second popup (`chrgen2.txt`, a once-only latch replacing the popup's text in place
once the detailed page opens, reading a text node unrelated to the fighter/mage split) — that
mechanism stays out of this story's scope; only WHICH of the two already-resolved, already-in-scope
texts is shown now follows the player's own choice. `pkg/ui/chargen_tip_test.go`'s
`TestPreCreateTipTextFollowsTheLiveClassChoice` and `pkg/game/tippanel_release_test.go`'s
`TestReleaseChargenTipPanelFollowsTheLiveClassChoiceAgainstRealArt` are the new regression
witnesses, the second against real shipped art on both preserved installs. The release-side fit
regression test (`TestReleaseTipPanelTextsDrawWholeOnBothClasses`) still reads both
`ChargenFighterTipPath` and `ChargenMageTipPath` directly, independently of which one a given setup
picks, to prove the fit against `ChargenTipRect` for each shipped node on its own.

## Behaviour 3 — per-visit close

`townScreen.tipClosed [roomTalk + 1]bool` is indexed by room. `CloseTip()` sets
`tipClosed[t.room] = true`; `tipView` returns a zero view for any room whose bit is set. The array is
reset only by `resetForNewGame`, never by a room transition within the same game — "for the rest of
the visit" is read as the whole session, on `townUI`'s own "remembered on the front end" precedent.
`resetForNewGame` has four call sites (round-2 review correction, DIV-162): one is a genuinely new
campaign (`frontend.go`'s `ChargenEntry.Begin`), and three are loading an existing save — the
original `.sav` form, both a mid-mission and a between-mission branch (`originalsave.go`), and this
build's own `.ags` form (`resume.go`). Every one of the four installs a genuinely different game's
session state, which is what `resetForNewGame`'s own doc comment enumerates; a loaded save's
previously-closed tips do not carry into the game now installed. Closing the shop's panel does not
close the school's: each room has its own bit.
`ui.Chargen.tipClosed` is the same shape for the generator's own single room, set by `CloseTip()` and
read by `TipPanel()`.

## Behaviour 4 — the permanent toggle and its store

`pkg/game/tipstore.go`'s `OptionsStore` is a flat key=value text file (`options.txt`) beside
`saves/`, resolved by `DefaultOptionsPath` as `saves/`'s own parent directory joined with the file
name — beside the executable, beside the working directory for `go run`, or beside an overridden
saves directory, so `cmd/againrom`'s own `-saves` flag places both consistently. `readAll`/`writeAll`
keep every key they do not recognise byte for byte, so the file is shaped to hold the twelve other
option names `TOWN-186` lists (`GameSpeed`, `FormationMode`, ...) without this story's own write
dropping them, though this story reads and writes only `TipsMode`. A missing store, a missing file or
a missing key all answer "tips shown" — `TOWN-186`'s own hardcoded default (`R1845`
initializes the flag to 1 with no read of prior state).

`FrontEnd.tipsOff` mirrors the store, read once by `LoadOptions()` (called by `cmd/againrom`'s own
wiring, once, after `Options` is set — a main-package concern, on `SaveSeams`'s own precedent, not
`NewFrontEnd`'s). `TipsOff()` reads the cached flag; `SetTipsOff(off)` updates the cache and persists
immediately. `ToggleTips()` (on `townScreen` and on `ui.Chargen`, through a `SetTipsOn func(bool)`
callback since `pkg/ui` may not cross the seam to a store itself) flips the flag; the effect is
tested at a room's own next entry (`loadTip`'s own gate), not against an already-open panel — toggling
does not retroactively hide or restore a panel already showing on screen.

**The generator's own "next entry" is `Back()`, not only a fresh `Chargen` (round-3 review, P-2).**
Every town room re-reads `TipsOff()` through `loadTip` on its own next entry, including a return from
a conversation; the generator has no room transition of that shape; its only re-entry into the
pre-create page is `Back()` from the detailed page. Before this fix `Chargen` read the suppression
gate once, at `ChargenSetup()` construction, and never again: turning tips off, going `Forward` then
`Back`, showed the panel again with its own gem drawn unlit — a widget contradicting its own toggle
state. `Chargen.tipSuppressed bool`, set by `Back()` to `!c.setup.TipsOn` and read by `TipPanel()`
alongside `tipClosed`, mirrors `loadTip`'s own room-re-entry read without giving the generator a
`loadTip` of its own: `Back()` is the generator's own room-construction-equivalent moment, the same
way a town room's next `Choose` is. Toggling mid-visit still does not retroactively hide or restore
an already-open panel, since `tipSuppressed` is set only on entry (`Back()`), never by `ToggleTips()`
itself — the same rule this behaviour already states for the town rooms.
`TestChargenTipSuppressionSurvivesForwardAndBack` (`pkg/ui/chargen_tip_test.go`) is the regression
witness: showing after a fresh setup; still showing (not retroactively hidden) immediately after
`ToggleTips()` turns tips off; not showing on `DetailedStage`; not showing after `Back()` (the fix);
still not showing after a second, symmetric `ToggleTips()` back on mid-visit (only the *next* entry
applies it); showing again after a second `Forward()`/`Back()` once tips are back on at that entry.

**The toggle is one-way from inside the game** (round-3 adversarial review, D-4). The gem is drawn
on the panel and hit-tested only through `TipPanelControlAt`, so it is reachable only while a panel
is showing; a panel shows only where `TipsOff()` was false at that room's own entry. The one
reversible moment is the panel that is still on screen when the player turns tips off, since toggling
does not retroactively hide it: clicking the gem a second time in that same panel turns them back on.
Once that panel is closed or its room left, no panel shows anywhere in the game, and there is no
in-game control that reaches `SetTipsOff(false)`. The player turns tips back on by editing or
deleting `options.txt`. Production behaviour matches the owner's own request as the contract states
it — a Close that makes the tip never appear again — and the store round-trips correctly in both
directions; what was missing is this sentence.

**This build does not write the Windows Registry** (`DIV-160`). The original writes `TipsMode` under
`HKEY_LOCAL_MACHINE\SOFTWARE\1C\Allods` (`TOWN-186`, High: both routines' own single callers,
`R1984` and `R1985`, call `RegOpenKeyExA((HKEY)0x80000002, "SOFTWARE\1C\Allods", …)`).
This build is a Go program that must build and run on a machine with no registry; the value goes to
the flat file instead, deliberately.

## Behaviour 5 — the shop's tip becomes the widget

Story 1011's `pkg/ui/shopscreen.go` drew the shop's tip as bare text (`drawShopTip`) at
`shopTipRect`, clipped above `shopMessageRect` to avoid bleed-through between unbordered glyphs and
the message strip's own short, centred text (`DIV-133`). `drawShopTip`, `shopTipInset` and the old
`ShopScreenView.Tip string` field are removed. `ShopScreenView.TipPanel ui.TipPanelView` replaces it,
built by `townScreen.ShopScreen()` as `t.tipView(roomShop, t.shopTip, ui.ShopTipRect())`.
`ComposeShopScreen` composes `ComposeTipPanel(dst, v.TipPanel)` **last**, after the drag icon —
opposite of the retired `drawShopTip`, which drew before the shop's own furniture and statistics
surface and so could already be overwritten with no gate of its own.

`ShopScreen()`'s `TipPanel` is keyed to the literal `roomShop` constant, not the live `t.room`:
`Choose(1)` loads the tip and then, if the shop has an offer available, opens
`openTownDialogue`, which reassigns `t.room = roomTalk`. `ShopScreen()` still resolves the panel
correctly regardless, since `roomShop` is the argument passed to `tipView`, not `t.room` itself.

`DIV-133`'s clip is re-typed rather than removed: `ShopTipRect()` keeps `shopTipRect`'s own 136-row
height (`SHOP-TIP-045`) and adds `tipPanelChromeH` below it, so the panel now extends **past**
`shopMessageRect`, not before it (`TestShopTipRectExtendsPastTheMessageStrip`). Since the panel draws
opaque and last, this is deliberate full occlusion of the message strip for the duration the panel is
open — not the bleed-through the original clip prevented, which no longer applies once the widget is
bordered and opaque rather than bare, transparently-drawn text. Once the panel is closed or
suppressed, the message strip composes pixel-identical to the build before this story
(`TestTipPanelOccludesTheMessageStripWhileShowing`, second half).

## Testing

Unit tests (no install): `pkg/game/tips_test.go`, `tipstore_test.go`, `tipart_test.go`,
`chargen_tip_test.go` (round 3 adds `TestChargenTipSuppressionSurvivesForwardAndBack`, Behaviour 4);
`pkg/ui/tippanel_test.go`, `chargen_tip_test.go`, `tippanel_overlap_test.go` (round-3 rewrite,
Behaviour 1's five enumeration tests); the shop's own updated `shopscreen_test.go`
(`TestShopTipRectExtendsPastTheMessageStrip`, `TestTipPanelOccludesTheMessageStripWhileShowing`).
Install-gated release tests (`AGAINROM_ASSETS` required, skip otherwise, golden rule 2):
`pkg/game/tippanel_release_test.go` — one test per screen
composing the real install through the production path, requiring the panel's own rect to differ
against the same frame with the panel forced off, and requiring at least one pixel of the tip body's,
the Close label's and the toggle label's own drawn colour inside their own rects (the specific check
that catches story 1017's "art paints, text does not" failure mode) — plus
`TestReleaseTipPanelTextsDrawWholeOnBothClasses`, asserting every one of the six shipped texts draws
whole with `ui.TipPanelFits`. All 6 release tests pass against both the `en` and `ru` preserved
installs.

`cmd/tippanelcheck` (new, round 3) is a committed instrument, not a test, on `cmd/buttonframecheck`'s
and `cmd/worldmapcheck`'s own precedent: it opens a real install through `pkg/game` and drives the
production town square, tavern, school, shop and chargen screens through `pkg/ui`'s own exported
interfaces and hit tests, and prints, per screen and per root, the rect, the minimum fitting height,
the spare rows at the chosen height, and the live-control pixel count under the rect — the table in
Behaviour 1 above, reproducible with `-assets <root>` or `AGAINROM_ASSETS`. It is registered in
`internal/archtest/dag.go`'s allow-map (`pkg/game`, `pkg/ui`).

Mutation tests, each reverted to a byte-identical restore after confirming the test reddened:
`chargenSize` on the border read in `LoadTipPanelArt` (`TestLoadTipPanelArtWrongSizedBorderFails`);
`paletteRGBA` forced to return nil (`TestLoadTipPanelArtReadsTheThreeShippedNodes`, after
strengthening the assertion to a typed-nil-safe check — `TipPanelArt`'s four fields are the
`image.Image` interface, so a typed-nil `*image.RGBA` produces a non-nil interface value that a plain
`== nil` comparison cannot see); `chargenTipPath` forced to always return the fighter path
(`TestChargenTipPathSelectsByClassStart`); `TipPanel()`'s stage guard forced to `false`
(`TestChargenTipPanelHidesPastPreCreate`); `TipPanelFits` forced to always return true
(`TestTipPanelFitsTellsAFullRectFromATruncatingOne`, added this story specifically because the
existing release test only ever exercised the true branch); round 3, P-1, each of `app.go`'s four
non-shop call sites individually reverted to the narrowed `consumed && kind != TipControlNone` guard
(`TestTavernTipPanelSwallowsEveryCellItCovers` and `TestSchoolTipPanelSwallowsEverySkillIconItCovers`
both redden for the shared `inSurface` site; `TestTownSquareTipPanelSwallowsEveryControlItCovers`
alone for the town-square branch; `TestShopTipPanelSwallowsEveryControlItCovers` alone for `inShop`,
confirming it is still wired even though it never carried the narrowed guard;
`TestPreCreateTipPanelSwallowsEveryPortraitItCovers` alone for `stepPreCreate`) — each mutation
reddens exactly its own screen's test and no other, closing the regression class by enumeration
rather than by site; round 3, P-2, `Chargen.Back()`'s `c.tipSuppressed = !c.setup.TipsOn` line
removed (`TestChargenTipSuppressionSurvivesForwardAndBack` reddens with "TipPanel() after Forward and
Back, tips off, is Showing()").
