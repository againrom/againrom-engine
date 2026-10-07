# Story `1028` — the mission command panel — spec

Canonical at landing. States what this build actually does, not the original intention.

## Result

The mission column's second slot (id 6, `(864,158)-(1024,238)` at 1024x768) composes the original's
own command panel: eight 34x34 cells built from the four shipped `interface/` bitmaps, with the
disabled cells and the selected cell drawn the way `MENU-COMBAT-018` draws them, and the label of the
cell under the cursor drawn from `main.txt`. The panel's eight keys are the original's own —
`A`, `M`, `G`, `D`, `C`, `S`, `T`, `R` — and the five keys that collided with them move elsewhere.
This build's four display-switch buttons (`hudtoggles.go`) no longer draw a box in this slot; `I`,
`S`, `D` and `E` still switch the pack bar, the spellbook bar, the doll and the worn set as keys
alone.

Pointable in `builds/current/`: the strip under the minimap shows the game's own eight-cell command
bar instead of four authored letter boxes, and pressing `G` orders Guard.

## Three decisions delegated to this lane

**D1 — the S/D keyboard collision.** `MENU-COMBAT-019` assigns `S` to Swarm and `D` to Defend; the
map screen's own display switches (owner, 0140, 2026-08-11) already used `S` for the spellbook bar
and `D` for the doll. Resolution: the display switches keep `S` and `D`; Swarm and Defend get no
keyboard route. Reason: Defend and Swarm are out of this story's scope (`DIV-230`) and ship disabled
through the panel's own skip mask, so neither order exists to collide for. Giving up a letter to an
order this build cannot issue yet costs nothing today; the collision becomes real only if a later
story builds Defend or Swarm, at which point it needs its own resolution (`DIV-232`).

**D2 — what cell 1 (Move) does.** `AI-PANEL-053`'s table entry 2 is an armed mode with no cell of the
original's eight naming a plain move explicitly built here before this story (Guard, Stand Ground,
Patrol, March and Attack all pre-exist; a plain ground tap already issues a move through `MapOrder`
outside any armed mode). Resolution: Move arms a new authored state, `commandMove`
(`pkg/ui/command.go`), on the same pattern Patrol arms `commandPatrol` and March arms `commandMarch`:
the press arms, the next secondary press aims it. `commandMove` is normalised to `commandNone` at the
single order-build boundary (`pkg/ui/command.go`, the `decide` function), so nothing downstream of
that boundary ever learns a third armed-order byte exists; it resolves to the same plain `MapOrder`
call a ground tap outside any arm already makes. Reason: this keeps every consumer below the order
boundary unchanged and gives the Move cell an arm/aim shape consistent with the panel's other arming
cells (Attack, Cast) rather than issuing on the press itself.

**D3 — pick-up, minimap toggle and screen-edge panning.** Pick-up (`Grab`) moves from `G` to `B`
(later moved again, off `B`, at the return visit below), freeing `G` for Guard. The minimap toggle
(`Minimap`) moves from `M` to `V`, freeing `M` for Move. Screen-edge panning needs no new code:
`panIntent` (`pkg/ui/viewer.go`) already runs unconditionally alongside the keyboard pan, so removing
the WASD-style letter-key pan (`DIV-233`) leaves it as the camera's own always-on fallback, with the
arrow keys beside it.

## Return visit after adversarial pass 1 (2026-08-22)

Four fixes, against `pipeline/reviews/1028-return-brief-pass1.md` and one owner ruling that arrived
mid-visit:

- **The readout omitted an armed Move.** `PanelFieldOrder` (`pkg/ui/readout.go`) switched on
  `commandPatrol` and `commandMarch` but fell through to `readoutDisarmed` for `commandMove`, so the
  HUD showed "ORDER: -" while Move was armed, indistinguishable from no order at all. Fixed by adding
  a `commandMove` case returning a new `readoutMove = "MOVE"` constant.
  `TestTheReadoutStatesWhichOrderIsArmed` (`pkg/ui/vocabulary_test.go`) gained a `{commandMove,
  readoutMove}` row and was confirmed red against the unfixed switch before the fix, green after.
  Independent enumeration of every `commandMove` consumer (`decide`'s order-build boundary,
  `armCommand`'s toggle check, `commandPanelSelected`'s highlight, and this readout switch) found the
  same four sites the review named, confirming the readout was the only one missing a case.
- **No committed test drove `composeCommandPanel` end to end.** Added
  `TestReleaseCommandPanelComposesFromRealArtInFourStates` (new file,
  `pkg/ui/commandpanel_release_test.go`, install-gated on `AGAINROM_ASSETS`): asserts, against the
  real four bitmaps on both roots, the inactive frame (Heads alone), the active/no-arm frame (Active
  plus all three disabled cells), the Attack-armed frame (cell 0 from Selected) and the Move-armed
  frame (cell 1 from Selected). Registered in `internal/gatedtests/testdata/population.txt`. Killing
  the Move-armed assertion's own production condition in `commandpanel.go` was confirmed to fail the
  test before the change was reverted byte-identical, ruling out a vacuous pass.
