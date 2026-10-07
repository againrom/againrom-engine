# 0157 round 3 — closure

Pipeline v2 closure evidence for `contract.md`. The behavioural contract is
`spec.md`, canonicalized to as-built at this landing.

## What landed, against the four directives

**1. No black regions.** Pure black is cleared to zero alpha on every 24-bit
bitmap this screen blits (`loadShopBMP`, `keyBlack`). That is what takes the
price digits off their black boxes and the black corners off the four button
plates. The digits are centred on the plaque's own ink rather than on its 60x10
canvas, so a three-digit price sits on the tablet instead of beside it. A place
with no element paints nothing in the shelf grid and on the table, so the shelf
rack and the table wood show through; a place with no element in the backpack
strip keeps `backinvg.bmp`, because that region has no picture of its own. The
money element is built: the strip's first element draws `backinv.bmp` with
`graphics\interface\money\money.16a` composited over it and prints the purse.

**2. The character block.** The region (480,238)-(640,480) draws the shown
member composed in his own equipment as a 160x240 canvas at the panel's left and
two rows down, with the two decoded 32x32 picker rectangles at (481,443,513,475)
and (599,443,631,475) and his name between them. One press steps the member with
wrap-around and, in the same call, binds the backpack strip to that member's
container, returns the strip to its first place, and makes the cell backgrounds
ask that member's class.

**3. The wheel turns the region under the cursor.** Over the shelf grid it pages
the rack by a whole row of two; over the backpack strip it moves one place;
elsewhere, the table included, it moves nothing.

**4. The garbled glyphs.** Root cause found and fixed at the source, not at the
symptom - see below.

## The garbled-glyph root cause

The hypothesis handed to this lane was CP866 bytes drawn through a wrong decode
path. That is refuted: the install's own strings arrive as code-page bytes and
are drawn correctly. `cmd/shopdump -hover` on the Russian root prints the shelf
item's name as `91E2A0ABECADA0EF2098A8AFAEA2A0ADADA0EF2084E3A1A8ADA0`, valid
CP866, and the composed PNG shows it as `Стальная Шипованная Дубина`.

The actual cause is the opposite direction. This project's fonts are the game's
own bitmap atlases and they are indexed BY BYTE: `render/text`'s `Font.walk`
steps one byte at a time and each byte selects a record through the install's
code page. A Go source literal is UTF-8, so the em dash in this build's own
message strings is three bytes - E2 80 94 - and the atlas draws three unrelated
glyphs. On the English install those are the CP437 shapes at those codes: `Γ`,
`Ç`, `ö`. The `Çö` in the owner's screenshot is the tail of that.

Twenty-one literals across eight files in two packages carried an em dash. The
eight are `formats/bmp/bmp.go` and, in `pkg/game`, `frontend.go`,
`originalsave.go`, `resume.go`, `shoproom.go`, `shopview.go`, `townscreen.go`
and `world.go`. Three of the twenty-one are the shop-screen messages: the
shelf-selection message (`shopview.go`), the unpriced-item message
(`shoproom.go`) and the town header (`townscreen.go`). The conversion took two
commits: `ac7592a` converted thirteen, `ad01fdb` the remaining eight. A further
five literals in `game/continuity_test.go` were converted in the same story;
they are outside the scan's population, which excludes `_test.go` files. All are
now hyphens, and `internal/archtest`'s new drawn-text scan
fails on any string literal under `pkg/` holding a byte above 0x7f. The scan
reads parsed syntax, so the em dashes in the same files' prose are not findings;
the live-tree test asserts that such prose still exists, otherwise the difference
between this check and a grep would stop being witnessed.

## Twelve-aspect matrix

