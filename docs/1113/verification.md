# Saved Group continuation verification

Initial code candidate: `9fe0b0a37503dcc88e4c623241f4333c4aac2ad3`.
Sole review-correction code: `380d6158e119fec54f722366cd527198675a305c`.
Base reconciliation: exact published1112
`3c030c75100767540b454224d7bc5bf07678ddad`, including published1111 alias
correction `afedb39da8e1b1230c9f05df245889e691808f62`.
Research advanced without rollback to promoted
`dde13d1d75c192a14250dec948eeb388af4275e0`.

## Contract witnesses

All inputs below are independently authored synthetic bytes or native worlds;
no original save, asset, runtime image or extracted experiment code was added.

| Contract surface | Executed proof |
| --- | --- |
| Canonical identity and membership | Distinct Group identities with equal selectors; empty registry versus absent mode; empty Group; ordered B/A/C rather than EntityID order; null/discordant owner and independent references; detached snapshots |
| Actual decision and mutation | Per-actor Move destinations distinct from GroupAI cell; Guard/stance, ordinary player move, current handover/death/removal; fresh successor after current removal, appended member and registry reallocation; live Group rate rather than stale Entity cache |
| Patrol and Roam | Four-cell ring with duplicate cells, first-equal cell cursor and nonzero re-anchor latch; accepted/rejected Roam draws, counter50/51/255 cases, impossible rectangle atomic refusal; no invented Group-cell-to-member-destination copy |
| Target orders | Native script Attack/Defend/Follow uses current membership and hashed Authored provenance; incoming8/11 uses exact nonzero source-key binding under stage0; different source keys choose different actors; already-admitted corpse target remains distinct from follower stage; raw zero range branches before stop-distance fallback |
| Unsupported operation boundaries | Selector collision and unbound membership retained through SAVE; count writes no fabricated value and dependent trigger does not fire; later missing order/invalid target refuses primary before earlier member writes; source/map/runtime/native-ID coincidences, missing key, wrong class, nonzero stage and ambiguous corpse key do not become bindings |
| Native persistence | Malformed span/count/bool/identity/late target leaves receiver unchanged; current mutable registry, separate lists, raw numeric operands and typed target bindings round-trip; removal expires binding but retains source key; genuine frozen form77 synthetic fixture preserves absent mode |

`TestSavedGroups1113BothDoorsOrdinaryMenuFreshLoadAndNextAction` reaches the
main-menu and mission-menu original LOAD routes, executes the first incoming
Move without replaying a command, then makes an ordinary map command and uses
the menu SAVE, fresh App LOAD and20 subsequent ticks.

The incoming-escort and arbitrary-Patrol App tests SAVE **before** first
dispatch. Fresh LOAD chooses the same next action and state: both8/11, two
different exact source keys, and the four-cell duplicate Patrol ring. The sim
tests also cover native commands, already-admitted corpse targets and late
invalid-target atomic refusal. This is executable state, not retained metadata.

The unchanged natural1111 release control checks12 literal living Group counts
through direct resume, both App doors, native fresh LOAD and the actual first
script writes: game0003 has4→3,3→3,5→1; game0001 has1→3,3→4,7→3,8→2;
game0021 has1→2,4→3,7→1,3→3,5→2. No absent-selector ALM fallback is used.

## Sole review correction

The seat's sole fresh pass, `pipeline/reviews/story1113-pass1.md`, returned one
P2: ff acquisition decided for an OffMap member, either acquiring a hostile or
clearing a retained movement order. The correction adds only the missing
current-presence predicate to that callback. Structural membership, GroupCount
and the fresh-successor walker are unchanged. This restores the existing
0164 FR-1/FR-3.3/FR-4 contract; it introduces no new divergence or byte form.

The exact unchanged review tests were supplied through a read-only Go overlay,
not edited or copied into production. Before the correction, the two named ff
tests failed; all other selected controls passed. Afterwards all six selected
independent probes plus the existing native OffMap control passed, including
634 atomic truncation offsets. The excluded StoneCurse and script-escort
observations were not promoted into findings or changed.

The new checked-in regression starts with a hidden CURRENT head before a visible
successor, in peaceful and hostile worlds. Through production Step, native SAVE,
fresh LOAD and35 next ticks, the hidden Entity and raw order remain unchanged,
membership order remains intact and GroupCount stays2. The visible successor
still clears its peaceful move or acquires the hostile. The retained fixture
generator from the earlier1114 handoff was renamed from `.go` to `.go.txt` after
the architecture census reported that ignored build-output package; no generator
content or production state was removed.

