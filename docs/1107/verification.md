# Saved current profile verification

## Candidate state

Corrected code candidate: 63f0dcd376b2db0ad2f7b79c87692393d8c3f4b1.
Published master base: f9fc5071172d3d1b4e69998c1987bef1d10edd68,
including corrected1104, saved books1105, cell-trigger overlay1106 and
holdings1108, plus completed city sales1102. The seat retains landing control.
Research pin: 1172d41a90ef928aa7345f76d0d3dcf0fc987bc9, moved forward
from f393bd2d6305c94160580de3753cb851f55cf3ec with the city-sale base.
The two combat claims were reread at this exact pin and remain active.
The final evidence-only commit changes this document, not the tested code.
The sole fresh-context review returned36e8e0b3 for R1/R2 in
pipeline/reviews/1107-adversarial.md. Both are corrected below. There is no
second independent review, lane landing or builds/current update.
No original-game process, desktop input or install write was performed; the
release gate ran the implementation's native cutscene helper.

The initial push request for evidence commit
204b24662dea544836b3ac92cd01cab411767438 to
the archived implementation repository, branch
story1107-nonparty-current-profile, was rejected by automatic approval
before execution. The reviewer classified the complete story branch as a
potential private-code export and required explicit payload/destination
authorization. No alternate transport was attempted. The owner subsequently
authorized the complete1107 code/tests/docs/pin payload and its destination.
The seat published checkpoint3644dec49a6692bd66eeaaee05504fdba25acf0e;
the lane's live origin query confirmed that exact named branch SHA before
the1108 reconciliation. The publication blocker is resolved. This is not an
independent-review or landing claim.

## Reconciliation with master 3bcf5912

Reconciled candidate d9ebc3f9aedfad1168ccdd66524f08168e78b3c8 merges published
df47dfb82306f8d24695ec095a4c961ba5ae989d with master
3bcf5912b8e7fd7e5b9450462c4e48707c9bffe5, 70 commits past the prior
f9fc5071 base. Research pin moves forward from
1172d41a90ef928aa7345f76d0d3dcf0fc987bc9 to master's own
682184eb8e2571ee9b23d2ee614b9f9a459a63d3.

Three files conflicted: docs/DIVERGENCES.md (both DIV-738 and master's five
new rows kept, in landing order), pkg/game/save_test.go and pkg/sim/world_test.go
(both sides' additions unioned; world_test.go's two exported-method lists stay
alphabetical). The textual merge could not see two further breaks: the current
Snapshot type now carries every field 1108..1124 added, so save_test.go's
form72/Sales-descriptor re-encode (previously pinned93b8ddb8, now
279f4c59c7a1da, matching what master alone pinned as its own current fixture
before this merge) and the main currentReleasedSaveFixtureSHA256 pin (now
fdcdce20bd3996be90ff36e0f7246f302ac1c8bb982bb10eb147b54b484cd791) both moved.
save_pre_quickspells_fixture_test.go's decode comparison carried the same
break silently (no conflict marker, since only one side ever touched that
line) and needed the same pre-1107 World peel already used elsewhere in
save_test.go. No production code changed; all three fixes are pinned-value or
comparison-basis corrections following the merged descriptor.

Every claim story1107 cites (HERO-DMG2-029, HERO-CLAMP-030, HERO-RESIST-012,
SAV-HUMRUN-444, SAV-HUMLOAD-445, SAV-HUMMUT-448, UNIT-DERIVE-003,
SAV-UNITFLD-049, SAV-UNITPROG-156 and the SAV-REGENWIDTH-528..SAV-REGENWIRE-532
family) was reread at682184eb: none is retracted, amended or otherwise changed
since1172d41a. DIV-634, which this story amends rather than owns, cites
SAV-DEATH-051 and SAV-DOC-053, each with a retracted clause; both retractions
predate1172d41a and are unrelated to this reconciliation, and DIV-634's own
text cites their surviving, unretracted order clause, not the refuted counts.

