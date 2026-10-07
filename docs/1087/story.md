# 1087 — Player Defend

Select owned units, press D or the Defend panel cell, then click an actor on the
map or minimap. Defenders accompany and cover that actor. A protected actor
inside the selection acquires enemies in place; one outside it keeps its order.

Initial base: `7677d697fe31dcdd7ce17cb3bdec724e795e1d64`. Reconciled with
master `2fae001a02496f6360808ba155514185575b86db` in `49f60ffd`. Research pin
advances from `26c755b2be9513b9ec90edba6527db4fdd613dc4` to its descendant
`1172d41a90ef928aa7345f76d0d3dcf0fc987bc9`. Root owns the sole independent
review, serialized landing, final merge gates and current-build rebuild.

## As built

- D and panel cell 3 reassert mode 4. Map release targets an actor only; minimap
  down/drag targets an actor and release spends the mode. Empty ground emits
  nothing. Right-click cancels arming without clearing selection. Focus loss
  drops Defend and cannot rearm it. Empty/foreign selections cannot arm it.
- `MapDefend` queues `KindGroupDefend`. Kind/tag collect the selected members
  into a fresh order-none command group, separate from authored script groups.
  The selected subject receives state `0xc`; other members receive state `8`,
  subject identity and range `3`. Actor ID zero remains valid.
- Existing escort closes first, then covers using the protected actor's
  diplomacy row. State `0xc` runs the standing, reach-limited picker and drops
  stale pursuit when the pick is lost; this also activates script-set acquire.
  Replacement group orders clear escort fields. Admitted direct attack replaces
  Defend; book casts retain their existing interruption policy.
- Master integration retains its shared headless idle-pointer and actor-point
  adapters. D/panel replaces an unissued item-cast aim. Admitted Defend cancels
  and refunds a reserved scroll. Missing/off-map subjects and
  dead/off-map/unowned/absent commanded members are bounded no-ops, including
  any existing scroll reservation.
- Actor/group/route fields already persist. Defend adds no native-form field or
  version bump. Later master save fields and compatibility fixtures remain.
  Loading does not enqueue or reissue a command.

The pinned claim reader confirms `AI-CMD-033`, `AI-CMD-054`,
`AI-FOLLOWSET-116`, `AI-FOLLOWRANGE-115`, `AI-ACQUIRE-002`,
`AI-FOLLOWDEATH-119`, `AI-PANEL-123`, `AI-KEY-125` and `AI-MINIMAP-124` at
the current pin. Existing escort/cursor authority also includes `AI-CMD-032`,
`AI-CLICK-050`, `AI-DEFEND-111`, `AI-FOLLOWTAB-113`, `AI-FOLLOWGAP-114`,
`AI-STATE-011` and `TOWN-355`. No raw research or original binary was read.

## Proof

Pre-review executable evidence below ran at `49f60ffd`; the bounded R1 correction
has separate evidence below. Commands used `GOCACHE=<seat>/.cache/go-build` and
process-local `GOFLAGS=-buildvcs=false` for the nested 386 helper.

- Focused sim/UI/game tests selected
  `PlayerDefend|Escort|Headless|CommandPanel|Scroll|Save|Upgrade|Restore|FogWalk`:
  pass. New regressions cover superseded scroll aiming and admission-before-refund.
- `gofmt`, `git diff --check` and `go test -trimpath -count=1 ./...`: pass.
- `TestReleasePlayerDefend1087AppPanelMapMinimapAndNative`: pass separately
  on preserved EN and RU. Actual App key/map events send actor 35 to protect
  actor 0 and observe movement. Panel/minimap self-target reaches `0xc`.
  Independent raw installed BMP rectangles verify Defend enabled/selected/spent
  and Retreat disabled. Fixture-only teleport and exposed fog make the actors
  reachable; installed identities, stats, queue and simulation stay production.
  Before/following/acquire snapshots restore exact native forms, empty queues
  and equal 33-tick continuations on both roots.
- Built `.local/againrom-1087.exe` and `.local/missionrun-1087.exe` from this
  worktree. `scenarios/1087-player-defend.json` passes all 22 App steps on EN
  and RU, including D twice, self-target, F2/F3 continuity and cancellation.
  Scratch saves stay under `.local/1087-resume-{en,ru}`.
- EN missionrun `-trace -ticks 1`: M10/M20 unsupported **0/0**, unchanged
  from the baseline. M10 script counts are 16/27/12; M20 are 14/15/11.
- Asset guard: clean tree scan. Allocation sweeps before/after ledger edits:
  32 namespaces, missing answers 0. Divergence scanner: 290 live rows,
  438 distinct claim IDs, no malformed rows, 72 retraction-bearing rows for
  clause judgment. The touched DIV-201 retains its TOWN-091 retraction boundary.
  Preserved-install check: 181 files, both roots unchanged. No commit in the
  candidate-only history has a `Co-Authored-By` trailer.

The observable result is a functioning installed App Defend route and native
continuation, not a lower script-node census. No desktop input or screenshot
claim is made. The full combined EN/RU release chain is deferred to root's final
merge gate; the focused pair above is the branch witness.

### R1 correction

The sole review found that left-up at a new minimap position could retarget
Defend. Repeats now require the held left button, per `AI-MINIMAP-124`.
The new four-case regression covers stationary/moved release, held drag and
both-button drag; mode/grab clearing and left-action precedence remain intact.
`PlayerDefend|Minimap|CommandPanel` tests pass in UI/sim/game. The unchanged
`.cache/review1087-pass1/overlay.json` command passes all seven top-level probes;
overlay and both probe SHA256 values still match the sole review report.
The ownership-loss diagnostic is unchanged and gains no new fidelity claim.
`gofmt`, diff whitespace and asset checks pass. No second fresh review or full
branch gate chain is run; root owns final merge verification.

## Remaining limits

DIV-232/261/288 close. DIV-230/201 retain Retreat, Cast and other unfulfilled
clauses. DIV-321 drops only the stale Tab clause. DIV-007 retains healing, idle
turn and per-subtick re-aim. DIV-558 discloses cover preference/radius,
stand-ground score substitution, cell-based step-away rounding, actor-pass
cadence and Unknown original target-death/global teardown. No Retreat/R,
F5–F8, general AI redesign, original SAV writer or original-runtime witness is
added. Pending unapplied UI commands retain the existing save policy;
continuation evidence concerns admitted canonical orders.
