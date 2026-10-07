# Terminal campaign defeat

Once a campaign mission is Lost, only loading another session permits play.
The failure panel offers installed Exit to Main Menu and Load Game labels.
There is no town return, retry, party repair or XP penalty.

## Authority and scope

Owner direction requires terminal single-player defeat and retains story
`1055`'s DYING Heal/revival window. Research pin `ba21c9a` supplies
`MISSION-DEFEAT-045/046`, `HERO-DEFEAT-136`, `SESS-DEFEAT-064/065` and the
amended `MISSION-PATH-015`/`MISSION-STOP-016`. Campaign mode excludes the
automatic repair branch. The prior campaign XP premise in `DIV-219` was wrong.

`pkg/game` owns loss admission, the latched native state and campaign routing.
`pkg/ui` owns the two-button modal, store availability, input hold and load
transaction. Simulation rules and binary form are unchanged.

## Behaviour

- Death, script loss and the reserved failure message reach the same terminal
  panel. Victory/Continue cannot dismiss it into play. Return/Escape exit.
- Load is disabled without a loader and listed saves. Opening Load retains the
  failure panel. Cancel, missing/malformed files and refused openers cannot
  resume the failed game. Successful load replaces the old session.
- The modal holds simulation and map input. Missing font cannot remove this
  terminal gate, including while an older-save disclosure replaces the panel.
  Return/Escape pages back to failure, then exits.
- Native `SnapshotResidue.MissionLost` retains frontend-only failure latches.
  Old saves default false and still derive death/script loss from their world.
  A saved Lost session reopens terminal before its first input. An older-save
  disclosure returns to failure after its last page. Mission saves already use
  `.ags`; no original writer is added.
- Victory and a refused non-loss outgoing opener retain their existing routes.
  No gameplay continuation carries the pre-mission party out of defeat.

## Proof and debt

`terminaldefeat1091_test.go` in game and UI covers the action table, pauses,
native latch/default, disabled Load, cancel, failed open and replacement.
The EN/RU release witness applies canonical lethal input to the installed
mission-10 primary, drives App and the existing admission rule, then tests
missing/malformed files and native continuation. `terminal-defeat.json` is a
continuation scenario seeded by that witness with temporary `lost 1091` and
`running 1091` saves; it is not a standalone menu-start scenario.

`verification.md` records gate results. `DIV-219` retains the timing conflict;
`DIV-125` retains the Unknown companion-guard scope. `DIV-137` no longer treats
defeat as a homeward route. `DIV-610` discloses authored geometry, keyboard and
cancel/no-font policy plus native compatibility. No original GUI, runtime loss
timing, network recovery, primary/companion admission change, install write or
current-build publication is claimed.

The single adversarial pass returned the combined Lost/disclosure/no-font
route. Its bounded correction retains a presentation-only terminal latch
until the notice closes, independently of its current picture kind. Combined
App regression covers the font absent before and removed after disclosure,
unchanged tick/hash, captured shortcuts, and keyboard exit to the main menu.
