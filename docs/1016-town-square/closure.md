# 1016 — closure

As-built. What was measured, what witnesses it, what remains open.

## The twelve aspects

| Aspect | Verdict | Evidence |
|---|---|---|
| Data | PASS | `LoadTownSquareArt` (`pkg/game/townsquareart.go`) reads and size-asserts six shipped nodes: `townmain.bmp` 640x480, `town_add.bmp` 552x92, `townmask.bmp` 640x480, `shop_l.bmp` 52x76, `tavern_l.bmp` 28x64, `trener_l.bmp` 140x116. No format change, no new archive kind. `TestLoadTownSquareArtReadsTheCompletePicture`, `TestLoadTownSquareArtMisSizedNodeIsRefused` |
| Runtime state | N/A | No new persisted or session field. `TownScreen.TownSquareView()` reads existing `FrontEnd.TownSquareArt`, itself loaded once at start-up |
| Simulation | N/A | `pkg/sim` is untouched. No hashed field, no serialized form, no `formatVersion` change |
| Player input | PASS | FR-2, the raster hit test: `TestTownSquareControlAtReadsTheFiveSignificantCodes`, `TestTownSquareControlAtRefusesAnUnlistedCode`, `TestTownSquareRasterMaskClicksChooseTheSameDoorIndexAsTheGrid`, and `cmd/townsquarecheck`'s production drive, end to end from a pixel to `Choose(i)`'s own room transition. Keyboard input (wheel, arrows, Enter) at the square once art is ready: round-2 adversarial review P-1 found these unguarded, silently moving and choosing a `townList` selection `ComposeTownSquare` draws no cursor for. Fixed by gating `stepTownAt`'s whole wheel/arrow/Enter switch on `townSquareView`'s own readiness, the shop's own precedent (the comment above `inShop`, `app.go`); Escape is unaffected, read before this dispatch everywhere. `TestTownSquareKeyboardSelectionIsInertOnceArtIsReady`. The guard itself broke a second, unshipped-facing path: `HeadlessActivate`'s generic Picker branch reaches a named row through `headlessChooseRow`'s own Down/Up/Enter simulation, which the new guard also makes dead. `HeadlessActivate` now resolves a town-square door directly through `townList.Select` + `chooseTown` once art is ready (`headlessChooseTownSquareRow`, `pkg/ui/headless.go`), the same call sequence a mask hit makes, never through key simulation. `TestHeadlessActivateChoosesTheSquareDoorByNameOnceArtIsReady` |
| AI | N/A | Not reached |
| UI / HUD | PASS | The whole subject. `TestTownSquareArtDrawsThePictureInsteadOfTheGrid`, `TestComposeTownSquareDrawsEachLabelAtItsOwnOrigin`, `TestComposeTownSquarePaintsNoHintText`, `TestComposeTownSquareWithNoArtIsBlank`. Round-3 adversarial review (P-2, mandatory fix) found `ComposeTownSquare` painting `town_add.bmp` with `draw.Src`, opaque and unconditional: the overlay is 40.7% pure black (20644/50784 pixels), the sky above the rooftops its own footprint does not cover, and `townmain.bmp` carries none (0/307200) — every one of those 20644 pixels landed as a black band over the base picture's own sky. Fixed by keying the overlay at load (`keyBlack`, `LoadTownSquareArt`) and drawing it with `draw.Over`; the three door labels carry no pure-black pixel (measured, zero across all three) and are left unkeyed and `draw.Src`, unchanged. Install-gated witness: `pkg/game/townsquare_release_test.go`, `TestReleaseTownSquareOverlayDoesNotPaintBlackOverTheSky` and `TestReleaseTownSquareOverlayIsKeyedAtLoad` (mutation result under "The integration witness", below) |
| Triggers / scripts | N/A | No script opcode is touched. Measured below (missionrun's `UNSUPPORTED` count), not assumed |
| Inventory / equipment | N/A | Not reached |
| Persistence / save-load | PASS, narrowly | FR-5 opens the pre-existing save/load mini-menu; no new save field. `TestTownSquareStatueClickOpensTheMiniMenu` |
| Campaign / session | N/A | `Rows()` and `Choose(i)` are unchanged; the four doors and the gate already existed as row choices before this story |
| Shipped content | PASS | Every offset, fraction and mask code is measured against both preserved installs and reported identical (`cmd/townsquarecheck`, EN and RU roots). Round-3 review (W-5) found the tool's own `code N count M box B` lines printed in Go map order, which varies run to run, so the two roots' output could not be diffed as claimed. `cmd/townsquarecheck/main.go` now sorts those lines by code before printing (`sort.Slice`); three consecutive runs on one root now print byte-for-byte identical output, and the EN and RU roots' own output now differs in exactly one line, the printed root path — see "The two roots' output", below |
| Interactions with existing mechanics | PASS | The raster click feeds `townList.Select(c.Door)` then `chooseTown()` — the identical seam the row-button grid always used — so `Rows()`/`Choose(i)` are unmodified. `pipeline/check-scenarios.sh` selects and runs the same 12 scenarios (see Gate, below). This was checked, not assumed, after the P-1 guard landed: 5 of 12 broke (the four naming a town-square door or an NPC reached through one, plus one that hung inside `headlessChooseRow`'s own now-unsatisfiable Down loop), all four traced to the same cause — `HeadlessActivate` reaching a row through key simulation the guard now suppresses — and the `HeadlessActivate` fix above (`headlessChooseTownSquareRow`) restored 12 of 12 |

No aspect is GAP.

## The integration witness

`cmd/townsquarecheck` opens a real campaign town through `game.NewFrontEnd`, reads the square's own
art through the production `TownSquareArtScreen` interface, resolves each of the five mask regions
through production's own `ui.TownSquareControlAt`, and — for the four doors — feeds the resolved
index to production's own `Choose(i)`, reading the resulting room back from `Header()`, a call this
loop never fed the index into.

```
$ go build -o <outside-repo>/townsquarecheck ./cmd/townsquarecheck
$ <outside-repo>/townsquarecheck -assets <root>/gameversions/en -png <outside-repo>/townpng
```

On both `gameversions/en` and `gameversions/ru` (identical output on both):

```
hit-test code 128 at (167,368) -> kind 1 door 0, want kind 1 door 0 ok
hit-test code 144 at (293,305) -> kind 1 door 1, want kind 1 door 1 ok
hit-test code 160 at (200,155) -> kind 1 door 3, want kind 1 door 3 ok
hit-test code 176 at (373,353) -> kind 2 door 0, want kind 2 door 0 ok
hit-test code 192 at (503,369) -> kind 1 door 2, want kind 1 door 2 ok
drive code 128 door 0 Choose -> header "the tavern - chapter 30    [esc: back]" contains "tavern" ok
drive code 144 door 1 Choose -> header "shop - chapter 30    [esc: back]" contains "shop" ok
drive code 192 door 2 Choose -> header "the school - chapter 30    [esc: back]" contains "school" ok
drive code 160 door 3 Choose -> header "the gates - chapter 30    [esc: back]" contains "gates" ok
drive statue at (373,353) -> kind 2, want menu ok
townsquarecheck: ok
```

(Anchors moved from round 1's own quoted run — `(267,268)`, `(392,296)` — to the coordinates above at
`W-1`'s fix, below. Every line still reads `ok`.)

Mutation result for the mask-code witness. Reverting the gate door mapping in `TownSquareControlAt`
from `Door: 3` to a wrong door index (mutated to `Door: 2`, colliding with the school) reddens both
`TestTownSquareControlAtReadsTheFiveSignificantCodes` and `TestTownSquareGateClickEntersThroughChoose`.
The file was restored from a backup taken before the mutation; `go test ./pkg/ui/...` was rerun
clean afterward.

Mutation result for the P-1 keyboard guard (`TestTownSquareKeyboardSelectionIsInertOnceArtIsReady`).
Neutralizing the guard's own condition (`townSquareView(a.flow.town); ready` mutated to
`townSquareView(a.flow.town); false && ready`, so dispatch always falls through to the wheel/arrow/
Enter switch exactly as it did before this fix) reddens the test:

```
townsquare_test.go:200: chosen = [3], want none: Down/Wheel/Enter moved or chose a selection
the composed picture draws no cursor for
```

Two Down presses moved the picker's own selection to index 3 (starting at 0, over
`fakeSquareArtTown`'s four rows) and Enter chose it — reproducing P-1 exactly. The file was restored
from a backup taken before the mutation; `go build ./...` and the full `pkg/ui` suite were rerun
clean afterward.

Mutation result for the `HeadlessActivate` companion fix
(`TestHeadlessActivateChoosesTheSquareDoorByNameOnceArtIsReady`). Neutralizing the new branch's own
condition (`townSquareView(a.flow.town); ready` mutated to
`townSquareView(a.flow.town); ready && false`, routing every town-square `HeadlessActivate` call back
through the generic Picker path) originally produced no clean red at all: `headlessChooseRow`'s own
`for p.Selection() < target { HeadlessKey("down") }` loop looped forever once Down no longer advanced
the selection — the same mechanism that made `scenarios/1005-doll-and-shop.json` appear to hang for
several minutes during round 2's own first `check-scenarios.sh` run, before the cause was traced to
that round's own fix rather than to the scenario. `pipeline/check-scenarios.sh` sets no per-scenario
timeout, so a scenario reaching this loop unbounded stalls the gate rather than reporting it
(round-3 review, W-4).

`headlessChooseRow` is now bounded, on `headlessActivateWorldMap`'s own precedent
(`pkg/ui/headless.go:382`): each direction runs at most `len(p.Rows())+1` key presses, then reports an
error naming the row it stopped at, rather than looping. Repeating the same mutation under a
30-second timeout now fails in 0.01s:

```
townsquare_test.go:233: HeadlessActivate("GATES") = headless row 3: production controller stopped at 0
after 5 key press(es) (the picker's own row bound); Down/Up may be inert on this screen, want nil
```

The file was restored from a backup taken before the mutation; `go build ./...` and the full `pkg/ui`
suite were rerun clean afterward.

Mutation result for the P-2 overlay fix
(`TestReleaseTownSquareOverlayDoesNotPaintBlackOverTheSky`, install-gated,
`pkg/game/townsquare_release_test.go`). Reverting `ComposeTownSquare`'s overlay draw from `draw.Over`
back to `draw.Src` (`pkg/ui/townsquare.go`) reddens the test against the real EN install:

```
townsquare_release_test.go:51: composed town square: 20644 pixel(s) are pure black where the base
picture is not, want 0 (the overlay's own transparent surround painted opaque)
```

The count matches the review's own figure exactly. The sibling test,
`TestReleaseTownSquareOverlayIsKeyedAtLoad`, stays green under this mutation: it reads
`LoadTownSquareArt`'s own output directly, which this mutation does not touch, confirming the two
tests watch distinct, correctly isolated concerns rather than one masking the other. The file was
restored from a backup taken before the mutation; `go build ./...` and both tests were rerun clean
afterward.

A before/after PNG pair, composed from the real EN install through the production
`ComposeTownSquare` (before: the overlay drawn `draw.Src`, unkeyed, reproducing the pre-fix picture
by restoring the opacity `keyBlack` clears, RGB untouched; after: the shipped, fixed code path) was
written outside both repositories for the owner, at
`againrom/review/1016-town-square-proof/townsquare-before-P2.png` and
`townsquare-after-P2.png` in the same directory. The before image shows a black band across the top
of the square, over the sky behind the rooftops; the after image shows the sky.

One limit of the drive: the started campaign's own party has one member, so nothing distinguishes a
one-member party's own `Rows()` content across the four doors — the check is that `Choose(i)` moves
the room to the door named, not what that room then shows. What each room shows once entered is
built and witnessed by its own earlier story (0157 shop, 1015 school, the tavern's own story), not
this one.

## What the two instruments measured

Instrument A, correlation (slide a patch over the base picture, count opaque pixels matching within
`matchTolerance=12`, report the winning fraction against the next-best anywhere else in the search
window):

| Patch | Winning offset | Fraction | Next-best |
|---|---|---|---|
| `town_add.bmp` | `(0,0)` | `1.0000` | `0.1903` at `(1,5)` |
| `shop_l.bmp` | `(264,264)` | `0.7715` | `0.1546` at `(264,269)` |
| `tavern_l.bmp` | `(144,332)` | `0.5893` | `0.1618` at `(144,338)` |
| `trener_l.bmp` | `(436,300)` | `0.6884` | `0.0951` at `(434,295)` |

Instrument B, the raster mask's own colour codes (more than 1000 pixels each, background code 0
excluded): 128, 144, 160, 176, 192 — five, matching the tip text's five named regions. Each label's
own dominant underlying mask code: `shop_l` → 144 (3883/3952, 98.3%), `tavern_l` → 128 (1783/1792,
99.5%), `trener_l` → 192 (13210/16240, 81.3%).

The two instruments and production agree everywhere `cmd/townsquarecheck` checks them against each
other: instrument A's winning offsets equal `ui.TownSquareAddOrigin` and `ui.TownSquareLabelOrigin`;
instrument B's label-to-code table equals the code-to-door table production's own
`ui.TownSquareControlAt` answers, read over the install's own mask loaded through
`game.LoadTownSquareArt` rather than through the tool's own decoder; `LoadTownSquareArt`'s own
`Background` is pixel-identical to the tool's independent `bmp.Decode` of the same file.

## The two roots' output

Round-3 review (W-5) found `cmd/townsquarecheck`'s `code N count M box B` lines printed by ranging a
Go map, so three consecutive runs on one root printed the six lines in three different orders; every
measured VALUE matched between the EN and RU roots, but the WORDING did not, and the two roots'
output could not actually be compared with `diff` as the earlier closure claimed. The tool now
collects the codes, sorts them (`sort.Slice`), and prints in that order. Three consecutive runs
against `gameversions/en` now print byte-for-byte identical output, and the EN and RU roots' own
full output now differs in exactly one line, the printed root path:

```
$ go build -o <outside-repo>/townsquarecheck ./cmd/townsquarecheck
$ <outside-repo>/townsquarecheck -assets <root>/gameversions/en
root <root>/gameversions/en
townmain.bmp 640x480 black 0
townmask.bmp 640x480
distinct codes 150
code   0 count 266003 box (0,0)-(640,480)
code 128 count   5764 box (127,93)-(400,413)
code 144 count   5771 box (254,261)-(397,395)
code 160 count   7911 box (156,92)-(570,424)
code 176 count   5592 box (342,289)-(407,420)
code 192 count  14733 box (418,299)-(584,426)
town_add.bmp 552x92 black 20644
town_add best (0,0) frac 1.0000 (30140 total) next 0.1903 at (1,5) production (0,0) ok
shop_l 52x76 black 0
shop_l best (264,264) frac 0.7715 (3952 total) next 0.1546 at (264,269) production (264,264) ok
shop_l dominant mask code 144 (3883 of 3952 px) want 144 ok
tavern_l 28x64 black 0
tavern_l best (144,332) frac 0.5893 (1792 total) next 0.1618 at (144,338) production (144,332) ok
tavern_l dominant mask code 128 (1783 of 1792 px) want 128 ok
trener_l 140x116 black 0
trener_l best (436,300) frac 0.6884 (16240 total) next 0.0951 at (434,295) production (436,300) ok
trener_l dominant mask code 192 (13210 of 16240 px) want 192 ok
production Background matches this tool's own decode: ok
hit-test code 128 at (167,368) -> kind 1 door 0, want kind 1 door 0 ok
hit-test code 144 at (293,305) -> kind 1 door 1, want kind 1 door 1 ok
hit-test code 160 at (200,155) -> kind 1 door 3, want kind 1 door 3 ok
hit-test code 176 at (373,353) -> kind 2 door 0, want kind 2 door 0 ok
hit-test code 192 at (503,369) -> kind 1 door 2, want kind 1 door 2 ok
self-check: label shop_l's own code 144, this tool's expect table door 1, want door 1 ok
self-check: label tavern_l's own code 128, this tool's expect table door 0, want door 0 ok
self-check: label trener_l's own code 192, this tool's expect table door 2, want door 2 ok
drive code 128 door 0 Choose -> header "the tavern - chapter 30    [esc: back]" contains "tavern" ok
drive code 144 door 1 Choose -> header "shop - chapter 30    [esc: back]" contains "shop" ok
drive code 192 door 2 Choose -> header "the school - chapter 30    [esc: back]" contains "school" ok
drive code 160 door 3 Choose -> header "the gates - chapter 30    [esc: back]" contains "gates" ok
drive statue at (373,353) -> kind 2, want menu ok
townsquarecheck: ok
```

The `self-check:` lines (round-3 review, D-3) compare two tables this tool itself authors —
`labelDomCode`, measured by correlation above, against `expect` and `wantDoor`, hand-written
constants in `cmd/townsquarecheck/main.go` — and never call `ui.TownSquareControlAt` or any other
production code. The `hit-test` lines above them are the production check: they call
`ui.TownSquareControlAt` against the real, install-loaded `townmask.bmp`. Under a mutation of
`TownSquareControlAt`'s own tavern arm, the `self-check` lines stayed `ok` while the `hit-test` lines
failed, so the `self-check` lines are relabelled to read as what they are rather than as a second
production check; production's own code-to-door mapping is covered by the `hit-test` lines alone.

## Control for the correlation method

A bounding-box centre is not a safe anchor for an irregular region: the gate (code 160) and tavern
(code 128) regions are thin, arch-shaped outlines, and their own bounding boxes are large relative to
their own pixel population (160: box 414x332 = 137448px, only 7911 of it code 160; 128: box 273x320 =
87360px, only 5764 of it code 128). The tool's first version used the raw bounding-box midpoint and
that midpoint landed on a different code for both regions (`kind 0`, no hit, at both). Replacing the
anchor with the region's own pixel nearest the bounding-box centre (`codeAnchor`,
`cmd/townsquarecheck/main.go`) fixed both; the other three regions' anchors were unaffected by the
change, since their own shapes are close enough to convex that the midpoint already fell inside them.
This is recorded because it found nothing wrong in production — the defect was in the probe's own
first anchor choice, not in `TownSquareControlAt` — and the correction is why the tool's numbers in
this document differ from an earlier local run that reported two FAILs at the hit-test step.

**Second round.** The nearest-bounding-box-centre anchor above is still not safe: for an irregular
region a straight-line nearest pixel can land inside a small isolated fleck of the same colour code
rather than inside the region's own visible outline. A 4-connected flood fill of `townmask.bmp`
over each of the five codes, run against both preserved installs, found the mask itself carries this
noise: code 128 (tavern) is 15 connected components, 14 of them single pixels; code 160 (gate) is
17 components, 16 of them single pixels; code 144 (shop) is 5 components, 4 of them single pixels,
3 of those 4 landing inside the statue's (code 176) own bounding box. `codeAnchor` was changed again:
it now flood-fills the code, keeps only the largest component by pixel count, and anchors on that
component's own pixel nearest its centre (`floodFillCode`, `cmd/townsquarecheck/main.go`). This
moved three of the five anchors (tavern `(267,268)` → `(167,368)`; shop `(325,328)` → `(293,305)`;
gate `(392,296)` → `(200,155)`) and left the other two materially unchanged (statue `(374,354)` →
`(373,353)`; school `(501,362)` → `(503,369)`). All five still resolve to the same door or menu kind
on both installs (the "integration witness" block above).

The anchor now guarantees only that it sits inside the region's own largest visible blob, nearest
that blob's own centre. It does not guarantee the blob is the only place a player's click resolves
to that code, and it does not guarantee every pixel inside the blob is itself free of same-code
noise from a neighbouring region — `DIV-148`'s addition on the code-144 strays inside the statue's
box is exactly this residual, and it is a property of the shipped mask, not of the anchor choice.

At the story's first candidate no claim published the town square's mask-code-to-room mapping. The
pin bump added `TOWN-161`, `TOWN-163` and `TOWN-164`; read together, they settle all five identities
and the exact reject rule, independently of the story's art and tip-text measurements. A 2026-08-24
seat audit therefore closes `DIV-148`. `DIV-149` remains open for the overlay's unconditional
composition, and `DIV-150` remains open because this build draws all three labels without the
original's state gate.

`DIV-151` and `DIV-152` are returned unused. This story's own findings did not surface a third and
fourth distinct technical fact beyond `DIV-148`, `DIV-149` and `DIV-150`.

## The census and the game

```
$ go build -o <outside-repo>/mr ./cmd/missionrun
$ for m in 10 20; do AGAINROM_ASSETS=<root>/gameversions/en <outside-repo>/mr -mission $m -trace -ticks 1 | grep -c UNSUPPORTED; done
```

Mission 10: `0`. Mission 20: `0`. `pipeline/milestone-baseline.txt` carries no `cannot run` line for
either mission on either root, so the count is unchanged, as the contract predicted: this story
touches no script opcode.

## Gate

```
go build ./...                                                       clean
go vet ./...                                                         clean
gofmt -l $(git ls-files --cached --others --exclude-standard '*.go') clean (no output)
go test -trimpath -count=1 ./...                                     ok, 55 packages (39 ok, 16 `[no test files]`)
bash scripts/check-no-game-assets.sh                                 check-no-game-assets: clean (tree scan)
```

`internal/archtest`'s `TestLiveTreeClean` required registering `cmd/townsquarecheck` in the DAG
allow-map (`internal/archtest/dag.go`), on `cmd/schoolcheck`'s own precedent: `pkg/game`, `pkg/ui`,
`pkg/formats/bmp`, `pkg/render/terrain`.

```
$ AGAINROM_IMPL=<this worktree> bash pipeline/check-scenarios.sh <root>/gameversions/en
check-scenarios: selected 12 scenario(s)
check-scenarios: ok (12 of 12)
```

Five of the 12 scenarios activate a town-square row by name — `0152-save666.json` and
`0163-mission-to-town.json`'s own "TAVERN", `1013-world-map-one-click.json`'s own "TAVERN" and
"GATES", `1005-doll-carry-over-worn.json`'s own "TAVERN", and `1005-doll-and-shop.json`'s own
"SHOP". `App.HeadlessActivate` resolves the target against `townList.Rows()`'s own text. Round-1's own closure claimed this "does not go through `stepTownAt`", read from
`HeadlessActivate`'s own `ScreenTown` branch calling `headlessChooseRow` directly. That reading
missed one level: `headlessChooseRow` reaches the target row by simulating Down/Up presses through
`HeadlessKey`, which drives `App.step` exactly as a live keyboard does, and `App.step` **does** reach
`stepTownAt`. Landing the P-1 guard on that premise, unverified, broke all five scenarios above, two
different ways depending on the target row's own index. "TAVERN" is `townList` row 0, its own
starting selection, so `headlessChooseRow`'s "down until reached" loop runs zero iterations and goes
straight to the also-dead Enter: `HeadlessActivate` returns no error, but `chooseTown` never fires,
so the scenario's own NEXT step fails with a wrong-screen error (`0152-save666.json`,
`0163-mission-to-town.json`, `1005-doll-carry-over-worn.json`, `1013-world-map-one-click.json`).
"SHOP" is row 1, one Down press away, and that Down is also dead, so
`for p.Selection() < target { HeadlessKey("down") }` never terminates
(`1005-doll-and-shop.json`, a hang rather than a failure). Rerunning `check-scenarios.sh` against the
guard, before writing the fix, is what surfaced this: 5 of 12 failed, all reducible to the same
cause. `HeadlessActivate` now special-cases the town square once art is ready
(`headlessChooseTownSquareRow`, `pkg/ui/headless.go`): it resolves the target the same way and then
selects the row directly (`townList.Select` + `chooseTown`), the identical call sequence a mask hit
makes, never through key simulation — so it again does not go through `stepTownAt`, for the reason
stated, not the reason first written. The raster-mask click path itself is witnessed separately, by
`cmd/townsquarecheck`'s production drive (above) and by `pkg/ui/townsquare_test.go`'s click tests
against synthetic fixtures; no headless scenario drives a mouse-coordinate click at the square.

