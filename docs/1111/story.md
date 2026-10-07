# 1111 — saved actor registry

## Result

Implemented and author-verified; awaiting review and dependency landings. A living saved actor must not disappear because its
map identifier is zero, absent from the ALM, or not a unique ALM join. Original
LOAD constructs one actor per supported source record. Ordinary native SAVE
and fresh-process LOAD retain its identity, control, appearance, and next action.

## Contract

- One exact source-record registry binds archive index and identity to native
  entity ID. Record offsets diagnose and join projections of that document;
  they are not persistent campaign identities. Repeated references bind once.
- Keep existing placement, initial-party, and script identities. Allocate extra
  IDs above the existing population and reserved original-dead identities. Never
  fabricate a map identifier or select the first ambiguous map match.
- Construct supported Unit/Human records from verified class/definition inputs.
  Apply saved ownership, position, facing, holdings, books, pools, and landed
  profile/load state through the shared binding. Validate the complete candidate
  before publication through either original LOAD door.
- Native reconstruction metadata is entity-ID keyed and separate from canonical
  HP, inventory, books, and other World state. It rebuilds person sheets, figure
  or creature tier, derivation inputs, and applicable carry metadata before the
  first frame. Control follows ownership; temporary allies do not become heroes.
- Old native saves without a manifest retain their existing reconstruction path.
  Invalid manifests or late source failures cannot replace the current game.

## Authority and exclusions

Owner-directed import/native-continuation policy is not proof of ROM1 acceptance
of synthetic saves. Research pin `659c3f69d4cb579006d4b584e0246cf309a40fc7`
supplies SAV-ACTORBIND-544 through SAV-ACTORLIMIT-548 and SAV-GRPLOAD-560 through
SAV-GRPAI-563, with amended SAV-GRPFLD-060. Source-only Humanoid and unresolved
Unit/Human constructor rows remain unsupported. Uniquely ALM-matched Humanoid
uses the owner-approved existing ALM construction/person policy for uncovered
fields, with common saved literals taking precedence. Raw class3 and raw Token
row remain distinct from SourceActor class2, the native humanoid numeric policy.
This does not establish a ROM1 Human alias or a SAV Humanoid definition row.
Natural zero-ID Humans are not presumed missing solely because a map join fails.

No original-runtime writes, original mission writer, denied1107 correction access,
active1110 worktree inspection, game assets, or install modifications. Story1109
remains unchanged. Reserved divergence numbers: DIV-770 through DIV-777.

## Surfaces and proof

Typed SAV graph; mapload construction; sim batch admission; game actor overlays;
person/render/control binding; carry eligibility; native snapshot reconstruction.

Required focused proof: independent source-only Unit/Human fixtures, zero and
nonmatching IDs, duplicate IDs, graph aliases, invalid late actor and manifest,
both LOAD doors, actual action, menu SAVE, fresh-process LOAD and next action,
temporary-versus-persistent carry, and unchanged existing placement/party IDs.
Install witnesses must use both EN and RU; fixture tests do not establish ROM1
runtime acceptance or natural missing-actor counts.

## State

Base: `e20d054ef1c91ec51847278e6613ac7400ee5dff`, verified against live origin/master.
The branch contains exact published1110 checkpoint
`1fdf689586edcfcb8b015fdec5f7c55566f831ad`. Both original LOAD doors now use
the typed Group graph and a single archive-index registry for construction and
all actor overlays. New IDs follow surviving placements and initial party IDs;
source MapUnitID is never rewritten. Effective owner also supplies retained
HasOwner/mana-reserve inputs. Exact row selection seeds only the bounded native
policy in DIV-770, then saved fields overwrite the covered state.

Native form76 adds a fixed35-byte canonical SourceBinding tail to each entity
(457 to492 bytes), preserving all form75 variable sections. The additive gob
manifest holds only entity-ID-keyed saved name and new-construction presence.
Human person/figure/carry templates rebuild from canonical source state. Missing,
duplicate or late invalid manifests refuse without replacing the live game.

Focused SAV/sim/game checks pass: graph aliases/owner precedence, preserved ID
namespaces, literal source bytes and hash, form75 count-overflow/truncation
refusal, late admission atomicity, both original LOAD doors, temporary versus
persistent carry, actual source-only movement, ordinary App menu SAVE and
fresh-process App LOAD with rendered Unit/Human names/persons and next MapOrder.
The synthetic original fixture has three distinct source-only actors where the
old unique-MapID importer constructed none. It does not establish ROM1 acceptance.
The existing natural EN holdings drive also passes both original doors and
native continuation; no new natural missing-actor count is claimed.