Correction gates on `380d6158`: focused unchanged review probes PASS; `gofmt`
and `git diff --check` PASS; final `go test -trimpath -count=1 ./...` PASS;
paired release EN149/149 and RU149/149,0 missing subjects; no-game-assets PASS.
The same required native cutscene escalation used only process-scoped Git
configuration and test-owned output. The28-map milestone passed on both roots,
with mission10/20 unsupported counts0/0 unchanged and the same4/36 moved,
1 fell,304-tick unattended drive. Corrected binaries passed0152 and1005 on
both roots with fresh isolated correction save directories. Preserved-install
name/size checks passed for181 files. Research and divergence ledgers did not
change in the correction. The seat owns replay of the same probes on the
published correction; no second fresh review is requested or claimed here.

## Initial final gates

- `gofmt` and `git diff --check`: PASS.
- `go test -trimpath -count=1 ./...`: PASS on the code candidate.
- `check-no-game-assets.sh`: clean tracked-tree scan.
- Paired `check-release-tests.sh` over EN/RU: PASS;149 of149 ran on each
  root,0 lacked a subject. One asset-free census,8 selected packages.
- `check-milestone.sh`: PASS,28 maps per root. Missions10/20 have0/0
  unsupported nodes, unchanged from `pipeline/milestone-baseline.txt`.
  Both unattended drives moved4 of36 units,1 fell,304 ticks; the lost outcome
  is printed, not asserted as a playability verdict.
- Headless0152-save666 and1005-doll-carry-over-worn: PASS on both roots using
  the candidate binary and separate test-owned EN/RU save directories.
- `check-preserved-installs.sh`: PASS,181 files, name/size baseline unchanged.
- `check-div-claims.sh`: PASS;306 live rows,465 distinct cited IDs,
  291 retraction IDs,81 flagged rows. New DIV-789 uses the narrowed025
  Group-cell/evaluation contract, not guaranteed actor wandering. DIV-792
  explicitly uses the amended035/460/547 hit-only order repair, not universal
  missing-to-null. DIV-386 was clarified after publication: the still-missing
  script17 setter is separate from supported loaded Roam, and its closure no
  longer demands an unproved actor-wandering result. This final clarification
  changes documentation only. Existing unrelated flags remain inherited debt.
- Allocation sweep bracketing ledger edits:32 ledgers,0 missing answers.
  DIV-786..792 used; reserved793 unused. Seat floors remained seat-owned.
- No deleted paths or `Co-Authored-By` trailers in the composed story delta.

The first full-suite attempt exposed stale form77 fixture tail offsets and the
missing77 readable-version census entry. Those were corrected; old historical
digest constants stayed unchanged. The final full suite above is after that
correction. Release setup first stopped before tests on MSYS absolute cache
creation; a scoped relative cache resolved it. A subsequent sandboxed paired
attempt failed only the native cutscene witness with Access is denied on both
roots. The unchanged paired gate was then rerun with approved tool escalation,
without ACL, global Git configuration, profile, registry or install edits.
Its account could not print the Git label; a separate process-local safe-directory
read verified the exact candidate SHA and the clean worktree.

The old-city release fixture was only read from the specifically authorized
location into a test-owned ignored copy. Gate output and scenario saves remain
under ignored `builds/1113/`; no private fixture bytes or derived content are
part of publication.

## Observable result and remaining limits

The candidate headless game, outside these new test files, now reports
`saved Groups 18 RESTORED; bounded continuation issues: []` while running
0152 on each lawful root. Previously that LOAD report excluded incoming
Group/actor AI entirely. The carried-state and subsequent command scenarios
still reach their expected result. The script census did not change.

No physical desktop or original ROM1 session was driven. No original SAV writer,
complete original pointer lifetime, first-loaded global chronology, arbitrary
callbacks, unnamed fields, cached mover paths or complete default construction
is claimed. DIV-786..792 record the specific safety/native-policy boundaries.
The source-order block is not a native pointer image; current native effects and
validated target bindings are separate from retained raw source keys.

No master merge, shared build replacement, seat edit, denied dependency use or
independent review was performed by this lane. The seat owns the one fresh
adversarial review, the unresolved authority dependency and serialized landing.