- **Grab moved again, off `B`, and March moved off `R` — both on the owner's 2026-08-22 ruling** that
  a key of this project's own must not occupy or scramble a letter the shipped game names, since our
  own bindings are a temporary measure. `text/main.txt` lines 8-9 (`restool cat <root>/main.res
  text/main.txt`, both roots) read "Open/Close Spellbook `<Q>,<B>`"; `Grab` now binds `F` instead.
  `text/main.txt` line 7 reads "Retreat `<R>`", a decoded immediate opcode (`AI-PANEL-123`) this build
  does not yet issue; `March` (this project's own order, 0146, not a decoded one — `go run
  ./tools/claim -k "march"` and `-k "standing order"` return no matching order, only an unrelated
  line-of-sight accumulator) now binds `H` instead, leaving `R` free for a future Retreat. Both
  replacements were checked against `text/main.txt` lines 0..14 on both roots, which name neither
  letter, and against `Q`, `TAB` and `~` being the original's own unbound letters and therefore not
  free to take. Recorded as `DIV-236` (Grab/F) and `DIV-237` (March/H). `S`→Book and `D`→Doll
  (`DIV-232`) are unaffected; that collision was ruled separately, the same day.
  `TestTheFourKeysAreBoundOnceEach` (`pkg/ui/vocabulary_test.go`) checks `H` in place of `R`.
- **The Inventory-key comment overstated an absence.** The code comment beside `app.go`'s four
  display-switch bindings said no decoded fact names any of their keys. `text/main.txt` lines 10-11
  read "Open/Close Inventory `<~>,<I>`", and this build's own `Inventory` binding already reads `I`.
  The comment is corrected to say `I` is named by shipped content and the other three letters
  (`S`/`D`, ruled separately; `E`, still unestablished) are not; no binding changed.

## Keys as built

| Key | Field | Behaviour |
|---|---|---|
| `A` | `Attack` | Arms attack (guarded by `!ctrlHeld()`; `Ctrl+A` is Autocast) |
| `M` | `Move` | Arms `commandMove` |
| `G` | `Guard` | Issues Guard immediately |
| `T` | `StandGround` | Issues Stand Ground immediately |
| `C` | `Cast` | Read and discarded; no seam exists to call (`DIV-234`) |
| `S` | `Book` | Toggles the spellbook bar (display switch, not Swarm) |
| `D` | `Doll` | Toggles the doll (display switch, not Defend) |
| `H` | `March` | Arms `commandMarch` (pre-existing order, 0146; moved off `R` at the return visit, `DIV-237` — `R` is `MENU-COMBAT-019`'s own Retreat letter and a decoded immediate opcode this build does not issue) |
| `P` | `Patrol` | Arms `commandPatrol` (pre-existing key, no panel cell of its own, `DIV-231`) |
| `F` | `Grab` | Pick-up (moved from `G`, then off `B` at the return visit, `DIV-236` — `B` is `text/main.txt`'s own Spellbook letter) |
| `V` | `Minimap` | Minimap toggle (moved from `M`) |
| `I` | `Inventory` | Toggles the pack bar (display switch) |
| `E` | `Worn` | Toggles the worn-set box (display switch) |
| arrows | `PanLeft/Right/Up/Down` | Camera pan (only keyboard pan left; `DIV-233`) |

`W`, `A`, `D`, `S` no longer pan the camera; `A`, `S` and `D` are claimed above, and screen-edge
panning (`panIntent`) is unconditional throughout, before and after this story.

## B1 — the panel draws

`composeCommandPanel` (`pkg/ui/commandpanel.go`) paints, in `MENU-COMBAT-018`'s own order: the
inactive ground (`Heads`, `interface/headsr.bmp`) alone when `commandPanelActive` is false;
otherwise the active ground (`Active`, `interface/commandbarr.bmp`), then each skip-masked cell's own
rectangle from the disabled bitmap (`Disabled`, `interface/commandempr.bmp`), then the selected
cell's own rectangle from the selected bitmap (`Selected`, `interface/commanddnr.bmp`). Cells are
34x34 at panel-local `(8 + 34c, 7 + 34r)`, row-major, four columns by two rows (`commandCellRects`,
`TOWN-092`'s own formula). There is no hover bitmap, no border and no separate per-cell icon; the
four bitmaps are read once at load (`LoadCommandPanelArt`, `pkg/game/commandpanelart.go`) and held on
`Viewer.commandPanelArt`, recomposed every frame it draws rather than cached, matching the panel's own
low per-frame cost (eight cells, at most two overlays).

The cell under the cursor draws its `main.txt[0..7]` label (`commandHoverPresent`,
`pkg/ui/commandpanel.go`), resolved through `ui.Words.Command` (`pkg/ui/words.go`), populated by
`InstallWords.Words()` (`pkg/game/installtext.go`) from the install's own `text/main.txt`.

A nil `CommandPanelArt` (an install this build could not resolve all four bitmaps from) draws
nothing: there is no authored fallback appearance for a panel with no shipped bitmaps behind it.

## B2 — the cells act

`pressCommandPanelCell` (`pkg/ui/commandpanel.go`), reached from `command.go`'s own gesture arm: left
down acts; because every press edge fires independently, a double-click's second press is just
another primary press and needs no separate bookkeeping. A held left-drag re-enters the press on
every new cell it crosses (`cmdDragCell`, reset whenever the button is not held). A right release
(`SecondaryReleased`, new `Input` surface scoped to this one panel) cancels the drawn overlay only,
leaving any armed mode intact; every other right-button edge is a no-op here. A margin press, or a
press on a skip-masked cell, or a press while the panel is not active (`commandPanelActive`, the same
ownership gate `canArmAttack` already reads) clears only the overlay and arms nothing.

Five of the eight cells act: Attack arms attack (`v.armAttack`, unchanged); Move arms `commandMove`
(D2 above); Guard and Stand Ground issue immediately onto one-shot pending flags app.go's own stance
dispatch consumes once each (`consumeCommandStancePending`), the same seam the `G`/`T` keys call, so a
mouse press and a key press reach `a.flow.stance` through one statement; Cast reflects the
spellbook's own selection and arms nothing (D2's sibling gap, `DIV-234`). Three cells — Defend,
Swarm, Retreat — are skip-masked (`commandPanelSkipMask`) and refuse every press (`DIV-230`).

## B3/B4 — the eight keys, and the collisions that move

Covered in the keys table above. Every replacement letter (`F` for Grab, `H` for March, `V` for
Minimap, `A` for Attack) is verified free by grep against the rest of `pkg/ui/app.go` and `readInput`
in `pkg/ui/viewer.go` before being bound, the way each binding's own code comment records; `F` and `H`
are additionally checked against `text/main.txt` lines 0..14 on both roots, at the return visit.

## B5 — the display switches lose their buttons

`hudtoggles.go`'s four switches (`I`, `S`, `D`, `E`) no longer draw a box in the command-panel slot;
`hudPanelPack/Book/Doll/Worn` and `toggleHudPanel` are unchanged, but `viewer.go`'s `Draw` no longer
calls the removed `hudTogglePresent`/draws the removed `hudToggleImg` — that call site now draws the
command panel and its hover label instead (`commandPanelPresent`, `commandHoverPresent`). The four
switches are keys only, per the owner's 2026-08-22 ruling (`pipeline/OWNER-RULINGS.md`, that date).

## Out of scope, as built

- Defend, Swarm, Retreat as orders: not built; their cells ship disabled (`DIV-230`).
- Patrol: not one of the eight; keeps its pre-existing key `P` and gains no cell (`DIV-231`).
- The minimap's content (id 5) and the character panel's content (id 7): untouched by this story.
- The panel's own message protocol: this story routes a cell to an existing seam
  (`v.armAttack`, `v.armCommand`, `a.flow.stance`), not a reconstruction of the original's message
  post.
- 800x600: this build composes at 1024x768 only; the panel's rect at 800x600 is not built.

## Install-gated witness

`pkg/game/commandpanel_release_test.go`,
`TestReleaseCommandPanelArtAndLabelsComposeFromTheRealInstall` (part of B1's own commit, not a
follow-up): resolves `FrontEnd.CommandPanelArt` from a real asset root, checks all four bitmaps are
160x80 and pairwise pixel-distinct over their whole area (a swapped path constant would read the same
file twice and pass a corner-sample check), and checks each of the eight resolved `main.txt` labels
ends with the bracketed accelerator letter this build actually binds for that cell. Runs once per
root through `pipeline/check-release-tests.sh`.

`pkg/ui/commandpanel_release_test.go`,
`TestReleaseCommandPanelComposesFromRealArtInFourStates` (added at the return visit, item 2 of
`pipeline/reviews/1028-return-brief-pass1.md`): drives `composeCommandPanel` itself, end to end, from
the same four real bitmaps, over four subtests — inactive, active/no-arm, Attack-armed, Move-armed —
and compares the composed frame byte for byte against an independently built expected frame per state.
The `pkg/game` test above witnesses that the right bitmaps and labels are resolved and match the
build's own key bindings; this one witnesses that the composer itself paints the right cells for the
right panel state, which no committed test reached before this visit. Also runs once per root through
`pipeline/check-release-tests.sh`, and is listed in
`internal/gatedtests/testdata/population.txt`.

## Divergence rows

`DIV-230` through `DIV-234` are spent, none returned. `DIV-201` is narrowed, not closed: the ordering
half was never a divergence, id 6's own content now matches the decode for the five cells this story
builds, and the remainder is exactly `DIV-230`..`DIV-234`. Full text: `docs/DIVERGENCES.md`.

Two more rows are added at the return visit, both instances of the same owner ruling (2026-08-22):
`DIV-236` (Grab moved off `B`, `text/main.txt`'s own Spellbook letter, to `F`) and `DIV-237` (March
moved off `R`, `text/main.txt`'s own Retreat letter, to `H`). Neither reopens `DIV-232`
(`S`→Book, `D`→Doll), which the owner ruled separately the same day and this ruling does not disturb.

## Style

`PROSE.md` governs this document.
