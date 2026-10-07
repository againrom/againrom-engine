# story1193 verification

## What moved

No production Go source changed. `pkg/sim/story1193_escort_stall_test.go`
(new) documents current behaviour with a synthetic fixture built from the
owner's own save numbers. `docs/DIVERGENCES.md` gains `DIV-1316` (type
`UNKNOWN`, table "Authored where research is silent"). Nothing in
`pipeline/check-milestone.sh`'s script-gap census could move, and it did not.
The observable result this story produces is `docs/1193/story.md` itself,
read against the owner's own two save files rather than against a
constructed retreat, plus the new test as a witness a later fix has
something to flip.

## Correction pass

The story's first pass (commit `bbe44a0`) investigated mission 121's *other*
Fat troll (unit #29, `u61`, Vertical Bridge) and concluded the owner's report
was explained by that troll's guard-leash mechanism. Two coordinator
corrections, cross-checked against the owner's own saves
(two AGS saves in `engine/saves/`, read-only, never modified — copies used for a
throwaway probe deleted before this commit), found that troll idle and
uninvolved in both saves, and identified the actual troll (unit #30, near
the Horisontal Bridge) standing on the bridge deck, its attack order held
continuously across ~1.73M ticks (`Stall` 3 then 13, position (35,15) then
(37,15)), blocked by an occupancy conflict between its 2x2 footprint and up
to five 1x1 party members packed onto the deck. `docs/1193/story.md` is
rewritten around this finding; the entity-29 leash material is kept as a
correctly-scoped secondary finding, not an explanation of the report.

## UNSUPPORTED script-node census (missions 10 and 20)

```
go build -o /tmp/mr ./cmd/missionrun
AGAINROM_ASSETS=<en install> /tmp/mr -mission 10 -trace -ticks 1 | grep -c UNSUPPORTED   -> 0
AGAINROM_ASSETS=<en install> /tmp/mr -mission 20 -trace -ticks 1 | grep -c UNSUPPORTED   -> 0
```

`pipeline/milestone-baseline.txt` does not carry a matching "UNSUPPORTED node
count" row for missions 10/20 (it carries script checks/instants/triggers
counts per mission, both unchanged at m10 `16/27/12` and m20 `14/15/11` for EN
and RU). Both measured UNSUPPORTED counts are 0, matching an install with no
unrunnable script nodes at tick 1 on either mission. This story changed no
sim, script or AI-decode source (the new file is a test, not production
code), so these counts are unchanged from master by construction, not by
measurement alone.

## Gates run (this worktree, `wt-story-1193-bridge-target`, base `86d55b0`)

- `gofmt -l .`: reports one pre-existing, unrelated file
  (`pkg/game/installtext.go`, last touched by `12fea8e`, not this story) —
  left untouched, out of scope. The new test file is clean.
- `go test -trimpath -count=1 ./...`: all packages `ok`, no game asset
  required, including the new
  `TestAnAttackerBlockedByItsTargetsEscortHoldsTheOrderAndNeverAdvances`.
- `bash scripts/check-no-game-assets.sh`: clean.
- `bash pipeline/check-scenarios.sh <EN root> 1193`: 1 of 1 ok (the
  entity-29 leash scenario; unchanged).
- `bash pipeline/check-scenarios.sh <EN root> 1060`: 28 of 28 ok (mission-121
  reachability scenario included, unaffected).
- `origin/main` had not moved past this branch's own merge base (`86d55b0`)
  at reconciliation time; no rebase was required.
- EN/RU release tests: not run. This story touches no screen, shipped-data or
  save behaviour (a test file and documentation only).

## Claims cited

`AI-SCORE-069`, `AI-GROUPSEE-068`, `AI-GUARD-012` — all High confidence, each
read end to end via `cd knowledge && go run ./tools/claim <ID>`. No claim
answers the entity-30 mechanism itself (`DIV-1316`, type `UNKNOWN`).

## Existing tests that already pin the entity-29 mechanism

`pkg/sim/release_test.go`:
`TestAReleasedMemberHoldsNoneOfItsSevenFields`,
`TestAReleasedMemberWalksHomeThenStandsThereForeverAfter`. Both predate this
story (story 0106) and were left unmodified.

## New witness for the entity-30 mechanism

`pkg/sim/story1193_escort_stall_test.go`,
`TestAnAttackerBlockedByItsTargetsEscortHoldsTheOrderAndNeverAdvances`:
documents current behaviour (order held, no net advance, the walk-cancel/
reissue cycle observed) against a synthetic fixture, not against the owner's
saves or the shipped map — named so a later fix changing this build's own
behaviour has a test to flip or replace rather than patch.
