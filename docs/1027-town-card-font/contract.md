# Story 1027 — the town's statistics card at a font it fits

## Result

The statistics card is legible in the tavern, the school and the shop, on both roots. Every right
column value is drawn rather than truncated to nothing, all sixteen rows fit, the character name is
visible and centred, and nothing draws over it.

Pointable in `builds/current/`: press STATS on a party member in any town room and read the card.

## Why this story exists

Story `1022` landed at `401e59e` and its adversarial pass 3 found three player-visible defects on the
one surface the owner asked for on 2026-08-21 (*"such a window must be everywhere we show anyone's
statistics"*). All three were confirmed independently at the orchestrator's seat before this contract
was written. `1022` reached its three-pass ceiling, so the remainder is this story rather than a
fourth round, which is the scoping diagnosis this project's own rule prescribes.

## What is decoded

Nothing new. This story changes no ROM1-derived value. `DIV-191` records that `UNIT-PANEL-011`
forecloses a decode of the panel's row order, so the arrangement stays the owner's; this story does
not touch the arrangement.

## The three defects

**D-A. The town card composes at font1 and was designed for font2.** `pkg/game/townshell.go:14` sets
`TownCharacterView.Font` to `t.f.Font`, which resolves `DefaultFont = "font1"` (16x15).
`pkg/ui/chargen_page.go:908` passes `p.Font`, which is font2 (8x10). `pkg/ui/townshell.go:670` renders
the same 160x242 `CompactPanelLayout` with whichever it is given.

At font1 the review measured, on both roots: the left column's widest cell at 120 px, the shared
right origin at 126, and a right budget of 16 px against value origins of 36..126, so `panelFitValue`
truncates all twelve right values to the empty string. The twelve labels clip at the canvas, reaching
x=159 of 160, so the player reads `HE`, `MA`, `AB`, `DE`, `RE`, `FI`, `WA`, `AI`, `EA`, `AS`. With
`lineH=15`, pitch 16 and `Pad.Y=18`, fourteen of sixteen rows fit, so SIGHT and SPEED never draw.
**Re-measure all of this rather than taking these numbers.** They are stated here because the seat
must state what it believes; they are the review's, not a committed instrument's.

**D-B. The shop's `Book` plate covers the card's name row.** `pkg/ui/shopscreen.go:1065` fills
`shopBookRect` `(548,242,632,272)` opaquely whenever Statistics is on. `TownCharacterRegion` begins
at `(480,238)`, so the plate is card-local `(68,4)-(152,34)` and the name row's ink is card-local
`(84,18)-(138,32)`, entirely inside it. Checked at the seat by arithmetic on the two rectangles.
Round 3 stopped drawing the separate member-name row precisely because the card carries the name, and
then covered the card's own copy. On the shop this leaves the owner's ruling unsatisfied.

**D-C. The name is not centred.** `AlignValues` moves the name row's pen to the shared left-column
edge (`pkg/ui/panel.go:1591`) before `Center` measures the line (`:1629`). The review measured the
town card's name ink centred at 111 against a card centre of 80, and the chargen card's at 86.5
against 80.

## Behaviours

- **B1 — the card composes at a font that fits it.** `TownCharacterView` carries the card's own font,
  separate from the chrome's. `(*FrontEnd).tipFont()` (`pkg/game/tips.go:112`) already resolves font2
  with a graceful fallback to font1 and was built for this reason in `1018` P-3; use that seam rather
  than a second resolution path. The chrome (the chevrons, the mode box, the shop's `Book` label, the
  member-name row in DOLL mode) keeps the shell font it has today.
- **B2 — every right-column value is drawn on every town room, both roots.** Nothing truncates to the
  empty string at the production font, and all sixteen rows fit inside the card.
- **B3 — the shop's `Book` plate does not cover the card.** Solve it where it belongs: the plate exists
  so a transparent control does not sit on the card's own digits, and that need is real. Either the
  plate no longer overlaps the card's populated rows, or the control moves. State which and why.
- **B4 — the name row is centred on the card.** Fix the `AlignValues` and `Center` ordering rather
  than special-casing the first row. Both call sites must end centred.
- **B5 — the chargen page is unchanged.** It composes correctly today. A pixel change there is a
  regression, and its own tests are the check.

## The witness is part of the fix, not a follow-up

The two unit tests round 3 wrote use `panelWideFont()` (`pkg/ui/panel_test.go:672`), a synthetic cell
of 7x6 with advance 7, under half font1's width. That fixture is why round 3's truncation appeared to
work while the shipped screen was illegible. A synthetic font cannot witness a font budget.

**This story's witness is an install-gated render at the production font, through the production
composer, on both roots**, asserting that no right-column value is empty and that every row's ink is
inside the card. Golden rule 2 keeps it out of the repository chain, so it is one of the tests
`pipeline/check-release-tests.sh` selects, and the selection count that script prints must rise.

Mutate at the use site, not at the helper's own return, and revert byte-identical before applying the
real change.

## Also fix, and they are not behaviours

- **D-1 (document).** `AuthoredPanelLayout.Size.X` is `sidebarWidth = 300`, not 0
  (`pkg/ui/panel.go:796`). Three places state the opposite: the code comment at `pkg/ui/panel.go:1520`,
  `docs/1022-chargen-composition/spec.md`, and that story's `closure.md`. The review measured that the
  truncation passes therefore **do** run on the mission panel, and that a shipped starting party
  member's `WORN Short Sword, Soft Mail, Soft Boots` row is 367 px against a 280 budget, changing the
  composed panel at `(288,335)-(300,345)` on missions 10, 20, 30 and 40. Correct the three statements,
  open a divergence row for the mission panel's row now being truncated, and say plainly that story
  `1023` is planning on a false premise. Verify the measurement before recording it.
- **D-2 (document).** `DIV-191`'s justification cell says the font "is already fitted to each page's
  other rows and the card composes at the same width budget those rows use". The town shell's other
  rows draw in rects up to 456 px wide and the card's usable width is 142. That sentence is the
  premise that produced D-A. Amend the cell.
- **W-1 (witness).** `docs/1022-chargen-composition/spec.md` quotes `HP=91/91` and `Combat.Defence=17`
  from a probe that was deleted with its worktree. Either a committed instrument prints them or the
  numbers go. `cmd/screenshot -screens chargen-detailed` prints the default allocation, not the
  maximum.

## Domains

**Client**, and `pkg/game`'s front end for the font seam. Two domains. No reach into hashed simulation
state: a font is a presentation value and nothing here changes what the simulation computes.

## Ceiling

**Three adversarial passes**, and it should not need them. Five behaviours, two domains, no hashed
reach. If a pass returns a P that only the missing install-gated witness could have caught, that
witness is built with the fix rather than filed as a ledger row.

## Out of scope

- The mission column and `pkg/ui/hud.go` / `pkg/ui/viewer.go`. Story `1023`.
- The carried-weight row. `DIV-209`, story `1025`.
- The hero card's corner controls. `DIV-217`; research is decoding them.
- The card's row order and its arrangement. `DIV-191`, accepted on the owner's word.
- Changing `AuthoredPanelLayout`'s own truncation behaviour. D-1 asks for the documents and a
  divergence row to become true, not for the mission panel to be redesigned here.

## Gates

Both repositories' Go chains by glob, exit codes captured. `pipeline/check-release-tests.sh` and
`pipeline/check-scenarios.sh` on both roots, by name: this story changes what the screen draws, the
repository chain cannot reach those tests, and a skip and a pass both print `ok`. The release-test
selection count is **35** today; it must rise, and report the number the script prints rather than
this one. Deletion set empty or explained.
