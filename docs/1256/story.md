# Story 1256: SAV milestone M7, remaining numeric domains

## Intent

A world the engine built, not one imported from an original SAV, keeps every
wide current numeric value through SAVE, cold LOAD and a second SAVE, and a
current actor Group registry has one shape for an empty sequence. A save point
never refuses.

## Authority

- `SAV-UNITFLD-049`: the fourteen stat words of a Unit record are 16-bit and
  include Health, HealthMax, Mana and ManaMax. Current values wider
  than their word are an engine value; the written word is the low 16 bits and
  a width operand carries the rest.
- Owner rules: SAV is the only format and the engine is one producer; a field is
  written from current state, then an evidence rule, then the loaded document's
  bytes, then a constructor value; a field with none is named debt, never a
  refused save.
- The original runtime reads only its ordinary words. No claim states how it
  interprets a wide synthetic value; none is made here.

## Root causes

1. Health 65535 and MaxHealth 65536 are written as the words 65535 and 0. The
   registry admitted the actor from those words alone, decoded -1 and 0, and the
   living or dying test refused it (`sim: unsupported original actor N`) before
   the width operand was applied. The same narrowed pair then re-entered at four
   later steps: source publication in `RestoreActorLoad`, the pool import, the
   dying import and the dead-actor body check.
2. The width operand added its lift to the value already held, so a value that
   admission had already lifted would lift twice.
3. A late corpse whose HP is wider than 16 bits failed the binary check
   `e.HP == int32(record.HP)`: the dead record holds a 16-bit HP derived from the
   Entity.
4. An empty Group Members, Words, Path or Patrol sequence was nil after a binary
   decode and a non-nil empty slice after an import or after the last member was
   detached.

## As built

- `pkg/game/currenthealthadmission.go`: for a current actor, the ordinary words
  and the Values widths give the full HP pair before the living or dying test;
  `admitOriginalActorRegistry` installs it. A positive full HP with a terminal
  wire stage is refused as malformed; a stale stage 1 becomes living.
- `pkg/sim/currentvalues.go`: one `restoreNumericResidue` for every ordinal. It
  applies while the low bits equal the wire anchor and sets the canonical decode
  of the wire plus the lift, so a second application changes nothing. An edited
  ordinary word wins.
- `RestoreActorLoad`, `ImportOriginalActorPools`, `ImportOriginalDyingActors` and
  the current dead-body check keep the full HP pair while the saved words are its
  low words. `applyOriginalDying` skips a current actor whose full HP is positive.
  No exported World method was added.
- `pkg/sim/originaldeadbinary.go` compares the late-corpse HP at 16 bits.
- `pkg/sim/savedgroups.go`: `cloneNonEmpty` gives every empty sequence the nil
  shape the decoder produces.

## Proof

- Reproduction on f9786837: `TestReleaseGeneratedWidePoolsSAV` (mission 10, EN)
  fails in the cold child with `sim: unsupported original actor 1`.
- `TestReleaseGeneratedWidePoolsSAV`: five actors carry HP/MaxHP/Mana/MaxMana
  tuples at the 16-bit edges and at int32 maxima (32767/65535, 32768/65536,
  65536/2147483647, 2147483647/-1, 65535/-65537); ordinary SAVE, two cold LOADs
  in fresh processes, 16 continuation ticks. Passes EN and RU.
- `TestReleaseGeneratedWideNumericSAV`: every ordinal 4..32 (including Load,
  Capacity and TypeID) lifted by a positive multiple of its word on one actor and
  a negative multiple on another, with the pools above: SAVE, cold LOAD, second
  SAVE, second cold LOAD without a tick. All Entity fields and the sample are
  equal at both cuts; actor records and operands are equal between the two SAVEs.
  Item identity numbers are assigned again by each SAVE and are not compared.
  `TestReleaseGeneratedWideNumericContinuationSAV` repeats the route over 16 ticks
  with the skill ordinals left out: a wide skill makes the live mission derive
  the hero sheet at the next tick in the uninterrupted run only, because the cold
  run starts with its derivation caches equal to the loaded values. Passes EN, RU.
- Unit tests: `TestCurrentHealthAdmission*` (anchors, malformed operands refused
  atomically, double application, stale stage, ordinary edits win, dying and
  retained wide bodies, two SAV cycles), `TestSavedGroupsEmptySequences...`,
  `TestSavedGroupsDetachingTheLastMemberLeavesNilMembers`.
- Corpus observer (all 114 AGS inputs, EN and RU, 99 missions and 15 cities, 16
  uninterrupted ticks, two cold entries of 8 ticks, unchanged comparator
  `f63ffb6a`, observer `aac447ce`) on a5380f22:
  227 of 228 cases pass, 3834 state cuts, 3606 cold cuts, 395 mission SAVs, 60
  city SAVs, 0 differences, 1 block (`en/save-20260908-223315.ags`, the cold child
  hit its 2m20s test timeout while other lanes held the machine at full CPU).
  The same case alone: 19 state cuts, 18 cold cuts, 0 differences, 0 blocked.
  Combined 228 of 228 cases, 3853 and 3624 cuts, 0 differences.
  Last recorded run before this story (b18b581): 228 of 228, 3852 and 3624, 396
  mission SAVs, 60 city SAVs, 0 differences; it was not rerun on the base.
  Receipts: `review/story1256-sav-m7/observer-a5380f22/` and
  `observer-a5380f22-rerun-save20260908/`. The frozen observer needed one shim
  (`observer_shim_test.go`, in the same directory) for a renamed test helper.
- `scripts/check-milestone2-acceptance.sh` on EN and RU separately: exit 0, 110
  original SAV (70 world files resumed, 40 city files decode-only), 0 refused, 0
  mismatches in the actor, dead, group, building and cell-record instruments;
  AGS round trip 114 of 114 exact (`m2-en.log`, `m2-ru.log`).
- Ordinary `go test` of `pkg/sim`, `pkg/game`, `internal/storyguard`,
  `internal/gatedtests`, `internal/divledger`; gofmt; `git diff --check`.

## Open debt

- DIV-1675: a Unit capacity whose low word is 0 is written as 300 and restored as
  300. Pinned by `TestCurrentUnitCapacityWithLowWordZeroRestoresTheDefaultWord`.
- Not run: the six-case owner-input domain witness for the virtual dead bodies of
  game0032 and game0033 (`review/sav-queue-4424/m7-remaining-domains/v4`). It
  targets an older engine head and an overlay this story replaced. The M2 dead
  root instrument covers both inputs (2 virtual, 347 materialized, 242 terminal
  bodies, 0 mismatches) but not retained Dead.Source health beyond 16 bits.
- Not exercised: wide Load or Capacity on an actor whose `ActorLoad` is present
  (`ActorLoadSnapshot.Validate` requires signed 16-bit words); the generated
  mission actors have none.
- Wide values were built through typed current APIs and width operands. No
  gameplay path reaches most of them, and the original runtime was not run on
  any of them.
