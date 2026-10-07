# 0124 — equip from the pack: verification

Everything below was re-run in the lane seat on the **committed** tree, not taken from a task
report. Where a claim is about a line being load-bearing, it was witnessed by **reverting the
line**, never by reading the assertion.

## The gate

`go build ./...`, `go vet ./...`, `gofmt -l $(git ls-files '*.go')` and
`go test -trimpath -count=1 ./...` are all clean on the branch head, after the last merge of
`origin/master`. `scripts/check-no-game-assets.sh` reports **clean (tree scan)**. No file is deleted
against the merge parent — `git diff --diff-filter=D --name-only origin/master HEAD` is empty.

Five task commits, one per entry, with `SDD-Task: 0124-equip-from-the-pack/T1` .. `/T5` and no
others; **no commit on this branch carries a `Co-Authored-By` trailer**, checked by
`git log --format='%h %(trailers:key=Co-Authored-By)'` over the whole branch.

## What was witnessed by reverting

**AC-8, the number moves.** `mw.rearm()` in `tick` (`pkg/game/world.go`) is the one statement that
writes the new combat block onto the live entity. Commented out, re-run:

```
--- FAIL: TestRearmMovesTheEntitysDamageBaseToTheNewLoadout
    DamageBase = 999, want 10 (Hero.Recompute for the equipped sword) — got the pre-equip value 999
--- FAIL: TestRearmFallsBackToTheStartingWeaponWhenSlotOneIsEmpty
    DamageBase = 0, want 10 — the recompute for the starting weapon, slot 1 being empty
```

A **number**, 999 to 10, not a flag. Restored and re-run green.

**FR-17, the pick-up log fades.** `v.stepPickupRows()` in `command` is the whole of the dwell.
Commented out, re-run:

```
--- FAIL: TestAPostedRowExpiresAfterExactlyItsOwnDwell
    the row is still standing after all 240 of its own frames, want it gone
```

**The click hotfix under this story.** Reverting its guard turns a right press over the open window
back into one move order and a tap back into a cleared selection — the owner's two symptoms
exactly. That witness is recorded in `docs/hotfix/LEDGER.md`'s own row; the fix itself reached
master separately, as `872487b`.

**The double-click threshold** (T3) and the **name-join order** (T1) were each witnessed the same
way by their executors, on the boundary case rather than the happy path: removing the frame
decrement reddens the *one frame beyond the threshold* test, and swapping shape and material in the
reconstructed name reddens the round-trip on all five synthetic weapons.

## The byte form

`formatVersion` is **34**. The strongest available evidence that the equipment section is the *only*
change to the form is arithmetic rather than argument: `TestThePinIsThePreStoryPinPlusTheEquipment`
strips exactly the new section out of this branch's pinned world and reaches

```
preEquipPinDigest    = 0xb9105ae160cf28d9
preEquipRoutedDigest = 0x7cf2b3820d012994
```

and those two values are, byte for byte, `origin/master`'s own live `pinDigest` (`hash_test.go`) and
`rtfDigest` (`routeform_test.go`) at the commit this branch merged. A world with no equipment set
therefore hashes exactly as it did before this story, which is P-3 measured rather than asserted.

`TestThePreviousVersionFormIsRefused` gained two cases for this section, keeping all four of
master's. `pkg/sim/corpseloot_test.go`'s deliberate version tripwire was re-pinned to 34 and its
note now records that it has fired three times, naming the story that moved it each time.

## What was NOT done, and where it stopped

- **Only a weapon folds.** Twelve slots exist in the world and in the byte form, and the command
  writes whichever slot an item's own code names — but `Loadout` carries a weapon, so equipping into
  any other slot would change the picture and no number. That is spec Scope, stated up front.
- **P-4, the disclosed hole.** Equipment a mission *starts* with does not reach `pkg/sim`. The first
  weapon equipped therefore displaces nothing back into the pack — the starting weapon is
  superseded. Every later equip displaces properly, which is the decoded behaviour
  (`ITEM-EQUIP-006`: the displaced item goes back at the source index). Closing this needs a seeding
  door at mission open and was cut rather than half-built.
- **No take-off, no drag-and-drop.**
- **FR-12's drawing half is wired but not pixel-tested.** `refreshEquipment` recomposes the figure
  and the twelve slot icons through the same composer `buildInventorySubject` uses, and the equip
  path is tested end to end on the numbers; no test asserts the recomposed pixels changed.
- **`data.WeaponFromCode` does not bounds-check the three collection indices it reads** off a code
  before indexing. Every caller in this tree passes a code composed by this tree or read from a
  parsed table, so no reachable path trips it — but a hostile 16-bit code would, and this is written
  down rather than fixed, because T1 had already landed when it was found.