TestOriginalProfile1107ElementalBlowAppAndNativeContinuation (pkg/game)
re-runs the sole review's two corrected findings on the merged code: raw
selectors1..5 against protections10/20/30/40/50 leave HP82/88/86/84/90 through
both App doors (12 cases: 6 protection/selector combinations x App/mission
entry), and negative Fire-50 against base20 leaves HP70, not the
pre-correction80. All twelve subtests pass. The reviewer's own
review/story1107-scoped probe files are not part of any repository tree
(review/ is untracked everywhere); this persisted, committed regression is
what the correction pass added to keep that proof alive, and it is what ran
here.

Gates on d9ebc3f9: go test -trimpath -count=1 ./... passes across every
package. gofmt -l and git diff --check are clean. check-no-game-assets is
clean. check-claim-citations resolves1552 distinct citations against1890
claims (unmerged master alone: 1545). check-div-claims selects333 live rows
of333 citing485 ids (unmerged master alone: 332/332 citing475; the delta is
DIV-738 and its own claim set). One paired check-release-tests invocation
covers both roots at EN157/157 and RU157/157 with zero missing subjects
(unmerged master alone: 156/156 each; the added test is
TestReleaseOriginalProfile1107NaturalFractionAppAndNative). check-scenarios
passes50 of50. check-preserved-installs finds181 files, both roots unchanged.
GOFLAGS=-buildvcs=false was used for the release-test invocation only, to
route around this worktree's cross-account .git ownership breaking the native
cutscene helper's own go build; no other command needed it. No behaviour
changes relative to df47dfb8 beyond the three repinned/rebased fixture
comparisons above, all of which reflect the merged Snapshot descriptor rather
than a semantic change.

## Sole correction pass

R1: HERO-DMG2-029 maps raw third-component selectors1..5 to the address-order
protection indexes0,3,2,1,4. The importer now uses that permutation rather
than subtracting one. The original review probe now leaves HP100 through
the selected100 protection for all five selectors; raw2/raw4 previously left80.
The native canonical selector meaning and byte form do not change. Already
persisted canonical selectors are not guessed back into source wire values.

R2: HERO-CLAMP-030 retains signed source-current target protection until the
third-component multiply, then clamps that component. Base20 with Fire-50
now deals30 and leaves HP70, not80. The target's source-current authority
selects this arm. Native/native-retired target sheets retain their existing
0..100 input clamp. Sim tests cover-32768, -50,150 and32767, both unchanged
native policies and64 event/hash-equal native steps.

ElementalBlowAppAndNativeContinuation loads independent wire profiles through
both App doors. With protections10/20/30/40/50, selectors1..5 leave literal
HP82/88/86/84/90; Fire-50 leaves70. Each of the12 cases issues a new attack,
uses ordinary menu SAVE, loads a fresh FrontEnd and compares96 further hashes.
The same candidate carries imported holdings and a saved book. The negative
protection proof reaches combat; it is not only an input-retention assertion.

The pre-correction failures are retained in correction-R1R2-before-36e8e0b3.txt
and correction-production-before.txt. All original reviewer probes pass on
the final code in correction-review-probes-63f0dcd3.txt, including65536
independent register cases and cold-native-LOAD potion retirement. Rerunning
the existing probes is correction verification, not another confidence pass.

The1102 merge changes the AGS gob descriptor. A read-only overlay on exact
published96f5d36 produced the genuine11616-byte pre-sales form72 envelope:
SHA256d747175f99163887f0a54d47024f3baa1e9f9792fc63ae224234a314d29e4d31.
The historical checkout remained clean. The frozen envelope is decoded and
compared separately from1102's current-descriptor/form72 anchor93b8ddb8.
The combined current-descriptor/form73 hash is4d217143. Original full literal
digests remain in save_test.go, and the genuine form73/earlier controls remain.
Producer source, overlay and output are correction-form72-producer* under
review/story1107. No new writer or version stripping created the old envelope.

## Requirement evidence

