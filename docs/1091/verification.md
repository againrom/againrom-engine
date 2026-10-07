# Verification

Base: `8ff87ec367008bf57eea927a501ecc796c2dccba`.
Research: `ba21c9aa9a949023b3d678b22ca29b3a3b0cd95f`, a forward descendant of
the base pin `26c755b2be9513b9ec90edba6527db4fdd613dc4`.

## Player result

The base's open-town loss route returned `NoticeToTown` with the pre-mission
party. The same continuity case now returns `NoticeToMenu`; the lost mission
cannot start a homeward route. The visible failure surface has Exit and Load
instead of the generic dismissal button. This is the story's player-visible
result, not a script-gap reduction. Seat publication to `builds/current/`
remains outside this branch.

The EN and RU installed mission-10 App witness applies a canonical lethal
command to the primary. Both runs report failure at tick 1 and read the title
and controls from main[141], dialogs[44] and dialogs[35]. Missing and malformed
listed saves leave the same world and failure panel intact. The production
scenario runner then loads a native Lost session, cancels Load, exits to menu
and loads a running session. The 14-step continuation scenario is
`terminal-defeat.json`; its saves are synthetic snapshots in a temporary store,
not install writes. No original GUI or original death-time observation is
claimed. The tick-1 lethal-command witness does not measure the DYING window.

## Contract coverage

- Death, script loss and message 255: game terminal action test; Victory and
  Continue refused; Exit/Load do not change world bytes, HP/MP or XP.
- Disabled Load, all failure/cancel routes, successful replacement, input and
  pause: UI terminal tests, including an empty store with a valid loader,
  unreadable file, refused opener, nil opener, missing town and missing font.
- Native continuation: frontend-only Lost latch round-trip; old envelope
  default; immediate restored failure; disclosure returns to failure; stopped
  world digest/tick. Inner simulation version remains 67, envelope version 1.
- Retained behaviour: existing DYING/revival, Victory/Continue, session-start,
  refused outgoing opener and F2/F3 tests. No admission timing or guarded-set
  policy changes.

## Gates

- `gofmt` and `git diff --check`: clean.
- `go test -trimpath -count=1 ./...`: PASS. Focused game/UI terminal and
  load/outcome tests: PASS. The additive residue descriptor changes the current
  native envelope fixture hash; old decode fixtures remain valid. The hostile
  gob-map allocation test now derives its field delta by name.
- One completed `check-release-tests.sh en ru` chain: six packages, 112 gated
  tests; EN 112/112 and RU 112/112, zero lacked a subject. Earlier attempts were
  instrument failures: restricted cache traversal, Git safe-directory settings
  and sandbox denial of the hidden native video helper. The completed chain
  used explicit process-local Git settings and approved helper execution.
- `check-scenarios.sh` filtered to `1080`: EN 1/1, RU 1/1. The new terminal
  continuation scenario also passes through the EN/RU release witness.
- `check-no-game-assets.sh`: `clean (tree scan)`.
- `check-preserved-installs.sh`: 181 files, both roots as recorded.
- Allocation sweeps before and after ledger edits: 32 ledgers plus DIV,
  `missing answers: 0`, DIV floor 618. Only DIV-610 is spent from this lane's
  610..617 reservation.
- `check-div-claims.sh`: 273 live rows, 380 distinct claims, 69 retraction
  references; no parse failure. Touched rows 219/125/137/610 use the corrected
  campaign predicates. Other rows are outside this story's audit.

## Mission census

Built `cmd/missionrun` from the story tree and separately from unchanged base
master, then ran EN `-mission 10/20 -trace -ticks 1`. `UNSUPPORTED` counts:

| Mission | Base master | Story | Delta |
|---|---:|---:|---:|
| 10 | 0 | 0 | 0 |
| 20 | 0 | 0 | 0 |

`pipeline/milestone-baseline.txt` records registration counts, not separate
UNSUPPORTED totals: mission 10 has 16 checks/27 instants/12 triggers; mission
20 has 14/15/11. Both story traces retain those exact counts. No simulation
script population changes. Binaries were written to seat temporary output,
not the installs or `builds/current/`.

## Remaining boundaries

The single fresh-context pass returned one combined disclosure/no-font gate
defect in candidate `19fc6a33`. Its correction preserves the terminal input
latch across replacement pages and clears it only when the notice closes.
This is UI state only: no native field, simulation hash, or rescue timing
change. The new App regression restores Lost, removes the font before or after
disclosure, and verifies ten frames plus gameplay shortcuts leave tick/hash
unchanged; Escape returns to failure and exits. A UI regression checks that a
subsequent ordinary fontless dialogue does not inherit the latch.

Correction checks: focused game/UI terminal, DYING/revival, Continue, refused
opener, outgoing and load-notice tests PASS; the installed terminal App/native
continuation witness PASS separately on EN and RU. `gofmt`, diff check,
no-assets and preserved-installs checks PASS. The earlier full chain above
belongs to the initial candidate; the seat owns the corrected merge chain
and reruns the unchanged independent reviewer overlay. No second review.

Merge and current-build publication belong to the seat.
No network recovery, XP penalty, original save writer, primary/companion
admission change, original runtime witness or GUI automation was added.
`DIV-219` retains the timing conflict; `DIV-125` retains the guarding Unknown;
`DIV-610` names authored presentation/cancel/native compatibility.
