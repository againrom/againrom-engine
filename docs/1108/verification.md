# 1108 verification

Original base: `d521686f108552918e115e853542cd42142b49e6`.
Reconciled base: `5e1598cad070157c2fa40ce85c2ef6da222c5465` (1104, 1105, 1106).
Research pin: `ba21c9aa9a949023b3d678b22ca29b3a3b0cd95f`, unchanged.
Branch: `story1108-nonparty-holdings`.
Original code/test candidate: `39f9bd2a54c06309e1fecbc35eb068f5e6768c8c`.
Reconciled code/test candidate: `62ab108cbb7cdb5c7413f2ca228ad967daaaea95`.
The following documentation commit changes no code, test, runner or research pin.

## Reconciled candidate evidence

Stock and Rearm precede 1105 books and saved pools in both LOAD doors. The
detached stock stage preserves the already-restored 1106 cell tails and the
1104 second physical pair. No current-profile or per-instance-weight work from
1107/1109 is included. No files were deleted; the research pin did not move.

Three new composition controls in `originalholdings_composition_test.go` prove:

- Empty stock clears starter items, Human Rearm changes the fixture maximum,
  its second physical pair and cell tails survive, and the saved book/pools
  replace template values without advancing tick0.
- Both LOAD doors restore distinct stock/book values; a new item transfer and
  book cast survive native SAVE/fresh LOAD with matching events and hashes for
  256 further ticks. The App arm uses ordinary menu SAVE and a fresh FrontEnd.
- A valid control restores two stocks, books and pool sets. Changing only the
  later actor's mana above its maximum refuses after both stock and book stages.
  The diagnostic door returns no Mission; FrontEnd and App retain the previous
  snapshot, live pointers, hash and subsequent native SAVE.

Focused game/sim/SAV tests PASS on 62ab108c. Four focused release tests PASS on
each of EN and RU: 1108 natural holdings, 1105 non-party books, 1106 cell overlay
and next-entry continuation, and 1104 synthetic city training/SAV/AGS route.
The final paired full-release chain belongs to the seat merge, not this
focused reconciliation. The original failed paired run below remains failed.

Logs: `review/story1108/focused-final-62ab108c.log` and
`release-focused-final-{en,ru}-62ab108c.log`. Both roots retain these new native
hashes: empty Witch `d1feb28b70d3ff6b`; M40 staff `226e681bca1cc3c9`; after
death/loot/equip `7933529f07d54b66`. Source anchors and literal values below are
unchanged. These hashes belong to the composed branch, not the original branch.

No-assets PASS; gofmt and diff whitespace clean; no Co-Authored-By trailers.
The divergence guard exits0 with 290 live rows and 73 partial-retraction
advisories. DIV-746/747 cite surviving amended clauses; DIV-748 is not flagged.
Allocation sweep: 32 namespaces, missing answers0. Preserved-install guard:
181 paths/sizes unchanged. `final-guard-{0,1}-62ab108c.log` retains the output.

EN missionrun was rebuilt from exact master5e1598 and candidate62ab108c into
separate scratch executables. Both carry their exact VCS revision and
`vcs.modified=false`; both print UNSUPPORTED counts0/0 for missions10/20.
Their 16/27/12 and14/15/11 script populations match
`pipeline/milestone-baseline.txt`. `trace-{base-5e1598,candidate-62ab108c}.log`
retains both runs. No census reduction is claimed; natural runtime holdings are
the observable change.

## Observable result

Independent published baseline: seat `review/sav-nonparty-stock/`.
Candidate: `TestReleaseOriginalHoldings1108NaturalEmptyAndValuedStaff`.

- M10 `2026-08-02/game0006.sav`, SHA256
  `c6b9506e986f5dcc3e4260b68b0fae41b510d6307b33909f26c801d68329ccde`:
  later Player Human Witch, map ID21, actor offset15484. Both original LOAD
  doors now import an empty pack instead of three template potions code0x0e06,
  kind3, price50, effect(8,1,0x03c00064). Original outcome is Won: its Victory
  modal intentionally blocks menu SAVE. This fixture uses the production SAVE
  callback and fresh App LOAD, not menu SAVE. Explicit HeadlessKill then reaches
  production terminal loot and proves the deleted starter potions do not return.
- M40 `2026-08-15/game0017.sav`, SHA256
  `eafce5d6575d54fdddc7a35f57531cd3df9317006c80f7c4085866c1b02b4fe0`:
  later Player Human map ID32, actor offset27305. Both original LOAD doors
  restore held staff0x810e, kind2, price981, effect(41,0,327681), instead of
  template price440; armor0xf70d/125 and0xf80b/150 remain in slots7/8 and pack
  stays empty. Ordinary App menu SAVE/fresh FrontEnd LOAD preserves the hash.
  Controlled terminal-kill command, death dwell, loot and equip retain the staff;
  another ordinary App menu SAVE/fresh LOAD preserves it and the full world hash.