FR-1 / DD-1: TestOriginalProfile1107ExactAllPlayerClassesAndAliases
transcribes Unit, Humanoid and Human across sparse/repeated Players and shared
group references. It checks detached values and no partial result after late
archive corruption. The join test covers party, dead, off-map, zero-ID,
unmatched and ambiguous-target cases. LateRefusalLeavesActiveSession refuses
ambiguous source IDs, bad active/elemental selectors and zero mana divisors
without changing the active session or carried party. Both empty and full
nonzero mana pools exercise the fault policy.

FR-2 / DD-2: ImportIsAtomicAndNotARebuild checks signed combat values,
unchanged pools, clock, RNG, active attack, skills and book. It imports signed
periods -32768/-1, modifiers -32768/-101 and remainder bytes255/199 without a
consumer call. Invalid late batch entries write nothing. RawHighPoolWordsAreNotLoadArithmetic
keeps u16 maxima65535 and mana65530 on LOAD; the subsequent signed consumer,
not LOAD, interprets those words. Final sim pool admission validates the saved
maximum, not the temporary constructed maximum.

FR-3 / DD-4: SignedWordRegeneration has ten literal vectors covering narrowing
before the upper bound, signed periods, negative modifiers, unsigned byte
reload, dword product/accumulator overflow, raw high words and retained
remainders at a bound. SignedAppActionAndNativeContinuation drives original
App LOAD, a new attack, ordinary mission SAVE and fresh App native LOAD.
After dispatch it saves negative mana/byte255 again and loads a second fresh
FrontEnd. Ninety-six further hashes agree. Health cases include32766->32764,
1->-1 and10->-56. The native same-tick decay policy is accounted for explicitly.
The second admitted mana call changes -1/255 to0/54 in the living case.
HealthCrossingStillRunsManaAndNativeDeath independently drives a real move,
checks health-1/254 and mana-1/255, then96 event/hash-equal steps.

ActualBlowUsesCurrentThreeComponents issues a new attack with a literal
damage34 and victim HP66. BothAppDoorsNewRegenAndNativeContinuation checks
the ordinary positive-range values, a newly charged attack and96 equal
native hashes. Native zero-divisor corruption is refused by the decoder.

FR-4: RealPotionRetiresCachedRosterAcrossNativeLoad uses an actual Humans-row
roster and carried potion command. The fixture explicitly grants one point
of potion headroom. The cached post-command rebuild changes source-current
to native-retired and emits DIV-738's diagnostic. The second physical pair
remains. Ordinary AGS and a fresh FrontEnd retain the retired sheet and agree
for32 ticks. ZeroManaFaultPolicyAndNativeRetirement verifies that retirement
switches back to the existing native arithmetic rather than silently retaining
the source-current consumer.

## Genuine predecessor controls (DD-3)

Form73 adds one provenance byte per entity; old native forms default native.
The form72 fixture was produced by exact
a95f00adb64239d66d2e95104b8ee82a56ebf5f4, not a stripped current encoder:
5307 bytes, SHA256862ac08d35dedc215349d887a0123fd7aabe8dda16b01c5cb71955974dfdf359.
Its actual book, second pair, regeneration values and late-dead record survive
migration and64 event/hash-equal steps. Independent form72 literal pins remain.

A genuine WIP d71426b61fd7030b77eb4c29f523a3ded3cf8d04 writer supplied the
whole form73 envelope:12711 bytes,
SHA2563d0f9b5e89cfd486b71cae099023490e7ef275ed2666cf790fbe4ea2a9736d1b.
Its world is5390 bytes,
SHA256bcdcd9a6d962d8e0233e48100ffb63d4a4577a155e3a984de740633d9713fc0b.
GenuineForm73EnvelopeAndContinuation decodes it and reproduces the old
producer's complete after64 world hash:
SHA256e77ead4a1bfcc0485b9f34e2d21e5f54215cfe00567c9116ea207d627f5bef34.
Producers and unmodified form73 log are retained in review/story1107.
Compatibility-test peels elsewhere are not this predecessor evidence.

## Observable original-input result

