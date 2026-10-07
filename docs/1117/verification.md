# Pickup completion evidence

The observable result is the installed mission's pickup continuation, not a
reduced script-node census. On EN and RU, the ordinary App pointer transfers
one dropped installed item at tick 28; the next gameplay tick changes state
2 to 12 at tick 29. F transfers at tick 3 and completes at tick 4. Ordinary
menu SAVE and a fresh FrontEnd LOAD preserve exact World bytes/hash at both
cuts; 33 subsequent production mapWorld ticks match. This is an installed
headless dispatch witness, not a GUI or original-runtime claim.

Instrument: `TestReleasePickup1117ClickKeyAndNativeBoundaries`, using only
controlled hero relocation and presentation-fog revelation. The item, actor,
map, order dispatch and native persistence are installed/production-owned.
The pre-change click test failed with state 11 instead of pending state 2.

Focused sim/game tests pass for 16 transfer phases and three hostile distances,
replacement orders, quest/missing-sack controls, dead/off-map actors, committed
transit, older queued commands and atomic malformed-native refusals. The
original `TakeSack` tests still pass without changing that primitive.

The pre-story missionrun build from master `9404ec98` reports M10=0 and M20=0
UNSUPPORTED nodes. The story worktree at code commit `16e1b00e` reports the same
0/0 using `-trace -ticks 1` and the lawful EN root. The counts are unchanged;
`pipeline/milestone-baseline.txt` carries those missions' decoded populations.

## Final reconciled evidence

Merge `1a3790e6` includes school master `7f0f7939` and preserves research
pin `1172d41a`. On that production tree:

- One paired release invocation passed EN 148/148 and RU 148/148, with zero
  missing subjects. The gate named this exact worktree but could not resolve
  its Git stamp without the safe-directory setting; HEAD was checked separately.
- Scenarios `1005` (two), `1087`, `1089` and `1090` passed on each root: ten runs.
- The story's own built missionrun passed the full 28-map census on both
  roots against `pipeline/milestone-baseline.txt`; every UNSUPPORTED count
  remained zero, including M10/M20 = 0/0. Both unattended drives ended at tick
  304 with 4/36 units moved and one fallen, matching the recorded census.
- No-game-assets, divergence-claim and seat-tree guards passed. The claim
  checker reported 73 rows referencing retracted claims for interpretation;
  this story relies on the unchanged progress arm 7, not its retracted clause.

The first full Go run at `1a3790e6` failed only the two fail-closed exported
World method inventories: the new writer was not yet classified. Commit
`db5c68e1` adds `CompleteSackPickup` to the method pin and writer list, without
changing production code. Its focused inventory/pickup tests and corrected
`go test -trimpath -count=1 ./...` passed. The release, scenario and census
evidence above remains attributed to `1a3790e6`, not to a fabricated rerun.

After the gates, preserved-install checks passed for all 181 recorded files
on both roots. No original process, GUI drive, install write or SAV lane was
used. Remaining approach/refusal/ranking debt is named in the story contract.