```
$ AGAINROM_IMPL=<this worktree> bash pipeline/check-release-tests.sh <root>/gameversions/en
check-release-tests: selected 18 install-gated tests, by variable:
         15 AGAINROM_ASSETS
          1 AGAINROM_ORIGINAL_SAVES
          2 AGAINROM_SAVE_666
check-release-tests: ok (18 of 18 install-gated tests ran and passed, 0 skipped)
```

`TestReleaseTemporaryPartyAndNonPartyKeepTheirOwnPresentation` (`pkg/game`) is fixed
(`pipeline/check-release-tests.sh`'s own header names the cause and the story: 1006 gave
`MercenaryType 1` a meaning three days after this witness used 1 as an arbitrary nonzero marker). The
fixing hotfix is merged into this branch (`386318f`). The selection above is 18, the prior population
of 16 plus this story's own two new install-gated tests (`pkg/game/townsquare_release_test.go`).

## Open items

- **Reconciliation at the pin bump.** The submodule moved from `3a78df1` to `4f2651a` at this
  story boundary, which landed `EXP-0196` — town paint — under rows this story had already
  written. Every one of the three was re-read against it. `DIV-148` is now CLOSED: `TOWN-163`
  publishes the same five mask bytes this story measured from art, and the room identity of all
  five follows from `TOWN-163`, `TOWN-161` and `TOWN-164` read together. The still-unread writer
  of `+0xb4` and message-to-handler table do not leave the mapping or a player-visible behaviour
  unresolved. `DIV-149` and `DIV-150` remain DEVIATION rows: the overlay and the three labels are
  both gated in the original and unconditional here, which the player sees as three permanently lit
  doorways. Both rows also carried a sentence the claims falsify — `DIV-149` said no claim
  addressed `town_add.bmp`, where `TOWN-002` had addressed it by name, and `DIV-150` inferred from
  the art's opacity that the labels were unlikely to be conditional, where the original blits them
  opaque and gates them anyway. Neither is a defect in this story's contract, which promised the
  labels at their positions and delivered them; both are the material for the follow-on story.
  `TOWN-161` also confirms the three label origins exactly (`spec.md` FR-3).