Independent published-d521 baseline: source2026-08-02/game0009.sav,
SHA25660267c82072c77446ab9b34913318e89eab8f70e49f3510ae64aaaf423819bd6,
M20 Player3 Unit MapUnitID8, HP7/15, health period40 and fraction75.
Both EN and RU published App LOAD lost the fraction to0. Its natural hit55,
damage1/3, active5, defence45 and Reaction/Mind/Spirit75/3/5 already matched;
those fields are not claimed as natural baseline defects.

TestReleaseOriginalProfile1107NaturalFractionAppAndNative verifies literal
source offsets independently of the new projection. On both EN and RU, both
App LOAD doors now retain fraction75 with HP7/15 and period40 unchanged.
The source is already victorious. Its modal captures SAVE keys, so the test
uses the production SAVE seam, then fresh App LOAD. Ninety-six terminal App
frames retain tick0, fraction75 and equal world hashes. This is not evidence
of an incoming original clock or first regeneration tick.

Before logs and probe source remain under review/story1107/seat-*.
After logs are natural-en.txt and natural-ru.txt. This changed production App
result is the observable outside the story's synthetic test files.
No builds/current change is claimed by the lane.

## Focused checks and remaining debt

Focused signed/action/legacy/span tests pass in focused-signed-final.txt.
Affected-package run found the corrected1104 span test's old456-byte current
length expectation;1107 now expects457, retaining the exact old-form peel.
The final full-suite result supersedes that intermediate failure. The first
full run also exposed omitted mapload form73 controls. Their literal old
digests remain unchanged; a literal457-byte peel and the one-byte length
increments restore those controls.

Allocation sweeps before/after the DIV738/DIV634 updates both print DIV-762
and missing answers:0 (alloc-signed-before.txt, alloc-signed-after.txt).
DIV-739 through DIV-745 remain unused and reserved.

DIV-738 retains constructed phase12/rate1, the native death-consistency callback
substitute, native retirement and zero-divisor refusal. Original callbacks,
loaded state16 lifetime, idle/action-end phase, scheduling and first post-LOAD
chronology are not closed. DIV-634's bounded pool admission remains.
Full container state, complete current-load restoration and a full-world
original SAV writer are mandatory separate debt. No source blob is retained.

## Final correction gates

All paths below are under review/story1107. The final correction code is
63f0dcd376b2db0ad2f7b79c87692393d8c3f4b1, reconciled with published
f9fc5071172d3d1b4e69998c1987bef1d10edd68 and exact research pin1172d41a.

- go test -trimpath -count=1 ./...: PASS
  (correction-go-all-63f0dcd3.txt).
- One paired check-release-tests invocation: EN141/141 and RU141/141,
  each0 lacked a subject, aggregate exit0
  (correction-release-63f0dcd3.txt). This includes the retained city-sale,
  holdings, book, current-profile and pool composition.
- Original review probes: PASS, including all five selector probes,
  negative Fire protection,65536 register cases, late atomic refusal and
  cold-native-LOAD potion retirement (correction-review-probes-63f0dcd3.txt).
- gofmt, git diff --check and check-no-game-assets tree scan: PASS.
- check-div-claims:293 live rows,441 distinct cited IDs and74 rows citing a
  claim with a retraction entry (correction-div-63f0dcd3.txt). The last
  count is a report, not74 defects.
- Allocation sweeps before and after ledger reconciliation print DIV-770
  and missing answers:0 (correction-alloc-before.txt,
  correction-alloc-after.txt). No new IDs were consumed.
- check-preserved-installs:181 files, both roots unchanged
  (correction-preserved-63f0dcd3.txt).
- No deleted files relative to final master; Co-Authored-By trailer count:0.

The evidence-only follow-up changes this document, not the tested code. The
earlier pre-review results below remain historical evidence, not substitutes
for this correction chain. No additional independent review was performed.

## Pre-review consumer and holdings evidence

All paths below are under review/story1107.