These are two original files consumed through both installs, not independent
recordings. Death controls are candidate runtime actions, not original-session
observations. No original game was launched and no desktop input was sent.

## Synthetic coverage

Literal archive fixture covers later Players, repeated/null references, Unit,
Humanoid, sparse armor, signed price, ordered duplicate effects and quantities.
It rejects cross-owner/worn-carried item aliasing and null inventory records.
Game tests cover complete-batch refusal, ambiguity including dead duplicates,
zero code/count, stacked equipment, wrong role/class/slot, unsupported effects,
aggregate expansion, party/excluded counters and no cell fallback. The App
refusal retains the old live hash. Unit cache tests cover None/Item/Innate/Legacy,
no repeated equip pool/teaching effect, transfer/equip/unequip/drop/loot and native
roundtrip. Existing pool-order witness confirms saved pools follow Rearm.

## Original branch gates

All logs below are under seat `review/story1108/`, outside every repository.

- `go test -trimpath -count=1 ./...`: PASS, `full-go.log`.
- `gofmt` and `git diff --check`: clean. No Co-Authored-By trailers.
- Paired `check-release-tests.sh <en> <ru>`: 8 packages, 134 gated tests,
  exit1. Each root reports exactly one failing test:
  `TestReleaseCutsceneNativeAppCompletionAndSkip`, `Access is denied` at the
  existing native witness after a successful helper build. The aggregate gate
  is not described as PASS. `release-paired.log` retains both failures.
- The unchanged native test, narrowly rerun with approved host access, passes
  on EN and RU: 90/90 frames composed, independent packed oracle6/6, all four
  skip routes return to map at unchanged tick0. `native-en-elevated.log` and
  `native-ru-elevated.log`. The sandbox narrow repeat also failed, recorded in
  `native-narrow.log`. No runner or source was modified to obtain the PASS.
- Natural holdings test independently PASS on EN and RU, `natural-en.log` and
  `natural-ru.log`: 25 M10 and47 M40 non-party stocks. Empty Witch native hash
  `1bfd2afd73243b84`; M40 initial native hash `1f9303b85c56e7a0`, after
  death/loot/equip native hash `0be00a173e72150d`. Both roots agree.
- No-assets: `clean (tree scan)`. Divergence guard: exit0, 289 live rows,
  72 rows cite partially retracted claims. New DIV-746/747 use amended surviving
  clauses, not the old head-width or universal enchantment-separator clauses.
  New DIV-748 has no flagged citation. `guards.log`.
- Preserved installs: `ok — 181 file(s), both roots as recorded`, including
  after the host-native witness. This guard compares path and size, not content
  hashes. `preserved-final.log`.

The paired gate's initial bootstrap attempts failed at `mkdir -p` on the
absolute cache ancestor. Setting `AGAINROM_GOCACHE=.cache/go-build` from the seat
resolved the same required `<seat>/.cache/go-build` cache and
allowed the one paired population run. No test executed on those bootstrap
failures. Child Git ownership used five process-level safe.directory entries.
Native narrow runs and missionrun builds bound GIT_DIR/GIT_WORK_TREE only to
the worktree's own repository. No buildvcs bypass was used.

## Mission traces and limits

Mandatory EN missionrun `-mission 10/20 -trace -ticks 1`: base d521 and candidate
39f9 both print UNSUPPORTED counts0/0. Both binaries carry exact VCS revisions
and `vcs.modified=false`; `build-stamps.log` and `base-m*.log`/`candidate-m*.log`.
`pipeline/milestone-baseline.txt` names M10 script16 checks/27 instants/12
triggers and M20 script14/15/11; these populations also match both binaries.
The baseline file contains no UNSUPPORTED line, so the before counts were
measured from the exact base, not inferred from absence. This story does not
change mission script support and makes no census reduction claim. The
observable result is the original-save runtime holdings change above.

Initial saved actor load/capacity words are not proved equal to recomputed load
by restoring stock or item weight. They remain required current-profile work,
alongside the full item/container and Effect lifecycle debt in DIV-746..748.

The original branch was published as e06ec15c. Reconciliation is on the same
named branch and awaits its one independent review and seat landing. No original
launch, install write, desktop input or rebuild of `builds/current/` was performed
by this lane. Its natural baseline and focused controls are not an adversarial
pass. Full current-profile, item/container and original-resave acceptance remain
separate required work.
