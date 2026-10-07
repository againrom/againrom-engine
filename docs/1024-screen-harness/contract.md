# Contract — 1024, the screen harness

## What the owner asked

Owner, 2026-08-21, in three parts:

1. Every screen must be drawable on its own, apart from the game, by passing it the payload it needs.
   The game uses the same renderer.
2. Every screen must have a test that checks its composition. The assets belong to the install, so
   the harness must take them as an input and draw with them.
3. Development of a screen starts with that screen's test, so a wrong or broken composition is
   reported by the suite rather than by a screenshot.

He also asked, separately, why no headless screenshot tool exists. That question is answered by the
`tool-screenshot` branch, which builds the single selection seam this story depends on. This story is
the test half.

## Where the tree already meets the ask, and where it does not

Measured at `16177e9`. The commands are named because no committed command prints these yet; **B4
below makes them reproducible**, which is the reason that behaviour exists.

Part 1 is largely built. `pkg/ui` exports 12 payload-shaped composers — `ComposeTownSurface`,
`ComposeTownSquare`, `ComposeWorldMap`, `ComposeChargenFrame`, `ComposeShopScreen`,
`ComposeTipPanel`, `ComposePickupRows`, `RenderCharacterPanel`, `RenderPanel`, `RenderNotice`,
`RenderDoll`, `RenderWorn`. Each takes a view struct and returns an `*image.RGBA`; none imports
ebiten. `App.Draw` calls them and uploads the result. So a screen already draws from a payload, and
the game already uses the same renderer.

Two things are missing from part 1, and only the second is this story's:

- The choice of which composer serves which screen exists once, inside `App.Draw`
  (`pkg/ui/app.go:2539`), on the ebiten side of the boundary. `tool-screenshot` moves that choice to
  an exported headless entry and refactors `Draw` onto it.
- The mission screen is not of this shape at all. `Viewer.Draw` (`pkg/ui/viewer.go:2569`) draws to
  the `*ebiten.Image` directly. It has no CPU composite and this story does not give it one.

Part 2 exists in a weak form. 30 `TestRelease*` functions, all in `pkg/game`, draw with a real
install taken through `AGAINROM_ASSETS`, so asset injection is solved. Two properties spoil it:

- They are named and organised by **incident**, not by screen. There is no way to ask what witnesses
  a given screen.
- They **skip silently** when the variables are unset, and a skip and a pass both print `ok`. This is
  the failure `pipeline/check-release-tests.sh`'s own header records: a release test was red on both
  installs for two days while every chain printed green.

The synthetic layer is uneven, and the unevenness is measurable. Counting composer calls from
`pkg/ui/*_test.go`: shop 15, notice 15, pickup 8, town surface 5, world map 4, town square 3, tip
panel 3, character panel 3, **`ComposeChargenFrame` 0**, **`RenderPanel` 0**. `ComposeChargenFrame`
is reached only from `pkg/game/chargen_release_test.go` and `pkg/game/tippanel_release_test.go`, both
gated. The character generator's composition is therefore witnessed only by the layer that skips
silently, and the character generator is the screen with the most owner-reported defects this month.

Part 3 has no rule. `AGENTS.md`'s golden rule 2 states the opposite in letter — tests are synthetic
and install-free — and it is correct and stays. It does not forbid what the owner asks: a gated test
taking the root as a runtime input **is** the asset injection he describes. What it does is make such
tests second-class, outside the repository's own chain, which is why nobody writes one first.

## Observable result

A screen cannot be added, and a screen's composer cannot go untested, without `go test ./...`
failing — with no game install present. The count of screens, their composers, their synthetic
witnesses and their gated witnesses is printed by a committed command, and a screen with no witness
is an error rather than an absence nobody counted.

## Behaviours