| Aspect | Verdict | Evidence |
|---|---|---|
| Data | PASS | `money.16a` resolves at 80x80 on both roots (`shopdump` prints the composed coin); the black key is applied at the one load site, `loadShopBMP`, and `.256`/`.16a` sprites keep their own transparency |
| Runtime state | PASS | `townScreen.shopMember` and `townScreen.shopFigures`; both reset on entering the room, both clamped on every read, neither reaches `pkg/sim` |
| Simulation | N/A | `git diff master..HEAD -- pkg/sim/` is empty; `formatVersion` is 50 |
| Player input | PASS | `ShopControlPickerPrev`, `ShopControlPickerNext`, `ShopControlMerchant` are hit-tested and dispatched; the wheel resolves through `ShopWheelAt` in `pkg/ui/app.go`'s shop arm, from the same `WindowToFrame` point a click uses, and `TestTheWheelCrossesTheSeamWithTheRegionUnderThePointer` drives that seam rather than only the region function |
| AI | N/A | Nothing acts on this screen |
| UI / HUD | PASS | The composed frame on both roots, below |
| Triggers / scripts | N/A | No script reaches the shop; the census is unchanged, below |
| Inventory / equipment | PASS | `shopPackItems`/`setShopPackItems` read and write the shown member's container; a purchase made with the second member shown lands in the second member's container (`TestBuyingLandsInTheShownMembersOwnContainer`); the doll is composed from that member's own `Worn`/`Carry.Equipped` |
| Persistence / save-load | N/A | No serialized field changes. The shown-member index is presentation and is not written, exactly as the shelf and pack bases are not |
| Campaign / session | PASS | The witness runs a real campaign session: `NewFrontEnd`, `FinishMission` on the mission the registry marks as offering the town, the square's own SHOP door, the merchant's greeting paged to its end |
| Shipped content | PASS | Both installs, same code path; `SHOP-SCREEN-038` establishes no shop-screen graphic is localised, and the run confirms identical geometry and identical black coverage in the button panel and the strip on both roots |
| Interactions with existing mechanics | PASS | The wear rule now asks the shown member instead of the first; the town dialogue's speaker figures and this panel share one composition pass; the drawn-text rule covers every string in the build, not only the shop's |

No known in-scope GAP.

## Self-review before the push

Three findings of the kind the adversarial reviewer hunts, found and closed
before the branch was pushed a second time.

`ShopScreenView.Offer` was produced and not consumed: the field said whether the
merchant had work and nothing drew it, so the one control with no decoded art
was also the one with no way to be found. His box is now outlined in the chosen
shelf's own colour while he has work (`TestAMerchantWithWorkIsMarked`).

The overlap test still excused `button|shelf pick`. That allowance belonged to
the round-2 build's draw rectangles, which reached x 445; the hit rectangles stop
at 459 and the buttons start at 483, so the pair is now asserted disjoint rather
than excused, and the merchant and the two picker rectangles joined the set.

The wheel's region rule was tested at `ShopWheelAt` and at `ShopScroll`, with the
`pkg/ui` seam between them untested. It is now driven through `flow.scrollShop`
against the fake shop seam, which is the call `pkg/ui/app.go` makes.

## Integration witness

`cmd/shopdump` is the witness. It builds a real campaign session and reaches the
shop the way a player does, then composes the screen through the same
`ui.ComposeShopScreen` the window calls. It never opens a window.

Run the owner's way, from the worktree:

```
$env:AGAINROM_ASSETS = '<seat>\gameversions\en'
go run ./cmd/shopdump -shelf 1 -hover "40,150" -png <dir outside the repo>
```

English root:

```
town opened after mission 20, chapter 30, purse 600
message: "weapons - 100 on the shelf"
member 1 of 2: Danath
shelf 1 (weapons), purse 600 buy 0 sell 0 total 0
figure: 160x240
  shelf[0] back=2 money=false count=1 price=333 icon=80x80
  shelf[1] back=1 money=false count=1 price=800 icon=80x80
  shelf[2] back=2 money=false count=1 price=384 icon=80x80
  shelf[3] back=3 money=false count=1 price=167 icon=80x80
  shelf[4] back=2 money=false count=1 price=225 icon=80x80
  shelf[5] back=2 money=false count=1 price=200 icon=80x80
  pack[0] back=0 money=true count=600 price=0 icon=none
hover (40,150): Steel Spiked Club | Damage 5-11 | To-hit 1  Defence 0
black pixels: screen 38515/307200 (12%), buttons 6675/41888 (15%),
              pack 11212/43200 (25%), corner 101/38720 (0%)
```

Russian root, same command with the ru root: identical stock, identical
geometry, identical black coverage in every region, and the hover reads
`Стальная Шипованная Дубина` with the same three characteristic lines. The
message line reads `weapons - 100 on the shelf` on both, with no garbage glyph.