The branch also contains published master `2fae001a02496f6360808ba155514185575b86db`.
The full sim and mapload package sweeps pass with the form76 literal widths,
independent pin tails and legacy peel helpers. Existing mapped Humanoid
compatibility now has a literal original/native test retaining raw class3,
native basis2 and saved pools; unmatched Humanoid refuses atomically.
The genuine form75 fixture is retained in `internal/savefixture/form75.go`:
6413 bytes, SHA256 `c6a07b23647192fd225d9d30b3956987dbed56ff6be0ebd37919cb554578e967`.
It was produced by unchanged published `1fdf689586edcfcb8b015fdec5f7c55566f831ad`
in the owner-approved detached `review/story1111/form75-predecessor` tree, using
`go test -trimpath -count=1 ./pkg/sim -run '^TestProduceGenuine75For1111$' -v`.
The local producer is `pkg/sim/fixture1111_test.go` in that retained tree. Its
two synthetic actors and quantity-two weapon use no installed or owner-save
bytes. The test checks exact predecessor bytes, signed load fields, source
equipment operands, and canonical native re-encoding; it refuses truncated and
overflow-count inputs. The detached producer tree is not removed automatically.

Source-only Unit and Human movement, actual temporary-versus-persistent mission
carry, and late original failures pass. A source-only Human equips literal
quantity-two armor through KindEquip, uses ordinary menu SAVE, and removes it
through KindUnequip after fresh App LOAD. Identity assertions separately pin
archive index/key, runtime ID, Token row, map ID, entity ID, party vector,
effective owner and presentation; a matching World hash alone is insufficient.

Existing native authored Group bindings are preserved. New source-only actors
use Group0; saved Group index/selector/owner remains separate provenance. The
independent seat control found 12 positive authored group checks returning0 on
the earlier offset-remap candidate. Literal regressions for M10 game0003 and
M20 game0001/game0021 now retain counts 3/3/1, 3/4/3/2 and 2/3/1/3/2 through
diagnostic LOAD, both App LOAD doors, native SAVE callback/fresh App LOAD and
the first real script phase. A separate GroupOrder dispatcher control uses
those imported identity/owner/group bindings; its positions/order are synthetic,
not incoming ROM1 AI evidence. The EN focused release test passes.

Final code `4be5eb14348b337bc998fbaad245a9cc1bf3f94e` passes
`go test -trimpath -count=1 ./...` (aggregate0), the no-assets guard, and one
paired release invocation: EN148/148 and RU148/148, zero missing subjects and
aggregate0. The full Go and paired release logs are retained outside Git at
`review/story1111/full-go-final.log` and `release-4be5eb14.log` beneath the seat.
The divergence-claim guard passes; the bracketed allocation sweep reports
missing answers0. No Co-Authored-By trailer is present.

The failed first Go sweep exposed a sim-test import of the internal fixture
helper. Its repair keeps the architecture guard unchanged: the frozen payload
lives once in `internal/savefixture/testdata/form75.gz.base64`, and the sim test
reads it through the standard library. The final full sweep includes that fix.

The first paired release reached both roots after exposing the predecessor
city fixture's environment gate. Three failures were repaired: city inventory
comparison now includes the already-decoded Weapon/Armor/Shield and owned-Spell
operands; the old dead-Weapon proof peels form75's load footer; the enchanted
ground-item expectation includes its independently checked Armor tail. Literal
city-tail/truncation tests and all three EN release regressions pass. The city
sale retains its real ShopClick and fresh-process AGS/SAV checks; no UI or
source-clock expectation was changed. Frozen city-v3 input is supplied through
`AGAINROM_WEIGHT_OLD_CITY=review/story1109/seat-city-v3/en` under the seat.

Actual headless binary `fb5efb05` passes both 1005 scenarios, 1025 mission weight
and 1090 consumables on EN and RU: 8/8. It is not a GUI observation. No new
script opcode or compile population changed, so no milestone census rerun is
claimed. The player-facing result is retained source-only actors and usable
authored group checks, witnessed by the original/native continuation drives.

Independent seat probes on exact `c0e045fbf8e150d854dbc771fe934bb633bc7fc4`
pass on both roots: 26 worlds, 1042 living observations, all33 zero-MapID actors,
separate native bindings/party vectors/hash, and all12 authored group checks
at the first script phase. This is bounded corpus proof, not the sole fresh
adversarial review or full incoming AI continuation. Later production changes
only repair city inventory projection; registry/layout code is unchanged.

## Sole review and bounded correction

The sole fresh pass returned `0bd40bc0cb20660fcb7bc572d9ae1fa688d950fa`
with one P2, recorded in the seat's `pipeline/reviews/story1111-pass1.md`.
A repeated persistent Human archive reference minted duplicate initial-party
members and SourceOffsets, then the strict registry planner refused both LOAD
doors. The unchanged review probe reproduced both failures before correction;
its single-reference control and four late native refusals passed.