- gofmt and git diff --check: PASS.
- go test -trimpath -count=1 ./...: PASS at7a67f02c
  (go-all-7a67f02c.txt). After reconciling published1106, the focused union
  sim/game/mapload/formats/sav/internal/gatedtests passes at587949fc
  (reconcile1106-focused.txt). No new consumer code changed during reconciliation.
- One completed check-release-tests invocation on587949fc: EN138/138 and
  RU138/138, each0 lacked a subject (release-587949fc.txt). This includes both
  natural App doors and the restored books/cell-overlay composition.
- check-no-game-assets: clean (tree scan), no-assets-587949fc.txt.
- check-div-claims:288 live rows,428 cited IDs,71 rows with partial-retraction
  references (div-claims-587949fc.txt). This is a report, not71 defects.
  DIV738 now cites the five promoted arithmetic claims. DIV634 retains its
  bounded pool policy and names the signed consumer; its amended dead/living
  distinction still matches the positive-health join.
- check-preserved-installs:181 files, both roots unchanged
  (preserved-587949fc.txt).
- No deleted files relative to final master. Co-Authored-By trailer count:0.

At78714792, the complete affected-package union
sim/game/mapload/formats/sav/internal/gatedtests passes
(reconcile1108-focused-78714792.txt). The merge of root98a7fdaf changes
ancestry only after the96f5d36 holdings reconciliation. No signed arithmetic,
byte-form layout or source-current/native-retired policy changed. The prior
full Go and EN/RU consumer chain above was not repeated.

reconcile1108-composition-78714792.txt records the explicit union witnesses.
HoldingsRearmBooksAndOverlayCompose retains the six-byte cell overlay, then
replaces starter stock, runs Rearm, restores the book and current profile,
and finally writes pools with tick0. The signed App-action cases now import
one held weapon, one armor item and three ordered-effect items. All five
values survive the new attack, signed health/mana dispatch, two fresh native
LOADs and96 subsequent steps. The item check counts canonical values across
the actor and any native corpse sacks; it does not establish original callback
behaviour.
RealPotionRetiresCachedRosterAcrossNativeLoad now consumes the SAV-imported
potion, without replacing stock after LOAD. Late profile refusal follows
successful stock/book staging; late pool refusal follows stock/book/profile
staging and leaves the old live session and its ordinary native SAVE intact.

Final gofmt/diff checks and no-game-assets scan pass. No files are deleted
against98a7fdaf. The divergence report selects291 live rows and433 cited IDs;
74 rows reference a retraction entry (reconcile1108-div-78714792.txt).
The three added matched rows are1108's unchanged DIV746..748 rows.
Allocation sweeps immediately before and after ledger reconciliation both
print DIV-762 and missing answers:0; the latter is retained in
reconcile1108-alloc-after.txt. No additional divergence IDs were consumed.

Missionrun was built from this worktree with its resolved gitdir and explicit
GIT_WORK_TREE. missionrun-exact-stamp.txt records revision587949fc and
modified=false. On the EN lawful root, mission10 and mission20 each report
UNSUPPORTED=0 (final-m10-trace.txt, final-m20-trace.txt). Exact earlier master
038f4b0b also reports0/0 (baseline-m10-trace.txt, baseline-m20-trace.txt).
These counts are unchanged; this story moves the natural App LOAD result,
not the unsupported-node population. pipeline/milestone-baseline.txt records
M10 16 checks/27 instants/12 triggers and M20 14/15/11 on both roots; it does
not contain UNSUPPORTED totals.

Infrastructure failures remain visible in the earlier logs. The first shell
attempts failed before tests because of cache-path admission. At2b086fe2 the
native helper build lacked Git trust; at7a67f02c the sandbox denied native
cutscene execution. The final owner-context gate above passed without changing
or skipping that test. missionrun-stamp.txt is the incorrectly seat-stamped
earlier build, not final attribution. The optional386 probe did not run:
existing step_test.go:221 passes untyped3735928559 to Fatalf as int and fails
compilation (signed-and-span-386.txt). No386 execution result is claimed.