The composed state is what the matrix asks for: the shown member is named, the
picker's position is `1/2`, the six shelf cells carry three different background
classes (2 = affordable, 1 = usable, 3 = the class cannot use it), the plaques
are selected by digit count, and the strip's first place is the money element
carrying the purse.

Stepping the picker once (`-steps 1`) shows `Reniesta`, `2/2`, a different doll,
and the second member's own container in the strip; the shelf backgrounds change
with her class.

## Pure-black coverage, before and after

Measured by `cmd/shopdump`'s own count over the composed frame, English root,
NO SHELF OPEN, which is the state the room is entered in and so the state both
readings share. "Before" is master `f28da48`; "after" is this branch. With one
shelf open the after reading is 12%, because six cells then carry a background
and an icon; the before run could not open a shelf, so that pair is not
comparable and is not tabulated.

| Region | Before | After |
|---|---|---|
| whole screen | 21% | 9% |
| button panel (464,0)-(640,238) | 20% | 15% |
| backpack strip (0,390)-(480,480) | 25% | 25% |
| bottom-right corner (480,238)-(640,480) | 88% | 0% |

Two readings need naming. The button panel's remaining 15% is the shipped art's
own dark-red plaque texture, whose channels are below the count's threshold of
16; the black bands the owner saw are gone, which the cropped PNG shows and the
number does not. The strip's 25% is unchanged and is deliberate: that region has
no picture of its own and the shop view fills it flat with colour zero
(`SHOP-VIEW-044`), so the five cells stand on the original's own black. Spec D-9
and ledger row DIV-016 carry it.

## Script-gap census

Unchanged, and this round was not meant to move it. `pipeline/check-milestone.sh`
on this branch prints `ok  the script gap and the drive are where they were
recorded, both roots`. Mission 10 and mission 20 each report zero unsupported
script nodes through `cmd/missionrun -trace -ticks 1`, which is what
`pipeline/milestone-baseline.txt` records. The result this round owes is the
second kind: what `builds/current/` shows.

## Reconciliation against research

| Claim | Reconciled how |
|---|---|
| `SHOP-SCREEN-034` | Its published four rectangles are no longer used as hit rectangles. This build used them that way until this round, which is the consumer error the retraction names; they are now `SHOP-SHELF-047`'s hit array, and the draw array is used only to mark the chosen shelf |
| `SHOP-SCREEN-036` | All five arms are now reachable: the money arm (a) is built, the quantity-zero arm (b) is the strip's empty place, (c) and (d) and (e) were already the three background classes |
| `SHOP-SCREEN-037` | The plaque is chosen and right-aligned as decoded; the FIGURE's placement on it is this build's own, because the claim gives the bitmap's alignment and not the text's |
| `SHOP-MONEY-048` | Consumed. The coin is loaded through the item icons' own `.16a` reader, which carries the per-pixel level into alpha (`SPR16A-ALPHA-025`) |
| `SHOP-FIGURE-041` | The rectangle and its derivation are reproduced. The re-parenting is not: see DIV-017 |
| `SHOP-FIGURE-042` | The 160x240 canvas and the `+0`, `+2` blit are reproduced through this build's own equipment compositor |
| `SHOP-PICKER-043` | Both rectangles, the wrap at both ends, and the strip rebinding in the same call are reproduced. The art is not decoded: DIV-015 |
| `SHOP-MERCHANT-046` | His box is used as a control. The original opens its merchant dialogue elsewhere; this is disclosed as spec D-7's neighbour and is an addition, not a decode. The claim's draw half is not consumed: neither the static picture `movies\shopanim\Pose2-3\1.bmp` at (277,112) nor the Yes and No poses that replace it are drawn, so the region is empty. Recorded as DIV-020 |
| `SHOP-VIEW-044` | The flat fill under the strip is kept rather than painted over |
| `SHOP-LIMIT-049` | The seam is not reproducible in this build's screen structure: DIV-017 |

## Gate

On the pushed tree, from the worktree:

```
go build ./...                                   clean
go vet ./...                                     clean
gofmt -l $(git ls-files ... '*.go')              prints nothing
go test -trimpath -count=1 ./...                 all packages ok
bash scripts/check-no-game-assets.sh             clean (tree scan)
```

`git log --format='%h %(trailers:key=Co-Authored-By)' master..HEAD` prints the
hashes with an empty trailer column: no commit on this branch carries one.