`sav.File.Party()` now projects each exact source-record offset once, retaining
first-reference order and the final effective-owner basis. `PartyWalk()` still
exposes every raw occurrence and its extent. Shared world and city consumers
therefore allocate matching unique party/source vectors before leader ordering.
Keys, runtime IDs and MapUnitIDs never deduplicate distinct records. ActorGraph,
ambiguous-ALM and duplicate-party planner guards are unchanged.

Independent production regressions emit three Humans with equal runtime/map
IDs and a repeated middle-object hero, within one Group or across two Groups.
Both original LOAD doors and both App menu entry points retain three distinct
bindings and one named leader, saved HP, final owner/reserve inputs, source
indices/keys, entity/party IDs and rendered names. Ordinary menu SAVE followed
by four fresh-process LOAD/next-movement runs passes. A separate later-Player
alias retains owner2/reserve63 rather than the earlier Token owner1/reserve17.
Late invalid source fields, a distinct-record key collision and a duplicate
native manifest leave the active App, art, town, selection and World unchanged.

Both city alias forms also restore two unique members, retain their full roster
through ordinary native SAVE and fresh FrontEnd LOAD, and preserve the existing
explicit alias original-SAV export refusal. No alias writer support is claimed.
The first synthetic city test omitted its fixture's separate framed state and
campaign tail; using its existing full container reframe corrected the test
input, without changing any production city or writer code.

The first correction full Go sweep found one older spellbook test that used
Party's raw duplicate count. It now checks those same two occurrences through
PartyWalk, and separately checks Party's unique member and identical saved book.
No spellbook production path or refusal was changed.

Correction focused probes pass, including the unchanged reviewer file
SHA256 `9eb483a3f0eee3c57a71623972d638d652e08cdcf79a9fde3616895c642a4149`
through `review/story1111/correction-review-overlay.json`. Final correction
code/test commit `841ef53a13ddda63113f7f3e948a7ef48ec9d23f` contains production
fix `f8899a218c4818fc0bf6b8aee5083e39a61ea5e8` and passes the full
`go test -trimpath -count=1 ./...`, no-assets guard, and one paired EN/RU release
invocation: 148/148 on each root, zero missing subjects, aggregate0. This includes
genuine75, frozen city-v3, source-only carry, city sales/fresh processes and the
three natural subjects'12 authored Group checks. Unchanged reviewer alias and
late-native-refusal probes also pass again on that exact commit. Retained logs
are `review/story1111/correction-841ef53a-{full-go,release,no-assets,review-probes}.log`
beneath the seat. The preserved-install name/size guard reports181 files, both
roots unchanged; it is not a content-digest claim. gofmt and diff whitespace
checks are clean, with no Co-Authored-By trailer or research-pin change.

The bounded visible result is that a supported repeated persistent source
reference now opens as one controllable/rendered actor, instead of refusing
LOAD, and remains unique through menu SAVE/fresh-process LOAD and next action.
No new script opcode or compiler population changed, so no new milestone census
or GUI observation is claimed. There is no second fresh review or lane
master-merge authorization. Reconciliation with final1107/1109/1110 landings
remains seat-owned. Denied1107 corrections were not inspected or consumed;
GroupAI runtime fidelity and original writer closure remain excluded.

## Reconciliation with master

Branch tip `afedb39da8e1b1230c9f05df245889e691808f62` merged
`origin/master` `da0a17448c6d5cc24a643b7b5ebf4189218a9af1` (two merge bases,
`2fae001a02496f6360808ba155514185575b86db` and published1110
`1fdf689586edcfcb8b015fdec5f7c55566f831ad`, the latter already contained on
this branch). Conflicts: `pkg/game/itemweight1109_release_test.go`,
`pkg/game/originaldead_release_test.go`, `pkg/game/originalground_release_test.go`,
`pkg/game/save.go`, `pkg/game/save_test.go`, `pkg/sim/world_test.go`. Every
resolution kept both sides' behaviour; the two list-typed conflicts
(`worldMethods`/`worldWriters` in `world_test.go`) took the union. `docs/DIVERGENCES.md`
and `pkg/game/originalcity_sales.go` auto-merged with no conflict;
`originalcity_sales.go` is byte-identical to master, so master's 1110 fix
(`inventoryInstance` blends from the positional baseline item, guarded by
`start < len(baseline)`, replacing the non-unique `Class+DefinitionRow` keyed
search) is intact and `originalcity_sales_collision_test.go` passes unchanged.
The research gitlink moved forward only, `659c3f69d4cb579006d4b584e0246cf309a40fc7`
to `682184eb8e2571ee9b23d2ee614b9f9a459a63d3`; the submodule working tree is
checked out to that same commit.