**B1 — a screen registry that fails closed.** Every `ui.Screen` value carries an entry naming the
payload composer that serves it and the synthetic test that exercises that composer, or a recorded
reason it has neither. A `Screen` value with no entry fails the test build. This is the shape
`internal/archtest` already uses for package edges, and it is chosen because it is the only shape in
this tree that has held: an unknown package is an error there, not a warning, so no package has ever
been added silently. `ScreenMap` and `ScreenPicker` are expected to be reason-carrying entries rather
than composer entries, for the two causes stated above; the reason text is part of the entry and is
read by whoever the registry stops.

**B2 — one synthetic composition test per registered screen.** Built from a payload the test
constructs, asserting on the composed image, with an expectation that does not come from the constant
or formula under test. The bar is mechanical and is stated here so the story can be checked against
it: **shifting the screen's destination rectangle by one pixel must fail at least one test.** A
composition suite that survives a one-pixel shift is not a composition suite. This closes the two
measured holes, `ComposeChargenFrame` and `RenderPanel`, and the five surviving destination-rect
mutations enumerated below.

**B3 — the gated layer becomes visible from inside the ungated chain.** A synthetic test asserts that
every gated test is enumerated and names the screen or subject it witnesses, and that the enumeration
matches the registry. It requires no install: it measures the population, not the pixels. The
purpose is that "the gated suite did not run" stops printing the same word as "the gated suite
passed". `pipeline/check-release-tests.sh` keeps measuring its own population at run time and is not
replaced; its number and the registry's must agree, and disagreement is a failure rather than a note.

**B4 — one command prints the census.** Screens, composers, synthetic witnesses, gated witnesses, and
the unwitnessed remainder. Its screen column must reconcile against `cmd/screenshot -list`'s own
9, and its gated column against `pipeline/check-release-tests.sh`'s own run-time count, which was 35
on both roots at `4de5d8b`. Two instruments over one population disagree only when one of them is
wrong, and the disagreement is a failure rather than a note. Every number quoted in this contract must come out of it afterwards, for
the reason `cmd/divcensus` exists: a census quoted from a program nobody can run is a number that
cannot be checked, and this contract is currently full of them.

Four behaviours, one domain. The contract's own ceiling is therefore **three adversarial passes**.

## The process rule, landed with the story and not as a fifth behaviour

`AGENTS.md` and `SDD/WORKFLOW.md` gain: a story that changes what a screen draws extends that
screen's composition test **before** the production change, and the extension is expected to fail
first. It is a rule no script can hold — a gate cannot tell which edit came first — so it belongs in
the normative text by this project's own test for what goes there.

## The measured starting set

Every number below was produced by a committed command or a mutation run at this seat between
`4de5d8b` and the writing of this section. None is an estimate. The lane starts from this list; it
does not re-derive it, and it reports which entries it closed.

### Screens the tool can compose today: 5 of 9

`cmd/screenshot -list`, landed with `tool-screenshot` (merge `291d13d`):

| screen | state | why |
|---|---|---|
| `menu` | composes | `menu.Assets.Compose` |
| `chargen-precreate` | composes | the generator's pre-create page |
| `chargen-detailed` | composes | the generator's detailed page, after Forward |
| `town-square` | composes | once the campaign's first town-declaring mission finishes |
| `town-shop` | composes | entered through the square's own SHOP door |
| `load` | refuses | `ScreenLoad` draws only through `ebitenutil.DebugPrintAt` |
| `picker` | refuses | `ScreenPicker`, same cause |
| `gameplay` | refuses | `Viewer.Draw` paints the ebiten canvas directly, no CPU composite |
| `game-menu` | refuses | ebiten vector dim and panel over a running mission, same cause |

**5 of 9 is the number this story moves.** B1's registry entries for `load` and `picker` carry the
debug-text reason; `gameplay` and `game-menu` carry the canvas reason and are out of scope above.
Whether `load` and `picker` become composable inside this story or are recorded as reason-carrying
entries is the lane's call, stated in `spec.md` either way.

### The seven unwitnessed symbols in the new dispatch layer (review W-1)

