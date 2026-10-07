# Imported pack loot through town SAVE

## Result

An imported same-roster Human party can retain supported acquired pack items
through mission return, ordinary town SAV, cold LOAD, the next mission item
action, and a second SAV. Current counts, ordered ownership, distinct child
objects, container bookkeeping, and Human load must survive each boundary.

## Scope and authority

The frontend captures the validated current Item/Effect/Spell graph before
CarryRoster clears object handles. The join is source Human identity to stable
PartyID to the actual mission entity. Mission AGS preserves immutable city
provenance and the native current registry. Returned graphs remain separate
from the immutable import baseline. The existing city writer grafts them into
a cloned semantic document, prunes unreachable objects and remints identities.
Cold mission entry imports that city's own graph before tick zero.

Actual acquisition constructs current operands before transactional insertion.
The shared simulation constructor preserves each Effect occurrence and owned
Spell as a distinct node. Explicit installed constructors supply concrete
equipment retained in PACK. Native Token/F47 defaults remain DIV-1214 policy.
The bridge uses the shared typed item/child builders. It never fabricates a
World for a city or borrows a donor city graph.

Same roster, worn equipment, modifier, book membership, and normalization
guards remain. Newly constructed equipment retained in PACK is included;
wearing it is outside this result. Unsupported alias, effect, lifetime,
overflow, or roster states keep native AGS and converter refusal without output.

Authority is ITEM-WHOLE-128, ITEM-EFFOBJ-072, ITEM-SAVE-014,
ITEM-GROUNDMOVE-130, ITEM-EFFSPLIT-074, ITEM-EFFGRAM-070, and ITEM-STACK-003.
Explicit native constructor defaults do not assert original Token allocation.
Original runtime acceptance is separate; original games are never run here.

## Proof

`TestReleaseImportedLoot1172VictoryColdNextTransfer` uses the read-only owner
source `2026-08-15/game0010.sav`, SHA-256
`89cfca4c14e2b0bafd1fe28911badf246e213d83398874699947008739e8c5d4`.
The installed M30 Sack at (32,31) contains armor 63109, price 500 and weight 4.
Teleport supplies bounded setup; production TakeSack performs the acquisition.
The ordinary mission AGS SAVE/LOAD preserves the full canonical World hash.
The real installed victory consumes quest item 0x0e1e and retains the armor.

The independent first-SAV oracle reads exact ordered Human ownership, counts,
all retained Token/Item/concrete equipment fields, child edges and values,
insertion index, pack accumulator and current Human load. Identity reminting
uses an explicit bijection; gameplay fields are never normalized for comparison.
A fresh child process receives only the new SAV, expected semantic values and
installed assets; its source-corpus input is removed. Cold LOAD opens M40 and
production MoveCarried transfers the acquired armor to the other member. A
second mission AGS restore precedes a controlled M40 return and second SAV.
The second return does not claim an M40 victory witness.

`TestReleaseImportedLoot1172ControlledRetainedGrant` separately retains the real
M30 quest grant through a controlled return. Actual victory's consumption is
asserted by the first test. EN and RU each pass these five new release tests,
the two retained story1169 tests and the independent game0016 item-operation
regression. Both installs consume the same owner recording; they are not two
independent original recordings.

Synthetic codec and simulation controls cover count 3, signed weight -7,
nonzero retained Token/F45/F46/F47/F48 fields, two distinct equal-valued Effects,
owned Spell operands, graph pruning on the next SAV and failed-insertion
rollback. Single-cause losses include omitted items, collapsed children, stale
count/load/index, swapped equal-code owners, foreign/orphan handles, stale
Spell edges, unsupported Effect lifetime and absent return provenance.
An unclosed Token reference stays in AGS and conversion refuses with no output.
Combined city-object and source-loadout expansion limits are checked through
the cold reader before the writer emits bytes; individual pack validity alone
cannot admit output beyond that reader's bounds.

Focused `pkg/formats/sav`, `pkg/sim`, `pkg/game` and gated-population checks pass.
Current gob descriptor expectations change for the new optional graph field;
the frozen historic envelopes are untouched and still decode. The seat runs
the final broad gates after the sole review and correction.

The F1 correction keeps actor references live across acquired-item insertion:
only the registry and destination pack commit, with failed insertion still
atomic. Headless terminalization writes its final decay marker to the current
actor after the outer loot transaction. The independent review probe now
passes. `TestAcquiredSavedItem1172TerminalMixedHoldings` covers ordinary damage
and decay, headless death, retained load, suppressed loot, distinct children,
native cold reload, repeat-call stability and the next pickup's split/merge
retirement. `TestAcquiredSavedItem1172SourceTerminalMixedHoldings` also checks
source-actor container destruction with mixed bound and unbound holdings.

## Open debt

DIV-1215 retains the same-roster and closed-pack admission boundary. Each graph
has at most 4096 objects; the current stack word and carry expansion bounds
remain enforced. Source sales, worn equipment changes, roster changes, book
membership changes, residual modifier changes and unsupported normalization
remain AGS. This result does not establish full World SAV coverage or original
runtime reload acceptance. The full fresh-world story1171 checkpoint is not a
dependency of this result; only its published grammar/import interface is reused.
