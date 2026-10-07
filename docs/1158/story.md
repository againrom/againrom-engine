# Unit combat-block acceptance

Original Unit, Human and Humanoid attack, defence, base attack and modifier
blocks are compared independently of the decoder that feeds the importer. The
reader also compares all six Human/Humanoid skill-XP dwords. Implementation and
focused proof are complete; final gates, review and landing remain pending.

## Scope and authority

SAV-UNITPROG-156 defines the raw sequence: UA6 24 bytes, UBE 22, U114 24 and
UD4 64. Amended SAV-HEROXP-063 defines six raw dwords in H1CC, interpreted as
signed i32, separately from the aggregate already checked by story1156.
SAV-HUMRUN-444 and SAV-HUMLOAD-445 bound retained blocks and traced load
behaviour. HERO-DMG2-029 fixes the active elemental selector permutation.

Production supplies decompression, structural starts and the explicit archive
to DTO permutation. The reader validates complete tagged-object coverage and
an injective permutation; expected values and counts never come from decoded
blocks, ActorHoldings or ActorGraph. Equal-valued records remain separate.
Token identity, archive index, DTO index and native EntityID are distinct.
The new HumanoidXPOff locator contains no expected field value or width.

Complete imported Document is checked before Snapshot. Complete live source
blocks and the existing entity counterparts have separate comparisons. The
instrument also requires each live source's explicit DTO projection binding.
Independently read stage and HP bytes name the existing non-live boundary;
their scalar acceptance remains in story1156. Missing expected living or
dying source bases fail. Other non-live records remain counted and named.

## Measured proof

Focused EN and RU runs each discover 102 paths: 62 world, 39 city and one
unreadable. The unreadable path is
2026-08-27/EXP-0261-owner-runs/game9000.sav, with Bsg& instead of Asg&. It is
named outside decoded-structure comparison, using the existing census policy.

Each root checks 2837 tagged actors: 2746 world and 91 city; 1792 Unit,
1045 Human and zero exact Humanoid. The four raw blocks and XP arrays account
for 405238 selected wire bytes. There are 2435 complete live combat bases,
311 named world records without that basis, zero mismatches and zero resume
refusals. City records receive complete raw-to-codec Document comparisons;
this is not city App acceptance. Counts precede resume, and future refusals
remain named in the census.

TestReleaseMilestone2UnitCombat1158 is registered in the existing gated-test
manifest. On both roots it reads preserved 2026-08-24/game0021.sav, SHA256
7acf1d56d3a98e388a527ae386f225036cc01551273480a4fafc52fa8fcf817c. Its 36
tagged actors pass title LOAD and map-menu LOAD separately. Each door passes
ordinary menu SAVE, removal of only the private original copy, fresh FrontEnd
native LOAD and 20 individually advancing steps. World hashes and retained
selected Document values are checked separately at every step. Current-state
persistence comparison never supplies an expected original-byte value.

Fixture controls cover every selected byte on two equal-valued distinct Units,
a Human and an exact Humanoid, under zero and duplicate Token identities.
They reject each missing/duplicate/short/long block, wrong class, unexpected
Unit XP, dropped/aliased DTOs and invalid structural/origin inputs. Live
controls reject every source-block byte loss, every byte in each of the six
XP dwords in both source and entity arrays, skill/protection/resistance losses,
the existing combat counterparts and absent/aliased projection bindings.
The synthetic both-door App witness carries nonzero blocks and signed/high-bit
XP values. It passes the same source-free 20-step continuation on each door.

Two compiling private Go overlays establish the independence boundary. A
shared raw decoder mutation changes UD4[63] from 0xc0 to 0x40 on two synthetic
actors. ActorHoldings and the importer agree on both complete mutated UD4
blocks, and TestUnit1156NonzeroAppContinuation stays green. The new reader
rejects two Document and two World differences before Snapshot; both new App
doors also fail. The mutation detaches the decoded block before editing it,
so the raw Body remains unchanged.

The second overlay changes only EncodeSave's owned retained Document. It
flips the same UD4 tail byte without touching World or the caller's Snapshot.
Fresh native LOAD retains equal World hashes; the explicit selected Document
comparison rejects the loss on both App doors. Unmodified baseline runs before
and after the overlays pass the explicit raw comparison and both App doors.
No checked-out Go source was changed by either control.

Reproduce the private controls with native Python:
`python ../review/story1158/run-overlay-controls.py`. The script writes all
Go overlays, JSON and logs explicitly as UTF-8. It rejects compilation/setup
failures and verifies the expected comparison failures. Evidence is in
review/story1158/overlay-results.json and the four overlay-*.log files above
the repository. No original bytes or generated artifacts are tracked.

Post-reconciliation focused Unit1158/Unit1156, Player/Group/Projectile controls,
the shared trailer locator, the gated-test scanner and EN/RU UnitCombat plus
UnitScalars corpus/App runs pass. The measured combat population above is
unchanged. Evidence is in review/story1158/reconciled-controls.log,
reconciled-en.log and reconciled-ru.log. gofmt, git diff --check and the asset
guard pass. No source-loss defect was observed in these focused baselines.

## Resume work and remaining gates

The initial dependency was published story1156 checkpoint
1b9dbb89f6d3327f841fffd8499913d52af7753f. Authorized dependency merge
dd53afbbd8ad209a82e7e8a58f7dc4ef99e3e8d5 includes fetched published story1157
036ccca6647b2696edf03416797848ee1b6a99b5. All measured post-reconciliation Go
content is that merge. The knowledge pin remains
83c9efe4d86e4d88ad74b2cde6bb410709ce28d0.

The merge preserves Trailer, Players, Groups, UnitScalars and Projectiles
release registrations and M2 output labels alongside UnitCombat. All fourteen
TestMilestone2 instruments are listed by the tagged build. DIVERGENCES matches
the incoming 1157 blob, including DIV-944/945/968/969/1003. No divergence IDs
from the reserved DIV-1082..1089 range have been consumed.

On the seat's direction, reconcile final engine main after story1157 lands.
Full Go, paired release, full milestone-2, asset/divergence and preserved-install
gates and the sole fresh-context review are still pending on that final base.
No final gate chain, independent review, main landing or builds/current
promotion is claimed by this checkpoint.

Aggregate XP/scalars, Token/Group/Player contents, owned-reference topology,
mover/order state and inferred modifier/derive/callback semantics are outside
this slice. Exact Humanoid runtime acceptance remains unwitnessed in the
discovered corpus. There is no original executable run, original-format world
writer or install mutation. Script behaviour is unchanged; mission-10/20
unsupported-node counts were not remeasured during this checkpoint.