`tool-screenshot`'s pass-1 adversarial review classed these W and they were not fixed there, so that
the fix lands with the harness that measures it rather than as a one-off. Two mutations of this layer
survived the module-wide suite at the time of the review.

| symbol | file:line at `4de5d8b` |
|---|---|
| `(*App).composeScreen` | `pkg/ui/app.go:2566` |
| `dialogueOrigin` | `pkg/ui/app.go:2819` |
| `composeTownRoom` (package-level), including its hover and drag branches | `pkg/ui/app.go:2853` |
| `(*App).composeTownRoom` | `pkg/ui/app.go:2918` |
| `(*App).composeTownScreen` | `pkg/ui/app.go:2955` |
| `(*App).composeChargenScreen` | `pkg/ui/app.go:3138` |
| `HeadlessFrame`'s `note` branch | `pkg/ui/headless.go:138` |

**Read the symbol, not the line.** The review handed these at `291d13d` and four of the seven had
already drifted by `4de5d8b`, because the review's own fix commit added `overlayTownDialogue` to the
same file: `composeTownRoom` moved 2847 to 2853, `(*App).composeTownRoom` 2912 to 2918,
`composeTownScreen` 2924 to 2955, `composeChargenScreen` 3113 to 3138. The numbers above were
re-read at `4de5d8b` and will drift again.

`overlayTownDialogue` (`pkg/ui/app.go:2945`) is NOT on this list — it was added by the same
review's fix and is witnessed by `TestTheWorldMapTakesNoDialogueOverlay`, mutation-proved at 24000
pixels.

### The five destination-rect mutations that still survive

B2's bar is "shifting a screen's destination rectangle by one pixel must fail at least one test."
That bar was measured rather than assumed. The test audit
(`docs/TEST-AUDIT-2026-08-21.md`) ran 13 mutations of exactly that shape at `16177e9`, killed 7, and
found **all 6 survivors in `pkg/ui`** — every `pkg/render` subpackage sampled killed its own.

**All six were re-run at `4de5d8b` and one is now killed.** Hotfix `23bb8e6`'s
`tiprectorigin_test.go` pins `shopTipRect`'s origin and width, so shifting it by one pixel now fails.
The starting set is five, not the audit's six:

| site at `4de5d8b` | what the mutation moves |
|---|---|
| `pkg/ui/chargen_page.go:100` | `chargenName`'s own control rect, `image.Rect(448, 20, 628, 46)` |
| `pkg/ui/chargen_page.go:742` | the plate copy's source rect, `copyNative(dst, p.Plate, ...)` |
| `pkg/ui/tippanel.go:497` | the tip panel's own left border tile destination |
| `pkg/ui/townshell.go:348` | a shell rect, `image.Rect(200, 196, 280, 228)` |
| `pkg/ui/townshell.go:869` | the button value's own text rect, `drawTownShellText(...)` |

| closed since the audit | by |
|---|---|
| `pkg/ui/shopscreen.go:125`, `shopTipRect` | `pkg/ui/tiprectorigin_test.go`, hotfix `23bb8e6` |

Each mutation above was applied and reverted with the same edit pair and every revert was
byte-identical; the tree was clean afterwards apart from this contract.

One survivor has a named cause worth carrying into the design. It is
`chargen_page.go:100`, and `TestPreCreateControlBoundsAndNativePixels` enumerates
`chargenChoice0`..`chargenChoice3` in
`preControlRegion`'s switch and never calls it for `chargenName`, which is the case the mutation
moved. The test is not weak; its enumeration is incomplete. That is this repository's own coverage
rule 1 — enumerate producers, not convenient data rows — and it is the failure mode B1's fail-closed
registry exists to make impossible.

### The worked precedent: hotfix `23bb8e6`

Landed 2026-08-21, and it is what this story is for. Two of the five tip panel rects shipped 160px
left of where the original draws them, for a year-old reason no test could see: a published
construction literal is parent-relative, and reading it as absolute is silent when the parent's
origin happens to be zero for some siblings and not others. **Reverting the tavern's origin to x=0
reddened no test in the module.** The generator's first portrait was 97.9% covered by the tip panel
and nothing said so.