- **Two rows the ledger did not carry, opened at the landing.** `DIV-153` (the town's own
  PRNG-rolled decoration and the four class-keyed sprite sheets, none of which this build paints)
  and `DIV-154` (the town's own tip widget, rect `(328,0,640,200)`, which this build does not
  draw). Both were out of this story's contract and are not gaps in it, but the mismatch had
  existed only in the contract's out-of-scope list, and a mismatch may not live only in a spec cut
  list, a story doc, or chat. `DIV-151` and `DIV-152`, reserved to this story and unused, stay
  retired rather than being reused for them.
- `EXP-0196`'s subject (town decoration: animated signs, doors, stars, weathervanes, townbirds,
  and the PRNG-placed horse/baba/dervish extras) is explicitly out of scope and untouched by this
  story; it now has a ledger row, `DIV-153`.
- The tavern's, shop's and school's own room interiors are unmodified.
- No witness combines the real shipped art with `stepTownAt`, the real pointer-click dispatch.
  `cmd/townsquarecheck` reads the real `townmask.bmp` and calls `TownSquareControlAt` and `Choose`
  directly, bypassing `stepTownAt`; `pkg/ui/townsquare_test.go` drives `stepTownAt` in full but only
  against `fakeSquareArtTown`, a synthetic fixture. Closing this gap would mean adding exported
  pointer-click API surface to `pkg/ui` for `cmd/townsquarecheck` (package `main`) to call, since
  `stepTownAt` itself is unexported; that was judged not cheap, so this remains disclosure rather than
  a fix. Round 3 (W-2) built the adjacent, cheaper witness this same gap analysis had left as
  disclosure since round 1: `pkg/game/townsquare_release_test.go` combines the real shipped art with
  the real `ComposeTownSquare`, install-gated, and its mutation found P-2 (above). Two of the story's
  three P findings — round 2's P-1, round 3's P-2 — were each found only when real art was driven
  through a production path a prior round had left as disclosure; a gap that has produced every P
  finding in the story is no longer disclosed alone. The remaining, narrower gap named here is
  `stepTownAt`'s own pointer dispatch specifically, not the composer. The keyboard-dispatch defect
  round 2's P-1 finding named is exactly the shape of defect
  this gap would have caught: production art plus production dispatch, driven together, would have
  shown the switch running unguarded.