## Premises in the brief that turned out wrong

- **`pkg/game/inventory.go` "runs once, at mission open" was already half-false.** `0112` had
  landed `refreshPack`, a per-frame rebuild of the pack area. What ran once was the **figure and the
  slot icons**; that is what this story changed, and the header now says so.
- **Byte-form version 30 was stale before it was written.** It was allocated to this lane, taken by
  T2, and then overtaken twice by lanes that landed first (`0122` at 31, `0117` at 32). The number
  moved to 34 mid-story. The lesson is already in the brief's own warning about version-spelling
  test names; the version-free naming is what made the correction a constant edit and a fixture
  edit, with no test renamed.
- **The line numbers in the brief were right**, both of them: `inventoryPresent` at
  `pkg/ui/inventory.go:432` and `inventoryLayoutRects` at `:143`. They were verified by reading the
  functions, not by trusting the grep.
- **The notice machinery does not fit the pick-up log**, and the reason is stronger than "several
  lines": the notice is **modal** — it gates the whole map arm, carries a dismiss button and an
  advance seam that can navigate out of the mission, and holds one string. FR-15 records it.

## Where each criterion is witnessed

Every id below is witnessed by a test that fails if the behaviour is removed, not by a sentence.

- **AC-1** `TestEquipSlotForAnswersFieldBAlone` (`pkg/data/equip_test.go`) — every B from 0 to 15,
  with 0 and `ItemClassCarried` named explicitly.
- **AC-2** `TestWeaponFromCodeRoundTripsEveryNameTheResolverAnswers` — five synthetic names, every
  numeric field compared. Swapping shape and material in the join reddens all five.
- **AC-3, AC-4** the equip move's own tests in `pkg/sim/equip_test.go`: `[a,b,c]` with slot 1 empty
  becomes `[a,c]` and slot 1 = `b`; with slot 1 = `d` it becomes `[a,d,c]`.
- **AC-5** the same file's refusal cases — absent entity, index past the container, slot outside
  1..12 — each compared on the **marshalled bytes**, so a partial write would show.
- **AC-6** `TestThePinIsThePreStoryPinPlusTheEquipment`, `TestThePreviousVersionFormIsRefused` and
  the differing-slot hash test in `pkg/sim/binary_test.go`.
- **AC-7** `pkg/ui/invequip_test.go`: inside the threshold raises one request and a second drain
  yields nothing; one frame beyond raises none; two different cells raise none; the figure box
  raises none.
- **AC-8** witnessed by reverting `mw.rearm()`, output above.
- **AC-9** `pkg/game/equip_test.go`'s four refusal cases, each asserted on `w.Hash()` being
  unchanged — an empty cell, a code naming no slot, a weapon-slot code that does not resolve, and a
  nil definition table.
- **AC-10** is the weakest line in this story and is stated as such. `refreshEquipment`
  (`pkg/game/world.go`) recomposes the figure and the twelve slot icons from the world's own
  equipment through `composeInventorySubject` — the very function `buildInventorySubject` already
  calls at mission open — and `refreshPack` already rebuilds the pack area on the same per-frame
  seam. **No test drives `refreshEquipment`**, so what is witnessed is the equip reaching the
  world's equipment slots and the numbers moving; that the recomposed picture then changes rests on
  it being the same composer, not on an assertion. Named in the *not done* list above.
- **AC-11** `pkg/game/pickup_test.go`'s `pickupRowsFor`: three codes, two equal, give two rows at
  quantities 2 and 1, in first-seen order; and a re-ordered variant, so the order is the walk's and
  not the map's.
- **AC-12** `pkg/ui/pickup_test.go` drives `command` with rows standing and compares the selection
  and the emitted orders against the same frame with no rows — byte-identical.
- **P-1** `internal/archtest`'s stdlib-only import check and its source scan over `pkg/sim`; both
  run in the ordinary `go test ./...` and both cover the new file.
- **P-2** every slot number reaching `sim` comes from `data.EquipSlotFor(code)`; there is no other
  producer, and `EquipSlotFor` reuses `validEquipSlot` rather than spelling a bound of its own.
- **P-3** the two digests above, equal to master's own.
- **P-4** `TestRearmFallsBackToTheStartingWeaponWhenSlotOneIsEmpty` exercises the fallback branch
  directly, and reverting the recompute reddens it on a number too.
- **P-5** `pkg/ui/inventory.go` and `pkg/ui/pickup.go` import only `image`, `image/color` and the
  render tier; the allow-map in `internal/archtest/dag.go` is fail-closed on that and pinned by
  `TestUITierAllowanceIsPinned`. A row crosses in as text and a count; no item code is named there.
