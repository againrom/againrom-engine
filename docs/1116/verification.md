# Verification

## Observable result

The school now composes every installed column frame instead of two endpoint
cuts. `TestReleaseSchoolColumnClassTransitionInstalledPixelsAndSound` drives
the production school picker and shared room compositor. Its independent BMP
oracle reads the literal installed frame path, decodes source RGB and compares
all 148x208 pixels at (168,176), including opaque black. EN and RU focused runs
each matched 954,304 pixels over 31 forward/reverse frames. The middle frame
differs from the fighter endpoint at 29,451 pixels in each direction.

Both runs also verify all ten endpoint skill-mask centers, no intermediate
mask hit, and two exact installed Rotate waveforms through a recording player.
Two characters generated through installed ChargenParty data have different
classes and figures. Literal next/previous corner hits change the shown member
and name immediately; the composed figure crop changes at 10,138 pixels.
Audio output hardware and original ROM1 runtime were not exercised.

Untracked composed PNGs for frames 0, 4, 8, 12 and 15 are under the seat's
`review/story1116/en/` and `review/story1116/ru/`. They are review aids, not the
pixel oracle. No extracted byte or generated image enters this repository.

## Contract coverage

- Frame cache/order/size/opaque alpha: `TestLoadSchoolColumnAllFramesOrderedOpaqueAndValidated`.
- Strict 83/84 ms gate, all frames, both directions, one step after a long gap,
  stop and sound count: `TestSchoolColumnStrictGateBothDirectionsAndNoCatchup`.
- Selected skill/price, semantic Train refusal, same-class/no-distance
  cancellation and reversal: `TestSchoolColumnSelectionControlsAndRetargetPolicy`.
- Pure reads, statistics mode, dialogue paint, independent diamond clock,
  other rooms, reentry and new game: `TestSchoolColumnReadsRoomBoundariesAndIndependentDiamond`.
- Backward time, missing art/frames/device/bank:
  `TestSchoolColumnBackwardClockAndDegradedArt`.
- UI-side intermediate/invalid frame defense:
  `TestSchoolColumnIntermediateFrameSuppressesIconsAndMaskHits`.
- Shared live/headless paint dispatch and update/read distinction remain
  covered by `TestSchoolAnimationAdvancesInAppPaintButNotUpdate` and the
  existing school diamond and tavern tests.

## Final gates

Implementation base is `9404ec98c174518a41ec601168a4aa9bda6ad245`, reconciled
without a conflict at `ca0669dbc8046f4b4c7446387305c5fb269c9329`. The final code
and tests are `8950fa983017a2040873f4354c772dde2c071842`; later verification
record edits do not change them. Research remains pinned at
`1172d41a90ef928aa7345f76d0d3dcf0fc987bc9`.

- `go test -trimpath -count=1 ./...`: PASS on 8950fa98. The initial ca0669db
  run caught an unseeded new clock field in the whole-FrontEnd reset fixture.
  The correction seeds it, classifies it as a retained process service and
  checks its identity and reading explicitly; the population gate is intact.
- `scripts/check-no-game-assets.sh`: PASS. `check-div-claims.sh`: PASS as an
  instrument; 72 pre-existing rows cite a claim with a retraction entry. None
  of the changed school claims is retracted. `check-seat-tree.sh`: PASS on
  ca0669db. Only the explicit clock reset fixture and its field comment changed
  afterward.
- `check-scenarios.sh <EN> 1005` and `<RU> 1005`: PASS 2/2 per root on ca0669db.
  Both doll/shop picker scenarios are unchanged by the later fixture correction.
- Built `cmd/missionrun` from this worktree, then ran each of missions 10 and
  20 with `-trace -ticks 1` on EN. The instrument runs through its 64-tick
  minimum. Unsupported counts are 0 and 0. Script populations are respectively
  16 checks/27 instants/12 triggers and 14/15/11, exactly the pre-story values
  in `pipeline/milestone-baseline.txt`; this UI story changes neither.
- One `check-release-tests.sh <EN> <RU>` invocation on 8950fa98: PASS EN
  147/147 and RU 147/147; zero missing subjects. Its printed Git label was
  `not-a-git-checkout` because the elevated helper environment lacked the
  ephemeral safe-directory setting. The explicit worktree path and unchanged
  clean code commit were checked separately; no checkout changed during it.
- `check-preserved-installs.sh`: PASS, all 181 file-manifest entries across
  the two roots remain as recorded. This is the manifest check, not a new
  all-asset cryptographic hash audit.
- `git diff --check`: PASS. No file deletions, research-pin change, denied
  story1107 ancestry or `Co-Authored-By` trailer was introduced.

Gate logs are untracked under the seat's `.cache/story1116-final-*` prefixes,
with the measured code commit suffix. No landing or current-build replacement
has occurred.

## Remaining debt

`DIV-142` retains original paint scheduling/shared-clock uncertainty.
`DIV-143` retains unidentified original slot state and marker consumers.
`DIV-814` discloses immediate request/retarget/visit policy. No exact ROM1
interruption or wall-clock-duration claim is made. The other town/interior
animations and SAV work are not part of this candidate.
