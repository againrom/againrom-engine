# Story 1122 — shop interior animation

## Player result

The playable shop keeps its installed room, four rack selectors, stock,
buy/sell table, backpack and doll interactions. The selected rack now runs its
installed opening and steady frames, a switched-away rack shows its closing
frame when the next paint has not reached the gate, and the merchant runs the
installed Idle, Yes and No poses with their shared-index priority.

This is presentation only. Prices, quantities, shelf and pack scrolling,
drag/drop, Book, messages, navigation, native saves, simulation state and the
story1121 tavern controller keep their existing rules.

## Authority and policy

`SHOP-ANIMATION-081` through `SHOP-ANIMATION-085` at research pin
`e212bacadff02811ef7fefe44e6c267ab645612b` establish the four eleven-file rack
families, the merchant base and three modes, the `>=100 ms` eligible paint,
one-step/no-catch-up rule, shared merchant index, Idle/Yes/No priority and the
known reaction request sites.

The outer original paint dispatcher, generator seed, inherited writes,
cross-visit reuse, cleanup ownership, malformed-family behavior and shop sound
identities remain Unknown. Againrom therefore advances only in focused live App
shop composition and reacquires the view after the step. A private per-visit
generator supplies the five-way idle candidate. Focus/menu/cutscene suspension
rebases clocks; room exit, new game and load discard the controller. Complete
art families live in the existing FrontEnd shop-art cache and degrade
independently. No new sound path is invented; existing generic click cues stay.

The animation visually arms rack0 on entry without changing the established
`shopNoShelf` model state. Thus stock remains hidden until the player clicks a
rack. `DIV-850` through `DIV-857` ledger that policy and the remaining Unknowns.

## Touched surfaces

- `pkg/game`: immutable frame-family loading, private controller, lifecycle,
  rack and buy/sell reaction bindings, native/hash boundary tests.
- `pkg/ui`: optional animator seam, live composition delivery and read-only
  selected pictures.
- `internal/gatedtests`: the installed EN/RU App/frame witness population.

## Proof

Focused tests cover the first eligible paint, exact 100 ms boundary, no
catch-up, rack3..8 steady loop, file10 close, same-rack no-op, additive shared
merchant index, Idle/Yes/No priority, reaction bindings, focus freeze/rebase,
fresh visit/session reset, family-local fallback and byte/hash invariance.

`TestReleaseShopInterior1122InstalledAppFramesAndNative` independently decodes
literal installed paths for all `4x11` rack frames, Pose files1..29 and Yes/No
files2..12. It drives actual `App.Draw` paints, composes expected frames from
those independent pictures, and checks native save bytes plus simulation hash.
The release gate runs it once on each preserved EN and RU root. Optional review
captures are emitted only when `AGAINROM_STORY1122_FRAMES` names an ignored
output directory.

The relevant town/shop scenario must pass on both roots. No milestone census is
needed because simulation, pathing and script populations do not change.

## Open debt

Original physical paint cadence, cross-visit object reuse, arbitrary retained
index interruption, picture destruction, sample identities and audible output
remain Unknown. No GUI-device or speaker result is claimed.
