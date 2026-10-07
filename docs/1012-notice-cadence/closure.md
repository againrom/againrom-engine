# 1012 — closure

Branch `1012-notice-cadence`, base `c3ac3d4`. Research pin `02a1403` (unchanged; not bumped, per
brief).

Gate: `go build ./...`, `go vet ./...`, `gofmt -l` over tracked and untracked Go files, and
`go test -trimpath -count=1 ./...` all pass with no game install present.
`bash scripts/check-no-game-assets.sh` is clean. The deletion set against base is empty (no file
under the repository was deleted; the measurement tool that produced the evidence below lives
outside the repository, per golden rule 2).

## Result

No code changed. `pkg/game/announce.go` is untouched. `docs/DIVERGENCES.md`'s `DIV-011` is edited
in place: its `ROM1 behaviour` cell now cites `DLG-LIFE-005` alongside `TRIG-MSG-023` and states
what each does and does not establish; its `Reason` cell carries the measured shipped population.
Status stays `OPEN`, narrowed to the residual gap `spec.md` names (a post-dismissal repeat of the
same trigger). `DIV-134` was reserved and is returned unused (recorded in `docs/DIVERGENCES.md`'s
own allocation paragraph).

## The twelve-aspect matrix

| Aspect | Verdict | Evidence |
|---|---|---|
| Data | N/A | No data format touched. |
| Runtime state | PASS | The announcer's rising-edge memory (`Announcer.prev`) is read, not changed. `TestAnnouncerMergesAdjacentFiringsOfARepeatingTrigger` and `TestAnnouncerSeesEveryNonAdjacentFiring` already witness its two edge cases and pass unchanged. |
| Simulation | PASS | `pkg/sim/script.go`'s per-pass re-firing of a repeating trigger is confirmed unchanged by reading (`scriptPass`, `pkg/sim/script.go:804-841`); no `pkg/sim` file is touched, so the byte form and every digest are untouched by construction. |
| Player input | N/A | Panel dismissal is unchanged (0066/0079 code, not touched here). |
| AI | N/A | No unit decision is reached. |
| UI / HUD | PASS | Reconciled: this build's one-notice-per-held-span behaviour and ROM1's discard-while-open rule produce the same visible panel count for the shipped case measured (`spec.md`). |
| Triggers / scripts | PASS | The subject of this story; measured directly over all 28 shipped campaign maps on both roots (below). |
| Inventory / equipment | N/A | Not touched. |
| Persistence / save-load | N/A | Not touched. |
| Campaign / session | N/A | Not touched. |
| Shipped content | PASS | 28-map, both-root static census plus a 2000-tick unattended dynamic sweep, below. |
| Interactions with existing mechanics | PASS | `DIV-008`'s message-255 collision reads the same recipient-side discard rule (`DLG-LIFE-005`) this row now cites; the two rows are consistent and neither contradicts the other. |

No in-scope GAP. This story's result is the corrected ledger row, not a code change.

## Integration witness

Two instruments, both run against the two preserved installs
(`<seat>\gameversions\en`, `...\ru`), both over the same 28 shipped campaign maps
`scripts/campaign-sweep.sh` and `pipeline/check-milestone.sh` already use.

**Static.** A tool built against this worktree loaded each mission's compiled `*sim.Script` the
same way `cmd/missionrun` does (`game.OpenArchives` → `game.LoadDefinitions` → `game.StartMission`)
and counted triggers carrying a message instant (`Op == 2`):

```
EN: 231 message-raising triggers, 231 one-shot, 0 repeating.
RU: 231 message-raising triggers, 230 one-shot, 1 repeating (mission 140, trigger 5, latch 5,
    not inert, 2 used pairs: nearest<=const, groupcount>const).
```

EN and RU otherwise compile to matching trigger structures for mission 140; this one trigger's
`Once` flag differs between the two shipped releases at the same map position. This is a shipped
data fact about the two localized releases, not an implementation divergence, and is recorded here
rather than as a new `DIVERGENCES.md` row.

**Dynamic.** `cmd/missionrun -mission <N> -trace -census -ticks 2000` (unattended, no waypoints —
the same "step and issue nothing" mode `scripts/campaign-sweep.sh` drives) was run for each of the
28 missions on both roots — 56 runs. `-trace` prints every trigger firing pre-merge, at the
simulation's own per-pass granularity, which is the raw signal the announcer samples above. Across
all 56 runs: 2 message-bearing firings total, both on triggers with `Once` set (mission 101 trigger
16, mission 50 trigger 13), both firing once at tick 6 and never again — by construction, since a
one-shot trigger cannot repeat. The RU mission-140 candidate above never fired in this drive: its
condition (`nearest <= const` and `groupcount > const`) never held over an unattended party that
never moves.

Read together: zero shipped message-raising triggers were observed to repeat across consecutive
passes on either root, and the one structural candidate that could (RU mission 140) is conditional
and unconfirmed. This is what `spec.md`'s conclusion — "not many" — is measured from, not asserted.

**What this does not witness.** The unattended drive does not explore conditions that depend on
player action (party position, kills, item possession), so a trigger requiring one of those could
hold in real play without ever being seen here. The static census is exhaustive over trigger shape
(every message-raising trigger, on or off the unattended path) and is what carries the "0 EN / 1
RU, both roots, all 28 maps" figure; the dynamic sweep adds only that the RU candidate did not fire
unattended, not that it cannot fire at all.

## Research reconciliation

- `TRIG-MSG-023`'s own confidence line already states it cannot see recipient behaviour (Medium for
  "no simulation state"). This story's finding is that `docs/DIVERGENCES.md`'s prior `DIV-011` text
  read the claim's High-confidence send-side finding as if it answered the client-side question,
  which the claim's own text disclaims. No claim is refuted; `DIV-011`'s citation was corrected to
  the claim that actually answers the question it asks.
- `DLG-LIFE-005` was already cited in `docs/0169-script-message/contract.md` ("High for the drop")
  and in `TestAnnouncerMergesAdjacentFiringsOfARepeatingTrigger`'s own failure message ("the
  original's own discard rule hides most of it"). `0169`'s own code and tests already understood
  this; `DIV-011`, authored separately when the ledger was introduced, did not carry it. This story
  brings the ledger row into agreement with what `0169`'s own artifacts already stated.
- No claim was refuted by this story.

## Script-gap census

Unaffected. This story runs no code path `pipeline/check-milestone.sh` measures differently; the
census is unchanged from master. Verified directly, from this worktree:

```
go build -o /tmp/mr ./cmd/missionrun
AGAINROM_ASSETS=<en> /tmp/mr -mission 10 -trace -ticks 1 | grep -c UNSUPPORTED  ->  0
AGAINROM_ASSETS=<en> /tmp/mr -mission 20 -trace -ticks 1 | grep -c UNSUPPORTED  ->  0
```

Both match `pipeline/milestone-baseline.txt`'s current values (0 and 0, all instant and check
opcodes on missions 10 and 20 are supported as of master `c3ac3d4`). This story does not move the
census; it was never expected to — the change made is to a ledger row, not to any script arm.