The fix built `pkg/ui/tiprectorigin_test.go`, which pins all five origins and widths against the
claims' own literals on the right-hand side, and proved it by mutation: +1 in X on each of the five
killed all five. **That test is the shape B2 asks for, at one screen's scale**, and the lane should
read it before writing the first composition test. Two things it demonstrates:

- the expectation came from the claim, not from the constant under test, which is coverage rule 2;
- both existing fixture guards **caught the move and were corrected rather than relaxed** — the
  crossing test now searches for a pixel production does not swallow instead of requiring a whole
  rect clear, and the swallow enumeration clicks each covered choice under its own pin. A fixture
  guard that fails when geometry moves is the harness working, not an obstacle to the change.

### What is not measured

The three audits ran 89 mutations against a production population in the tens of thousands of lines.
The survivor rates are informative about the presence of gaps, not a bound on their number. This
list is a floor.

## Twelve-aspect applicability

Data N/A. Runtime state N/A. Simulation N/A — this story reaches no hashed state and touches no
`pkg/sim` file. Player input N/A. AI N/A. **UI/HUD applies**: the registry and the composition tests
are entirely this aspect, and no shipped pixel changes. Triggers N/A. Inventory/equipment N/A.
Persistence N/A. Campaign/session N/A. **Shipped content applies** through the gated layer, which
must keep running on both installs. **Interactions with existing mechanics applies**: `App.Draw` is
refactored by the branch this story depends on, and every screen must keep drawing exactly what it
drew.

## Domains

**Client** (domain 8), plus `internal/` test infrastructure, which is not a domain. No other domain
is touched. The adversarial reviewer walks the Client interface and the `internal/archtest` pattern
this story copies.

## Dependencies and order

**`tool-screenshot` landed 2026-08-21 (merge `291d13d`), so this story is unblocked.** B1's registry
names the same selection seam that branch created — `composeScreen`'s switch and the package-level
`composeTownRoom` — and building a second one would reproduce the defect both are written against. It is
independent of stories `1021`, `1022` and `1023`, which change what is drawn; this story changes what
is witnessed. If it runs beside one of them it takes that story's files as they stand and does not
edit them.

## Out of scope

- **The mission screen.** Giving `Viewer.Draw` a CPU composite is a separate and much larger job —
  a software rasterizer or an offscreen graphics context — and it is not attempted here. The registry
  records `ScreenMap` as unwitnessable by this harness, with that reason, which is the honest state
  rather than a gap nobody wrote down.
- **Rewriting the 30 existing `TestRelease*` tests.** They are enumerated and made visible, not
  renamed or moved. Renaming them by screen is worth doing and is its own work.
- **Deleting any test.** The audit landed as `docs/TEST-AUDIT-2026-08-21.md` and its work list has six
  items; only item 4, the surviving destination rectangles, belongs to this story. The other five
  — the self-referential constant comparisons, `pkg/audio`'s channel identity, the draw-filter class,
  the clamp boundaries, and `headlesspointer.go`'s 0.0% coverage — are separate decisions and separate
  work. The deletion set of this story is empty.
- **Putting game assets in a test.** Golden rule 2 stands unchanged. The gated tests keep taking the
  root as a runtime input; nothing in this story reads an install from a test that is not already
  gated.

## Divergence rows

None expected. This story changes no shipped behaviour and states nothing about ROM1, so it neither
opens nor closes a row in `docs/DIVERGENCES.md`. If the composition tests written under B2 find a
difference from ROM1, that difference is a row and a defect story, not part of this one.

## Research claims

None cited. The harness makes no claim about what the original draws. The screens it witnesses are
witnessed against this tree's own current composition, and the question of whether that composition
matches ROM1 is owned by the stories that draw it.