Two SHA256 pins in `save_test.go` move because the merged Snapshot gob
descriptor and native binary form now include both 1110's and 1111's additive
fields together for the first time: `currentReleasedSaveFixtureSHA256`
(`18ade1bea5da9eadd3f12d2bc0f4c32f111a73caaf2bd4983a14f7a48bac4fa3`) and the
current-descriptor form72 re-encode hash inside
`TestReleasedEnvelopeAtTheCurrentSimulationFormFixture`
(`591285de9c108257d559bb433e609dbc5f9eca77d1900e63734f6edd59f111e1`). Neither
was derived by trusting the value the failing test printed: the same test's
separate raw pre-sales-form72 byte check and full decode round trip pass
unchanged, and the fixture's own `w.Hash()` literal (`0x39ef548000fb4d0a`) is
unmodified and still correct, because master's `formatVersion` stays 75 on
that fixture's own history and `releasedSaveFixtureSnapshot`'s World carries
zero entities, so no story between 1112 and 1124 touches its bytes. The
form75-to-76 World-binary change itself is independently proved by
`pkg/sim/sourcebinding1111_test.go`'s `TestSourceBinding1111PredecessorAndBoundedRefusal`,
which feeds the frozen historical fixture `internal/savefixture/form75.go`
(SHA256 `c6a07b23647192fd225d9d30b3956987dbed56ff6be0ebd37919cb554578e967`)
through the production `sim.UpgradeSaveForm`, a path that does not touch
either moved constant above.

`pkg/game/originalprofile_combat_test.go` is new on master, added by 1110's
own reconciliation before story 1111 existed on master; this merge is the
first time it runs against `admitOriginalActorRegistry`. All 12 subtests
failed post-merge: a player-issued `KindAttack` order reached `AttackCharging`
then was silently cancelled a few ticks later, every case. A temporary stack
trace (reverted; `git diff` against `pkg/sim/combat.go` is empty) traced the
cancellation to `(*World).decide` (`pkg/sim/engage.go:597-602`): a group whose
owner is not `SelfSlot` and scores no candidate has its attack released every
tick. The fixture's attacker landed at raw Owner 2, not `SelfSlot` (1),
because `poolFixtureSave`'s own default leaves an empty Player0 ahead of the
one populated Player, so the sole real Player becomes Slot 2. This is a stale
fixture, not a production regression: `actorregistry1111_test.go` already
builds its actors under a single, non-dummy Player (Slot 1) with no
leading empty entry, and `actorregistry1111_alias_test.go`'s
`TestActorRegistry1111AliasFinalPlayerOwnerAndCityProjection` independently
asserts `e.Owner == 2` for a real `StartingHero` as intentional, tested,
already-landed behaviour for a later-Player-alias scenario — so normalising
`admitOriginalActorRegistry`'s owner assignment would contradict already-passing,
reviewed production behaviour instead of fixing an unaware test. The fix
builds the combat test's SAV payload with `poolFixtureBody` directly so the
attacker's Player is Slot 1 (`SelfSlot`); no production file changed.

A standalone `cmd/missionrun` built from this candidate, run against the EN
root with `-mission 10 -trace -ticks 1` and `-mission 20 -trace -ticks 1`,
prints 0 `UNSUPPORTED` lines for each mission. `pipeline/milestone-baseline.txt`
records no `cannot run` row for m10 or m20 on either root, so this candidate
carries the census forward unchanged; this reconciliation adds no script
opcode and repairs no compiled trigger, matching what 1111's own landing
already claimed.

Final candidate `go test -trimpath -count=1 ./...`: 49 packages ok, 0 FAIL,
0 build failures. `gofmt -l .` empty. `go build ./...` clean. `go vet ./...`
reports the same 11 pre-existing unkeyed-`sim.BookSpell`-literal warnings
master already carries, across 5 files this story and this merge do not
touch. `scripts/check-no-game-assets.sh`: clean. `scripts/check-claim-citations.sh`:
1566 distinct citations resolve against 1890 claims and 298 experiments under
1008 prefixes. `git diff --check`: clean. No Co-Authored-By or other trailer
on this work.

This merge is the first time 1110's and 1111's independently release-tested
changes run together. Both landed individually against the full
`check-release-tests.sh` (1110: EN+RU 159/159) and the full
`check-scenarios.sh` (1110: EN 50/50). Given the one defect this merge did
find was an interaction only visible once both sides combined, the seat
should run the full paired EN/RU `check-release-tests.sh` invocation and the
full `check-scenarios.sh` set on the merge commit, not a narrowed subset.
